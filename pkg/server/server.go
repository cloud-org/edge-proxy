package server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
	"code.aliyun.com/openyurt/edge-proxy/pkg/profile"
)

// Server is an interface for providing http service for edge proxy
type Server interface {
	Run()
}

// edgeProxyServer includes stubServer and proxyServer,
// and stubServer handles requests by edge proxy itself, like profiling, metrics, healthz
// and proxyServer does not handle requests locally and proxy requests to kube-apiserver
type edgeProxyServer struct {
	stubServer  *http.Server
	proxyServer *http.Server
}

// NewEdgeProxyServer creates a Server object
func NewEdgeProxyServer(cfg *config.EdgeProxyConfiguration,
	proxyHandler http.Handler) (Server, error) {
	edgeMux := mux.NewRouter()
	registerHandlers(edgeMux)
	stubServer := &http.Server{
		Addr:           cfg.BindAddr,
		Handler:        edgeMux,
		MaxHeaderBytes: 1 << 20,
	}

	proxyServer := &http.Server{
		Addr:    cfg.EdgeProxyServerAddr,
		Handler: proxyHandler,
	}

	return &edgeProxyServer{
		stubServer:  stubServer,
		proxyServer: proxyServer,
	}, nil
}

// Run will start stubServer and proxy server
func (s *edgeProxyServer) Run() {
	go func() {
		err := s.stubServer.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	err := s.proxyServer.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

// registerHandler registers handlers for edge proxy server, like profiling, healthz.
func registerHandlers(c *mux.Router) {
	// register handler for health check
	c.HandleFunc("/v1/healthz", healthz).Methods("GET")

	// register handler for profile
	profile.Install(c)

	// register handler for metrics
	c.Handle("/metrics", promhttp.Handler())
}

// healthz returns ok for healthz request
func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}
