/*
Copyright © 2026 FunctionFly
*/
package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/functionfly/fly/internal/bundler"
	"github.com/functionfly/fly/internal/manifest"
	"github.com/functionfly/fly/internal/watcher"
	"github.com/spf13/cobra"
)

// devCmd represents the dev command
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run local execution environment identical to production",
	Long: `Runs a local development server that mirrors production behavior.

The local runtime:
- Executes functions in isolated Wasm sandbox
- Applies resource limits (memory, CPU, timeout)
- Supports deterministic caching
- Uses instance pooling for fast cold starts

Examples:
  fly dev
  fly dev --port=3000
  fly dev --watch`,
	Run: devRun,
}

var devFlags struct {
	port    int
	watch   bool
	runtime string
}

// Default port for local dev server
const defaultDevPort = 8787

func init() {
	rootCmd.AddCommand(devCmd)

	// Local flags
	devCmd.Flags().IntVarP(&devFlags.port, "port", "p", defaultDevPort, "Port to listen on")
	devCmd.Flags().BoolVarP(&devFlags.watch, "watch", "w", false, "Watch for file changes and reload")
	devCmd.Flags().StringVar(&devFlags.runtime, "runtime", "nodejs", "Runtime to use (nodejs, python)")
}

// devRun implements the dev command
func devRun(cmd *cobra.Command, args []string) {
	// 1. Load and validate manifest (auto-detects .jsonc or .json)
	m, err := manifest.Load("")
	if err != nil {
		log.Fatalf("Failed to load manifest: %v", err)
	}

	if err := m.Validate(); err != nil {
		log.Fatalf("Manifest validation failed: %v", err)
	}

	fmt.Printf("🚀 Starting local FunctionFly runtime...\n\n")
	fmt.Printf("   Function: %s\n", m.Name)
	fmt.Printf("   Version: %s\n", m.Version)
	fmt.Printf("   Runtime: %s\n", m.Runtime)
	fmt.Printf("   Port:    %d\n", devFlags.port)

	if devFlags.watch {
		fmt.Printf("   Watch:   enabled\n")
	}
	fmt.Printf("\n")

	// Channel for restart signals
	restartCh := make(chan struct{}, 1)
	var restartMutex sync.Mutex

	// Function to start/restart the runtime
	startRuntime := func() (*exec.Cmd, *os.File) {
		restartMutex.Lock()
		defer restartMutex.Unlock()

		// 2. Bundle function to Wasm for local runtime
		fmt.Printf("   Bundling function to Wasm...\n")
		wasmBundle, err := bundler.BundleForWasmRuntimeWithWorkingDirectory(m, "")
		if err != nil {
			log.Printf("Failed to bundle to Wasm: %v", err)
			return nil, nil
		}

		// Save the Wasm bundle to a temporary file
		wasmFile, err := os.CreateTemp("", "functionfly-*.wasm")
		if err != nil {
			log.Printf("Failed to create Wasm temp file: %v", err)
			return nil, nil
		}

		if _, err := wasmFile.Write(wasmBundle); err != nil {
			log.Printf("Failed to write Wasm bundle: %v", err)
			wasmFile.Close()
			os.Remove(wasmFile.Name())
			return nil, nil
		}
		wasmFile.Close()

		// 3. Find or build the Rust runtime
		runtimePath, err := findLocalRuntime()
		if err != nil {
			log.Printf("⚠️  Local runtime not found at: %s", runtimePath)
			log.Printf("   Building runtime...")

			if err := buildLocalRuntime(); err != nil {
				log.Printf("Failed to build local runtime: %v", err)
				os.Remove(wasmFile.Name())
				return nil, nil
			}
		}

		// 4. Start the local runtime as a sidecar process
		proc, err := startLocalRuntime(runtimePath, wasmFile.Name(), m, devFlags.port)
		if err != nil {
			log.Printf("Failed to start local runtime: %v", err)
			os.Remove(wasmFile.Name())
			return nil, nil
		}

		return proc, wasmFile
	}

	// Function to stop the runtime with proper cleanup
	stopRuntime := func(proc *exec.Cmd, wasmFile *os.File) {
		if proc != nil && proc.Process != nil {
			// Send SIGTERM first for graceful shutdown
			if err := proc.Process.Signal(syscall.SIGTERM); err != nil {
				// If SIGTERM fails, try SIGKILL
				proc.Process.Kill()
			}

			// Wait for process to exit with timeout
			done := make(chan error, 1)
			go func() {
				done <- proc.Wait()
			}()

			select {
			case <-done:
				// Process exited normally
			case <-time.After(5 * time.Second):
				// Timeout, force kill
				proc.Process.Kill()
				<-done // Wait for kill to complete
			}
		}

		// Clean up temporary Wasm file
		if wasmFile != nil {
			if err := os.Remove(wasmFile.Name()); err != nil && !os.IsNotExist(err) {
				log.Printf("Warning: Failed to clean up temporary file %s: %v", wasmFile.Name(), err)
			}
		}
	}

	// Initial start
	proc, wasmFile := startRuntime()
	if proc == nil {
		log.Fatalf("Failed to start initial runtime")
	}

	// 5. Wait for the server to be ready
	fmt.Printf("   Waiting for server to be ready...")

	if err := waitForServer(devFlags.port, 10*time.Second); err != nil {
		log.Printf(" failed\n")
		log.Fatalf("Server failed to start: %v", err)
	}

	fmt.Printf(" ready\n\n")

	// 6. Print startup message
	fmt.Printf("✅ Local FunctionFly runtime started!\n")
	fmt.Printf("   URL: http://localhost:%d\n", devFlags.port)
	fmt.Printf("\n")
	fmt.Printf("   Test with:\n")
	fmt.Printf("   curl -X POST http://localhost:%d -d 'Hello World'\n", devFlags.port)
	fmt.Printf("\n")

	if devFlags.watch {
		fmt.Printf("🔍 Watching for file changes...\n")

		// Set up file watcher
		fileWatcher, err := watcher.NewFileWatcher(func(changedFile string) {
			fmt.Printf("\n📝 File changed: %s\n", changedFile)
			fmt.Printf("🔄 Restarting runtime...\n")

			// Signal restart
			select {
			case restartCh <- struct{}{}:
			default:
				// Channel already has a pending restart
			}
		})

		if err != nil {
			log.Printf("Warning: Failed to create file watcher: %v", err)
		} else {
			// Identify files to watch
			watchFiles := watcher.IdentifyWatchableFiles(m.Runtime)
			if err := fileWatcher.WatchFiles(watchFiles); err != nil {
				log.Printf("Warning: Failed to watch some files: %v", err)
			}

			watchedFiles := fileWatcher.GetWatchedFiles()
			if len(watchedFiles) > 0 {
				fmt.Printf("   Watching %d files:\n", len(watchedFiles))
				for _, file := range watchedFiles {
					fmt.Printf("   - %s\n", file)
				}
				fmt.Printf("\n")
			}

			// Start watching
			fileWatcher.Start()

			// Handle file change restarts
			go func() {
				for range restartCh {
					// Stop current runtime
					stopRuntime(proc, wasmFile)

					// Brief pause to ensure cleanup
					time.Sleep(100 * time.Millisecond)

					// Restart
					newProc, newWasmFile := startRuntime()
					if newProc == nil {
						fmt.Printf("❌ Failed to restart runtime\n")
						continue
					}

					proc, wasmFile = newProc, newWasmFile

					// Wait for new server to be ready
					if err := waitForServer(devFlags.port, 10*time.Second); err != nil {
						fmt.Printf("❌ Server failed to restart: %v\n", err)
						continue
					}

					fmt.Printf("✅ Runtime restarted successfully!\n")
					fmt.Printf("   URL: http://localhost:%d\n\n", devFlags.port)
				}
			}()

			defer fileWatcher.Stop()
		}
	} else {
		fmt.Printf("   Press Ctrl+C to stop\n")
	}

	// 7. Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Printf("\n\n👋 Shutting down local runtime...\n")
		stopRuntime(proc, wasmFile)
		os.Exit(0)
	}()

	// 8. Block until process exits
	proc.Wait()
}

