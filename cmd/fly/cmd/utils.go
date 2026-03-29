/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"net"
	"os"
	"time"
)

// getAPIURL returns the API URL from environment or default
func getAPIURL() string {
	// Check for explicit environment variable override
	if url := os.Getenv("FFLY_API_URL"); url != "" {
		return url
	}

	// In development, try localhost first
	if isLocalhostAvailable() {
		return "http://localhost:8080"
	}

	// Default to production
	return "https://api.functionfly.com"
}

// isLocalhostAvailable checks if a local API server is running
func isLocalhostAvailable() bool {
	// Quick check if localhost:8080 is reachable
	// This is a simple connectivity check for development
	conn, err := net.DialTimeout("tcp", "localhost:8080", 100*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}