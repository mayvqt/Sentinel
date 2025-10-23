// Simple welcome binary. The real server runs from cmd/server; this main
// prints a short banner and system/version info to make local runs friendlier.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const (
	appName    = "Sentinel"
	appVersion = "0.1.0"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "(not set)"
	}

	fmt.Println("========================================")
	fmt.Printf(" %s — %s\n", appName, appVersion)
	fmt.Println("----------------------------------------")
	fmt.Printf(" Go version: %s\n", runtime.Version())
	fmt.Printf(" OS/ARCH:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("Press Enter to exit, or Ctrl+C to terminate")

	// Wait for either Enter key or an OS interrupt (Ctrl+C).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan struct{})
	go func() {
		// Wait for a newline on stdin. If stdin is closed, read will return
		// an error and the goroutine exits, triggering program shutdown.
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		close(done)
	}()

	select {
	case <-ctx.Done():
		// interrupted by signal
	case <-done:
		// user pressed Enter
	}

	fmt.Println("Goodbye")
	fmt.Printf(" CPUs:        %d\n", runtime.NumCPU())
	fmt.Printf(" Time:        %s\n", time.Now().Format(time.RFC3339))
	fmt.Printf(" Port:        %s\n", port)
	fmt.Println("========================================")
}