// findLocalRuntime finds the local runtime binary, building it if necessary
func findLocalRuntime() (string, error) {
	// Check current directory first (for manually placed binaries)
	localPath := filepath.Join(".", "functionfly-local")
	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	// Check GOPATH/bin (for installed binaries)
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	gopathBin := filepath.Join(gopath, "bin", "functionfly-local")
	if _, err := os.Stat(gopathBin); err == nil {
		return gopathBin, nil
	}

	// Check PATH (for globally installed binaries)
	pathDirs := filepath.SplitList(os.Getenv("PATH"))
	for _, dir := range pathDirs {
		binPath := filepath.Join(dir, "functionfly-local")
		if _, err := os.Stat(binPath); err == nil {
			return binPath, nil
		}
	}

	// Check if we have a local Rust project and try to build it
	runtimeDir := filepath.Join(".", "runtimes", "local")
	if _, err := os.Stat(runtimeDir); err == nil {
		// Check for built release binary
		releasePath := filepath.Join(runtimeDir, "target", "release", "functionfly-local")
		if _, err := os.Stat(releasePath); err == nil {
			return releasePath, nil
		}

		// Check for built debug binary
		debugPath := filepath.Join(runtimeDir, "target", "debug", "functionfly-local")
		if _, err := os.Stat(debugPath); err == nil {
			return debugPath, nil
		}

		// No built binary found, try to build it
		fmt.Printf("   Local runtime not found, building from source...\n")
		if err := buildLocalRuntime(); err != nil {
			return "", fmt.Errorf("failed to build local runtime: %w", err)
		}

		// Check again after building
		if _, err := os.Stat(releasePath); err == nil {
			return releasePath, nil
		}
		if _, err := os.Stat(debugPath); err == nil {
			return debugPath, nil
		}

		return "", fmt.Errorf("runtime built successfully but binary not found at expected location")
	}

	return "", fmt.Errorf("local runtime not found in any location. Expected locations:\n"+
		"  - ./functionfly-local\n"+
		"  - $GOPATH/bin/functionfly-local\n"+
		"  - $PATH/functionfly-local\n"+
		"  - ./runtimes/local/target/release/functionfly-local\n"+
		"  - ./runtimes/local/target/debug/functionfly-local\n"+
		"To build the runtime, ensure Rust is installed and run: cargo build --release in ./runtimes/local/")
}

