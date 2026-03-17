package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var validPlatforms = map[string]bool{
	"darwin-arm64":     true,
	"darwin-x64":       true,
	"linux-arm64":      true,
	"linux-x64":        true,
	"linux-arm64-musl": true,
	"linux-x64-musl":   true,
}

func dataDir() string {
	dir := os.Getenv("DATA_DIR")
	if dir == "" {
		dir = "/data/cc-download"
	}
	return dir
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.String(), rw.status, time.Since(start))
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	osParam := r.URL.Query().Get("os")
	arch := r.URL.Query().Get("arch")
	variant := r.URL.Query().Get("variant")

	if osParam == "" || arch == "" {
		http.Error(w, "missing required parameters: os, arch", http.StatusBadRequest)
		return
	}

	if osParam != "darwin" && osParam != "linux" {
		http.Error(w, fmt.Sprintf("unsupported os: %s", osParam), http.StatusBadRequest)
		return
	}

	if arch != "arm64" && arch != "x64" {
		http.Error(w, fmt.Sprintf("unsupported arch: %s", arch), http.StatusBadRequest)
		return
	}

	platform := osParam + "-" + arch
	if variant != "" {
		if variant != "musl" {
			http.Error(w, fmt.Sprintf("unsupported variant: %s", variant), http.StatusBadRequest)
			return
		}
		platform += "-" + variant
	}

	if !validPlatforms[platform] {
		http.Error(w, fmt.Sprintf("unsupported platform: %s", platform), http.StatusBadRequest)
		return
	}

	filename := "claude-code-" + platform
	filePath := filepath.Join(dataDir(), filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("binary not found: %s", filename), http.StatusNotFound)
		return
	}

	http.Redirect(w, r, "/files/"+filename, http.StatusFound)
}

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/claude/download", downloadHandler)

	srv := &http.Server{
		Addr:         addr,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Starting cc-download server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server stopped")
}
