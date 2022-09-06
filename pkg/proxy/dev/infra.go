package dev

import (
	"net/http"
)

type APIServer interface {
	IsHealthy() bool
	ServeHTTP(rw http.ResponseWriter, req *http.Request)
}
