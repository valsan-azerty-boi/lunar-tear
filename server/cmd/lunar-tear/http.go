package main

import (
	"fmt"
	"log"
	"net/http"

	"lunar-tear/server/internal/service"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func startHTTP(port int, resourcesBaseURL string, adminMux *http.ServeMux) {
	mux := http.NewServeMux()

	// Register admin routes if available
	if adminMux != nil {
		mux.Handle("/admin/", adminMux)
	}

	// Octo routes (asset delivery) — catch-all
	octoServer := service.NewOctoHTTPServer(resourcesBaseURL)
	mux.Handle("/", octoServer.Handler())

	h2s := &http2.Server{}
	handler := h2c.NewHandler(mux, h2s)
	log.Printf("HTTP server listening on :%d (HTTP/1.1 + h2c)", port)
	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: handler}
	http2.ConfigureServer(srv, h2s)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("HTTP server on %d failed: %v", port, err)
	}
}
