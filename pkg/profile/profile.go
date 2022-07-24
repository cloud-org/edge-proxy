package profile

import (
	"net/http"
	"net/http/pprof"

	"k8s.io/klog/v2"

	"github.com/gorilla/mux"
)

// Install adds the Profiling webservice to the given mux.
func Install(c *mux.Router) {
	c.HandleFunc("/debug/pprof/profile", func(rw http.ResponseWriter, req *http.Request) {
		klog.Infof("enter pprof profile: %v", req.URL.String())
		pprof.Profile(rw, req)
	})
	c.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	c.HandleFunc("/debug/pprof/trace", pprof.Trace)
	c.HandleFunc("/debug/pprof", redirectTo("/debug/pprof/"))
	c.PathPrefix("/debug/pprof/").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		klog.Infof("enter pprof, %v", req.URL.String())
		pprof.Index(rw, req)
	})
}

// redirectTo redirects request to a certain destination.
func redirectTo(to string) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		http.Redirect(rw, req, to, http.StatusFound)
	}
}
