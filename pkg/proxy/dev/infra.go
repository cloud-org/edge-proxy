package dev

import "net/http"

// APIServerProxy interface for localProxy and remoteProxy
type APIServerProxy interface {
	// IsHealthy check proxy healthy or not
	IsHealthy() bool
	// ServeHTTP should implements net/http http.Handler
	ServeHTTP(rw http.ResponseWriter, req *http.Request)
}
