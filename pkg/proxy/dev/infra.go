package dev

import (
	"net/http"
)

// APIServer interface for localProxy and remoteProxy
type APIServer interface {
	IsHealthy() bool
	ServeHTTP(rw http.ResponseWriter, req *http.Request)
}