// buildLocalRuntime builds the Rust local runtime with better error handling
func buildLocalRuntime() error {
	runtimeDir := filepath.Join(".", "runtimes", "local")

	if _, err := os.Stat(runtimeDir); os.IsNotExist(err) {
		return fmt.Errorf("runtime source directory not found at %s\n\n"+
			"Make sure you're in the functionfly project root directory.\n"+
			"Expected structure: ./runtimes/local/Cargo.toml", runtimeDir)
	}

	// Check if Cargo.toml exists
	cargoToml := filepath.Join(runtimeDir, "Cargo.toml")
	if _, err := os.Stat(cargoToml); os.IsNotExist(err) {
		return fmt.Errorf("Cargo.toml not found in %s\n\n"+
			"The runtime source appears to be corrupted or incomplete.\n"+
			"Try: git checkout runtimes/local/", runtimeDir)
	}

	// Check if cargo is installed
	if _, err := exec.LookPath("cargo"); err != nil {
		return fmt.Errorf("cargo (Rust build tool) is not installed\n\n"+
			"To install Rust and Cargo:\n"+
			"  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh\n"+
			"  source ~/.cargo/env\n"+
			"  rustup target add wasm32-wasi")
	}

	fmt.Printf("   Building Rust runtime (this may take a few minutes)...\n")

	// Build the Rust runtime with timeout
	buildCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	buildCmd := exec.CommandContext(buildCtx, "cargo", "build", "--release")
	buildCmd.Dir = runtimeDir

	// Capture output for error analysis
	var stdout, stderr bytes.Buffer
	buildCmd.Stdout = &stdout
	buildCmd.Stderr = &stderr

	err := buildCmd.Run()

	// If build failed, provide helpful error messages
	if err != nil {
		stderrStr := stderr.String()

		// Check for common errors
		if strings.Contains(stderrStr, "wasm32-wasi") {
			return fmt.Errorf("build failed: missing wasm32-wasi target\n\n"+
				"Install the WebAssembly target:\n"+
				"  rustup target add wasm32-wasi")
		}

		if strings.Contains(stderrStr, "linker") {
			return fmt.Errorf("build failed: linker error\n\n"+
				"Install build dependencies:\n"+
				"  Ubuntu/Debian: apt install build-essential\n"+
				"  macOS: xcode-select --install\n"+
				"  CentOS/RHEL: yum groupinstall 'Development Tools'")
		}

		if strings.Contains(stderrStr, "network") || strings.Contains(stderrStr, "crates.io") {
			return fmt.Errorf("build failed: network error downloading dependencies\n\n"+
				"Check your internet connection and try again")
		}

		// Generic error with output
		return fmt.Errorf("build failed: %v\n\nBuild output:\n%s\n\n"+
			"For more help, check the Rust runtime README or try:\n"+
			"  cd %s && cargo build --release --verbose", err, stderrStr, runtimeDir)
	}

	// Verify the binary was created
	binaryPath := filepath.Join(runtimeDir, "target", "release", "functionfly-local")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("build completed but binary not found at %s\n\n"+
			"This might indicate a build configuration issue.\n"+
			"Check the Cargo.toml file in %s", binaryPath, runtimeDir)
	}

	fmt.Printf("   ✅ Runtime built successfully\n")
	return nil
}

