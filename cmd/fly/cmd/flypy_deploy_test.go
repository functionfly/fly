/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/functionfly/fly/internal/cli"
	artifactPkg "github.com/functionfly/fly/internal/flypy/artifact"
)

func TestValidateArtifactSignatureVerification(t *testing.T) {
	// Generate a key pair for testing
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create a test artifact
	testArtifact := &artifactPkg.Artifact{
		WasmModule:    []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}, // Minimal valid Wasm
		Manifest:      &artifactPkg.Manifest{Name: "test", Version: "1.0.0", Runtime: "wasm32", Deterministic: true},
		CapabilityMap: &artifactPkg.CapabilityMap{},
	}

	// Compute the correct determinism hash
	expectedHash, err := computeArtifactContentHash(testArtifact)
	if err != nil {
		t.Fatalf("Failed to compute expected hash: %v", err)
	}
	testArtifact.DeterminismHash = expectedHash

	// Sign the determinism hash
	signature := ed25519.Sign(privateKey, []byte(expectedHash))
	testArtifact.Signature = signature

	// Test successful verification
	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKey)
	err = validateArtifact(testArtifact, publicKeyB64)
	if err != nil {
		t.Errorf("Expected signature verification to succeed, got error: %v", err)
	}

	// Test verification with wrong public key
	wrongPublicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	wrongPublicKeyB64 := base64.StdEncoding.EncodeToString(wrongPublicKey)
	err = validateArtifact(testArtifact, wrongPublicKeyB64)
	if err == nil {
		t.Error("Expected signature verification to fail with wrong public key")
	}

	// Test verification with invalid public key base64
	err = validateArtifact(testArtifact, "invalid-base64")
	if err == nil {
		t.Error("Expected error for invalid base64 public key")
	}

	// Test artifact without signature (should succeed)
	artifactNoSig := &artifactPkg.Artifact{
		WasmModule:    []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00},
		Manifest:      &artifactPkg.Manifest{Name: "test", Version: "1.0.0", Runtime: "wasm32", Deterministic: true},
		CapabilityMap: &artifactPkg.CapabilityMap{},
	}
	expectedHashNoSig, _ := computeArtifactContentHash(artifactNoSig)
	artifactNoSig.DeterminismHash = expectedHashNoSig
	artifactNoSig.Signature = nil // No signature

	err = validateArtifact(artifactNoSig, "")
	if err != nil {
		t.Errorf("Expected validation to succeed for artifact without signature, got error: %v", err)
	}

	// Test artifact with signature but no public key provided
	err = validateArtifact(testArtifact, "")
	if err == nil {
		t.Error("Expected error when signature present but no public key provided")
	}
}

func TestValidateArtifactBasicValidation(t *testing.T) {
	// Test missing manifest
	artifact := &artifactPkg.Artifact{
		WasmModule:    []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00},
		CapabilityMap: &artifactPkg.CapabilityMap{},
	}
	err := validateArtifact(artifact, "")
	if err == nil || err.Error() != "manifest is missing" {
		t.Errorf("Expected 'manifest is missing' error, got: %v", err)
	}

	// Test invalid Wasm
	artifact = &artifactPkg.Artifact{
		WasmModule:    []byte{0x01, 0x02, 0x03},
		Manifest:      &artifactPkg.Manifest{Name: "test", Version: "1.0.0", Runtime: "wasm32"},
		CapabilityMap: &artifactPkg.CapabilityMap{},
	}
	err = validateArtifact(artifact, "")
	if err == nil || !contains(err.Error(), "invalid Wasm") {
		t.Errorf("Expected Wasm validation error, got: %v", err)
	}
}

func TestGetPublicKey(t *testing.T) {
	// Save original flag value
	originalFlag := flypyDeployFlags.publicKey
	defer func() { flypyDeployFlags.publicKey = originalFlag }()

	// Test flag takes priority
	flypyDeployFlags.publicKey = "flag-key"
	result := getPublicKey()
	if result != "flag-key" {
		t.Errorf("Expected flag value 'flag-key', got '%s'", result)
	}

	// Reset flag
	flypyDeployFlags.publicKey = ""

	// Test environment variable takes precedence over empty flag
	originalEnv := os.Getenv("FFLY_PUBLIC_KEY")
	defer func() { os.Setenv("FFLY_PUBLIC_KEY", originalEnv) }()

	os.Setenv("FFLY_PUBLIC_KEY", "env-key")
	result = getPublicKey()
	if result != "env-key" {
		t.Errorf("Expected environment variable value 'env-key', got '%s'", result)
	}

	// Reset environment variable and test empty fallback
	os.Unsetenv("FFLY_PUBLIC_KEY")
	result = getPublicKey()
	if result != "" {
		t.Errorf("Expected empty string when no key configured, got '%s'", result)
	}
}

func TestDeployToRegistryAuthentication(t *testing.T) {
	// Test data
	req := &cli.DeployRequest{
		Provider: "test",
		Region:   "test-region",
		Artifact: "test-artifact",
	}

	expectedResponse := &cli.DeployResponse{
		DeploymentID: "test-deployment-id",
		Status:       "deployed",
		Message:      "Deployment successful",
	}

	t.Run("successful authentication with valid token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify authentication header
			auth := r.Header.Get("Authorization")
			if auth != "Bearer valid-token" {
				t.Errorf("Expected Authorization header 'Bearer valid-token', got '%s'", auth)
			}

			// Verify content type
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
			}

			// Verify method and path
			if r.Method != "POST" {
				t.Errorf("Expected POST method, got %s", r.Method)
			}
			if r.URL.Path != "/api/v1/functions/deploy" {
				t.Errorf("Expected path '/api/v1/functions/deploy', got '%s'", r.URL.Path)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(expectedResponse)
		}))
		defer server.Close()

		config := &cli.Config{Token: "valid-token"}
		resp, err := deployToRegistry(server.URL, req, config)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if resp == nil {
			t.Fatal("Expected response, got nil")
		}
		if resp.DeploymentID != expectedResponse.DeploymentID {
			t.Errorf("Expected DeploymentID '%s', got '%s'", expectedResponse.DeploymentID, resp.DeploymentID)
		}
	})

	t.Run("authentication failure with invalid token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "Bearer invalid-token" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "Invalid token"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := &cli.Config{Token: "invalid-token"}
		_, err := deployToRegistry(server.URL, req, config)

		if err == nil {
			t.Error("Expected authentication error, got nil")
		}
	})

	t.Run("no authentication when config is nil", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Should not have Authorization header
			auth := r.Header.Get("Authorization")
			if auth != "" {
				t.Errorf("Expected no Authorization header when config is nil, got '%s'", auth)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(expectedResponse)
		}))
		defer server.Close()

		resp, err := deployToRegistry(server.URL, req, nil)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if resp == nil {
			t.Fatal("Expected response, got nil")
		}
	})

	t.Run("no authentication when token is empty", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Should not have Authorization header
			auth := r.Header.Get("Authorization")
			if auth != "" {
				t.Errorf("Expected no Authorization header when token is empty, got '%s'", auth)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(expectedResponse)
		}))
		defer server.Close()

		config := &cli.Config{Token: ""}
		resp, err := deployToRegistry(server.URL, req, config)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if resp == nil {
			t.Fatal("Expected response, got nil")
		}
	})
}