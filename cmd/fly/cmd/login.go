/*
Copyright © 2026 FunctionFly
*/
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/functionfly/fly/internal/credentials"
	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with GitHub or Google OAuth",
	Long: `Log in via the browser (like Cursor and other CLIs).

Flow:
1. CLI starts a local callback server and opens your browser to the login page
2. You sign in with GitHub or Google on the site
3. The site redirects back to the CLI with your token
4. Token is saved to ~/.functionfly/credentials.json

Namespace: fx://username/* (e.g., fx://superfly/slugify)`,
	Run: loginRun,
}

func init() {
	rootCmd.AddCommand(loginCmd)

	loginCmd.Flags().StringP("provider", "p", "github", "OAuth provider (github or google)")
	loginCmd.Flags().BoolP("browser", "b", true, "Open browser automatically")
	loginCmd.Flags().Bool("no-browser", false, "Print the login URL instead of opening the browser")
}

// loginRun implements the login command (Cursor-style: browser → site login → callback with token).
func loginRun(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("provider")
	openBrowser, _ := cmd.Flags().GetBool("browser")
	noBrowser, _ := cmd.Flags().GetBool("no-browser")
	if noBrowser {
		openBrowser = false
	}

	if provider != "github" && provider != "google" {
		log.Fatalf("Invalid provider '%s'. Supported: github, google", provider)
	}

	baseURL := os.Getenv("FFLY_API_URL")
	if baseURL == "" {
		baseURL = "https://api.functionfly.com"
	}

	// Local server to receive the redirect with token
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("Could not start callback server: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	authURL, err := getOAuthURL(baseURL, provider, callbackURL)
	if err != nil {
		log.Fatalf("Failed to get login URL: %v", err)
	}

	fmt.Printf("Logging in with %s...\n", provider)
	if openBrowser {
		fmt.Println("Opening browser to sign in...")
		if err := openBrowserTo(authURL); err != nil {
			fmt.Printf("Could not open browser. Open this URL manually:\n%s\n\n", authURL)
		}
	} else {
		fmt.Printf("\nOpen this URL in your browser:\n%s\n\n", authURL)
	}
	fmt.Println("Waiting for you to sign in (Ctrl+C to cancel)...")

	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			errMsg := r.URL.Query().Get("error_description")
			if errMsg == "" {
				errMsg = r.URL.Query().Get("error")
			}
			if errMsg == "" {
				errMsg = "no token received"
			}
			http.Error(w, "Login failed: "+errMsg, http.StatusBadRequest)
			errCh <- fmt.Errorf("login failed: %s", errMsg)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><head><title>Success</title></head><body><h2>✅ You are logged in.</h2><p>You can close this tab and return to the terminal.</p></body></html>`)
		tokenCh <- token
	})
	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var token string
	select {
	case token = <-tokenCh:
	case err := <-errCh:
		_ = server.Shutdown(ctx)
		log.Fatalf("Login failed: %v", err)
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		log.Fatalf("Login timed out after 5 minutes")
	}
	_ = server.Shutdown(context.Background())

	// Fetch user info and save credentials
	creds, err := fetchUserAndBuildCredentials(baseURL, token, provider)
	if err != nil {
		log.Fatalf("Failed to save credentials: %v", err)
	}
	if err := credentials.Save(creds); err != nil {
		log.Fatalf("Failed to save credentials: %v", err)
	}

	username := creds.User.Username
	if username == "" {
		username = "unknown"
	}
	fmt.Printf("\n✅ Logged in as %s\n", creds.User.Email)
	fmt.Printf("   Namespace: fx://%s/*\n", username)
}

// getOAuthURL calls GET /auth/oauth/url?provider=...&redirect_uri=... and returns the URL to open.
func getOAuthURL(baseURL, provider, redirectURI string) (string, error) {
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
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned %d", resp.StatusCode)
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if out.URL == "" {
		return "", fmt.Errorf("API returned empty login URL")
	}
	return out.URL, nil
}

func fetchUserAndBuildCredentials(baseURL, token, provider string) (*credentials.Credentials, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/v1/users/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var user struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&user)

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	return &credentials.Credentials{
		Version: "1.0.0",
		User: credentials.User{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Provider:  provider,
			AvatarURL: user.AvatarURL,
		},
		Token:     token,
		TokenType: "Bearer",
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}, nil
}

func openBrowserTo(u string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name, args = "open", []string{u}
	case "windows":
		name, args = "cmd", []string{"/c", "start", u}
	default:
		name, args = "xdg-open", []string{u}
	}
	return exec.Command(name, args...).Start()
}
