package dev

import (
	"net/http"
)

type LoadBalancer interface {
	IsHealthy() bool
	ServeHTTP(rw http.ResponseWriter, req *http.Request)
}
