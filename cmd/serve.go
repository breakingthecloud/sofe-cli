package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/breakingthecloud/sofe-cli/internal/output"
	"github.com/spf13/cobra"
)

var servePort string

func pidFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".sofe", "server.pid")
}

func waitForServer(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url + "/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func startServer(port string) (int, error) {
	// Try multiple ways to start sofe-server
	var cmd *exec.Cmd

	// Option 1: sofe-server entry point (pip install sofe-server)
	if path, err := exec.LookPath("sofe-server"); err == nil {
		cmd = exec.Command(path)
		cmd.Env = append(os.Environ(), "UVICORN_PORT="+port)
	} else {
		// Option 2: python3 -m uvicorn (works if sofe-server is pip-installed)
		python := "python3"
		if p, err := exec.LookPath("python"); err == nil {
			python = p
		}
		cmd = exec.Command(python, "-m", "uvicorn", "sofe_server.app:app",
			"--host", "0.0.0.0", "--port", port)
	}

	cmd.Stdout = nil
	cmd.Stderr = nil
	// Detach from parent so it survives
	cmd.SysProcAttr = nil

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start sofe-server: %w\n  Install with: pip install sofe-server", err)
	}

	pid := cmd.Process.Pid
	// Release so it doesn't become zombie
	cmd.Process.Release()

	// Save PID
	os.MkdirAll(filepath.Dir(pidFile()), 0755)
	os.WriteFile(pidFile(), []byte(strconv.Itoa(pid)), 0644)

	return pid, nil
}

func stopServer() error {
	data, err := os.ReadFile(pidFile())
	if err != nil {
		return fmt.Errorf("no server PID found (not running?)")
	}

	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found", pid)
	}

	proc.Kill()
	os.Remove(pidFile())
	return nil
}

func isServerRunning(url string) bool {
	resp, err := http.Get(url + "/health")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

// AutoServe starts server if not running, returns true if it started one (caller should stop it)
func AutoServe(apiURL string) bool {
	if isServerRunning(apiURL) {
		return false
	}

	port := "8080"
	if strings.Contains(apiURL, ":") {
		parts := strings.Split(apiURL, ":")
		port = parts[len(parts)-1]
	}

	fmt.Printf("🔄 Server not running, starting on :%s...\n", port)
	_, err := startServer(port)
	if err != nil {
		output.PrintError(err.Error())
		return false
	}

	if !waitForServer(apiURL, 15*time.Second) {
		output.PrintError("Server failed to start within 15s")
		stopServer()
		return false
	}

	fmt.Println("✅ Server started automatically")
	return true
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start sofe-server in background",
	Long:  "Starts the SOFE API server (Python FastAPI) as a background process.",
	RunE: func(cmd *cobra.Command, args []string) error {
		port := servePort
		if port == "" {
			port = "8080"
		}

		url := fmt.Sprintf("http://localhost:%s", port)
		if isServerRunning(url) {
			output.PrintSuccess(fmt.Sprintf("Server already running on :%s", port))
			return nil
		}

		pid, err := startServer(port)
		if err != nil {
			return err
		}

		if !waitForServer(url, 15*time.Second) {
			output.PrintError("Server failed to start")
			return fmt.Errorf("timeout waiting for server")
		}

		output.PrintSuccess(fmt.Sprintf("SOFE Server running on :%s (PID %d)", port, pid))
		fmt.Printf("  Stop with: sofe serve stop\n")
		return nil
	},
}

var serveStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the background sofe-server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := stopServer(); err != nil {
			output.PrintError(err.Error())
			return err
		}
		output.PrintSuccess("Server stopped")
		return nil
	},
}

func init() {
	serveCmd.Flags().StringVarP(&servePort, "port", "", "8080", "Port to run server on")
	serveCmd.AddCommand(serveStopCmd)
	rootCmd.AddCommand(serveCmd)
}
