package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

var validPlatforms = map[string]bool{
	"darwin-arm64":     true,
	"darwin-x64":       true,
	"linux-arm64":      true,
	"linux-x64":        true,
	"linux-arm64-musl": true,
	"linux-x64-musl":   true,
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
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
	http.Redirect(w, r, "/files/"+filename, http.StatusFound)
}

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	http.HandleFunc("/claude/download", downloadHandler)

	log.Printf("Starting cc-download server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