// startLocalRuntime starts the Rust runtime as a sidecar process
func startLocalRuntime(runtimePath string, wasmFilePath string, m *manifest.Manifest, port int) (*exec.Cmd, error) {
	var runCmd *exec.Cmd
	var args []string

	// Check if runtimePath is a cargo project (contains Cargo.toml)
	runtimeDir := filepath.Dir(runtimePath)
	cargoTomlPath := filepath.Join(runtimeDir, "Cargo.toml")

	if _, err := os.Stat(cargoTomlPath); err == nil {
		// This is a cargo project, use cargo run
		args = []string{"run", "--release", "--"}
		args = append(args, []string{
			"--port", fmt.Sprintf("%d", port),
			"--function", m.Name,
			"--version", m.Version,
			"--runtime", m.Runtime,
			"--wasm", wasmFilePath,
		}...)

		runCmd = exec.Command("cargo", args...)
		runCmd.Dir = runtimeDir
	} else {
		// This is a built binary, run directly
		args = []string{
			"--port", fmt.Sprintf("%d", port),
			"--function", m.Name,
			"--version", m.Version,
			"--runtime", m.Runtime,
			"--wasm", wasmFilePath,
		}

		runCmd = exec.Command(runtimePath, args...)
	}

	// Add memory limit if set
	if m.MemoryMB != nil && *m.MemoryMB > 0 {
		args = append(args, "--memory-mb", fmt.Sprintf("%d", *m.MemoryMB))
	}

	// Add timeout if set
	if m.TimeoutMS != nil && *m.TimeoutMS > 0 {
		args = append(args, "--timeout-ms", fmt.Sprintf("%d", *m.TimeoutMS))
	}

	// Add deterministic flag
	if m.Deterministic != nil && *m.Deterministic {
		args = append(args, "--deterministic")
	}

	// Set up pipes for stdout/stderr so we can capture output on failure
	stdoutPipe, err := runCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := runCmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := runCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start runtime process: %w\n\nTroubleshooting:\n"+
			"1. Ensure the runtime binary exists and is executable: %s\n"+
			"2. Check if required dependencies are installed (Rust, wasm runtime)\n"+
			"3. Verify the Wasm file was created successfully: %s\n"+
			"4. Try rebuilding: rm -rf runtimes/local/target && cargo build --release",
			err, runtimePath, wasmFilePath)
	}

	// Monitor the process for immediate failure
	go func() {
		// Wait a short time to see if the process exits immediately
		time.Sleep(2 * time.Second)

		// Check if process is still running
		if runCmd.ProcessState != nil && runCmd.ProcessState.Exited() {
			// Process exited early, try to read any error output
			stdoutBuf := make([]byte, 1024)
			stderrBuf := make([]byte, 1024)

			if n, _ := stdoutPipe.Read(stdoutBuf); n > 0 {
				fmt.Printf("Runtime stdout: %s\n", string(stdoutBuf[:n]))
			}
			if n, _ := stderrPipe.Read(stderrBuf); n > 0 {
				fmt.Printf("Runtime stderr: %s\n", string(stderrBuf[:n]))
			}

			fmt.Printf("\n❌ Runtime process exited early (exit code: %d)\n", runCmd.ProcessState.ExitCode())
			fmt.Printf("This usually indicates:\n")
			fmt.Printf("  - Missing dependencies (try: apt install build-essential)\n")
			fmt.Printf("  - Invalid Wasm file (check bundler output)\n")
			fmt.Printf("  - Port %d already in use (try different port with --port)\n", port)
			fmt.Printf("  - Permission issues (ensure binary is executable)\n")
		}
	}()

	return runCmd, nil
}

// waitForServer waits for the server to be ready with better error reporting
func waitForServer(port int, timeout time.Duration) error {
	url := fmt.Sprintf("http://localhost:%d/ready", port)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("server failed to start within %v\n\n"+
				"Troubleshooting:\n"+
				"1. Check if port %d is already in use: netstat -tlnp | grep :%d\n"+
				"2. Verify the runtime binary is working: runtimes/local/target/release/functionfly-local --help\n"+
				"3. Check runtime logs for errors\n"+
				"4. Try a different port: fly dev --port=%d", timeout, port, port, port+1)
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				continue
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				// Log progress every 5 seconds
				if time.Since(startTime) > 5*time.Second && int(time.Since(startTime).Seconds())%5 == 0 {
					fmt.Printf("   Still waiting for server... (%v elapsed)\n", time.Since(startTime).Truncate(time.Second))
				}
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				elapsed := time.Since(startTime)
				fmt.Printf("   Server ready in %v\n", elapsed.Truncate(time.Millisecond))
				return nil
			}

			// If we get a response but not 200, log it
			if resp.StatusCode >= 400 {
				return fmt.Errorf("server responded with error status: %d", resp.StatusCode)
			}
		}
	}
}
