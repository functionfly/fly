package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func NewLoginCmd() *cobra.Command {
	var provider string
	var noBrowser bool
	var dev bool
	var email string
	cmd := &cobra.Command{
		Use:     "login",
		Short:   "Authenticate with FunctionFly",
		Long:    "Authenticate with FunctionFly using OAuth or dev email/password.\n\nIn dev mode (--dev flag required), use email/password against the API.",
		Example: "  ffly login\n  ffly login --provider github\n  ffly login --dev --email admin@functionfly.local",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(provider, noBrowser, dev, email)
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "github", "OAuth provider (github, google)")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Print the auth URL instead of opening a browser")
	cmd.Flags().BoolVar(&dev, "dev", false, "Use email/password login (dev mode). Requires FFLY_API_URL for local API.")
	cmd.Flags().StringVar(&email, "email", "", "Email for dev login (or set FFLY_DEV_EMAIL)")
	return cmd
}

func runLogin(provider string, noBrowser bool, dev bool, emailFlag string) error {
	baseURL := os.Getenv("FFLY_API_URL")
	if baseURL == "" {
		cfg, _ := LoadConfig()
		if cfg != nil && cfg.API.URL != "" {
			baseURL = cfg.API.URL
		}
	}
	if baseURL == "" {
		baseURL = "https://api.functionfly.com"
	}

	// Dev mode: only activated by explicit --dev flag or FFLY_DEV_LOGIN=1
	useDev := dev || os.Getenv("FFLY_DEV_LOGIN") == "1"
	if useDev {
		return runDevLogin(baseURL, emailFlag)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("could not start callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// Get OAuth URL from API (includes our redirect_uri so callback sends token to this CLI)
	authURL, err := getOAuthURLFromAPI(baseURL, provider, callbackURL)
	if err != nil {
		return fmt.Errorf("get OAuth URL: %w", err)
	}

	fmt.Printf("🔐 Authenticating with %s...\n", provider)
	if noBrowser {
		fmt.Printf("\nOpen this URL in your browser:\n%s\n\n", authURL)
	} else {
		fmt.Printf("Opening browser...\n")
		if err := openBrowser(authURL); err != nil {
			fmt.Printf("Could not open browser automatically.\nOpen this URL manually:\n%s\n\n", authURL)
		}
	}
	fmt.Printf("Waiting for authentication (Ctrl+C to cancel)...\n")
	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no token received"
			}
			http.Error(w, "Authentication failed: "+errMsg, http.StatusBadRequest)
			errCh <- fmt.Errorf("authentication failed: %s", errMsg)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body><h2>✅ Authentication successful!</h2><p>You can close this tab.</p></body></html>`)
		tokenCh <- token
	})
	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var token string
	select {
	case token = <-tokenCh:
	case err := <-errCh:
		server.Close()
		return err
	case <-ctx.Done():
		server.Close()
		return fmt.Errorf("authentication timed out after 5 minutes")
	}
	server.Close()
	client := NewAPIClientWithToken(token)
	var userResp struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		Provider  string `json:"provider"`
		AvatarURL string `json:"avatar_url"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := client.Get("/v1/users/me", &userResp); err != nil {
		fmt.Printf("⚠️  Could not fetch user info: %v\n", err)
	}
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if userResp.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, userResp.ExpiresAt); err == nil {
			expiresAt = t
		}
	}
	creds := &Credentials{
		Version:   "1.0.0",
		User:      UserInfo{ID: userResp.ID, Username: userResp.Username, Email: userResp.Email, Provider: provider, AvatarURL: userResp.AvatarURL},
		Token:     token,
		TokenType: "Bearer",
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	if err := SaveCredentials(creds); err != nil {
		return fmt.Errorf("could not save credentials: %w", err)
	}
	username := userResp.Username
	if username == "" {
		username = "unknown"
	}
	fmt.Printf("\n✅ Logged in as %s\n", username)
	if userResp.Email != "" {
		fmt.Printf("   Email: %s\n", userResp.Email)
	}
	fmt.Printf("   Provider: %s\n", provider)
	fmt.Printf("\nYour namespace: fx://%s/*\n", username)
	return nil
}

// getOAuthURLFromAPI calls GET /auth/oauth/url?provider=...&redirect_uri=... and returns the URL to open in the browser.
func getOAuthURLFromAPI(baseURL, provider, redirectURI string) (string, error) {
	u := baseURL + "/auth/oauth/url?provider=" + url.QueryEscape(provider)
	if redirectURI != "" {
		u += "&redirect_uri=" + url.QueryEscape(redirectURI)
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned %d", resp.StatusCode)
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.URL == "" {
		return "", fmt.Errorf("API returned empty OAuth URL")
	}
	return out.URL, nil
}

func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}

// runDevLogin uses POST /v1/auth/login with email/password (for local/dev API).
// The password is always prompted interactively to avoid exposure in process listings or shell history.
func runDevLogin(baseURL, emailFlag string) error {
	email := emailFlag
	if email == "" {
		email = os.Getenv("FFLY_DEV_EMAIL")
	}
	password := os.Getenv("FFLY_DEV_PASSWORD")
	if IsInteractive() && email == "" {
		email = Prompt("Email", "admin@functionfly.local")
	}
	if IsInteractive() && password == "" {
		password = Prompt("Password", "")
	}
	if email == "" || password == "" {
		return fmt.Errorf("email and password required for dev login (use --email and FFLY_DEV_PASSWORD)")
	}

	loginPath := "/v1/auth/login"
	if contains(baseURL, "localhost") || contains(baseURL, "127.0.0.1") {
		loginPath = "/auth/login"
	}
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req, err := http.NewRequest("POST", baseURL+loginPath, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	var loginResp struct {
		Token string `json:"token"`
		User  *struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Name     string `json:"name"`
			Email    string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(bodyBytes, &loginResp); err != nil {
		return fmt.Errorf("invalid login response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var errMsg struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(bodyBytes, &errMsg)
		if errMsg.Message != "" {
			return fmt.Errorf("login failed: %s", errMsg.Message)
		}
		return fmt.Errorf("login failed: HTTP %d", resp.StatusCode)
	}
	if loginResp.Token == "" {
		return fmt.Errorf("login response missing token")
	}

	username := "unknown"
	userEmail := email
	userID := ""
	if loginResp.User != nil {
		userID = loginResp.User.ID
		if loginResp.User.Username != "" {
			username = loginResp.User.Username
		}
		if loginResp.User.Email != "" {
			userEmail = loginResp.User.Email
		}
	}

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	creds := &Credentials{
		Version:   "1.0.0",
		User:      UserInfo{ID: userID, Username: username, Email: userEmail, Provider: "dev", AvatarURL: ""},
		Token:     loginResp.Token,
		TokenType: "Bearer",
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	if err := SaveCredentials(creds); err != nil {
		return fmt.Errorf("could not save credentials: %w", err)
	}
	fmt.Printf("\n✅ Logged in as %s (dev)\n", username)
	fmt.Printf("   Email: %s\n", userEmail)
	fmt.Printf("\nYour namespace: fx://%s/*\n", username)
	return nil
}
