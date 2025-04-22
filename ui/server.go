// ui/server.go
package ui

import (
	"embed"
	"fmt"
	"net/http"
)

//go:embed static/*
var content embed.FS

func StartUI(port int) {
	fs := http.FS(content)
	http.Handle("/", http.FileServer(fs))
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("UI available at http://localhost%s\n", addr)
	http.ListenAndServe(addr, nil)
}
