package dev

import (
	"net/http"
)

type APIServerProxy interface {
	IsHealthy() bool
	ServeHTTP(rw http.ResponseWriter, req *http.Request)
}
