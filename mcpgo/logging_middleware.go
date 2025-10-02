// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"
)

// responseWriter wraps http.ResponseWriter to capture the status code and body.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Read and log request body if present
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Create a response writer wrapper to capture status code.
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Log request details with session ID header if present.
		sessionID := r.Header.Get("Mcp-Session-Id")
		contentType := r.Header.Get("Content-Type")
		requestBody := strings.ReplaceAll(string(bodyBytes), "\n", "\\n")
		log.Printf(strings.Repeat("-", 80))
		log.Printf("[REQUEST] %s | %s %s | Session-ID: %s | Content-Type: %s | Body: %s",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			sessionID,
			contentType,
			requestBody)

		// Call the actual handler.
		handler.ServeHTTP(wrapped, r)

		// Log response details with session ID header if set.
		responseSessionID := wrapped.Header().Get("Mcp-Session-Id")
		responseContentType := wrapped.Header().Get("Content-Type")
		responseBody := strings.ReplaceAll(wrapped.body.String(), "\n", "\\n")
		log.Printf("[RESPONSE] %s | %s %s | Status: %d | Session-ID: %s | Content-Type: %s | Body: %s",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			responseSessionID,
			responseContentType,
			responseBody)
	})
}
