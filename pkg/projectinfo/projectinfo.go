package projectinfo

import (
	"fmt"
	"runtime"
)

var (
	gitVersion = "v0.0.0"
	gitCommit  = "unknown"
	buildDate  = "1970-01-01T00:00:00Z"
)

// GetProxyName returns name of edge proxy
func GetProxyName() string {
	return "edge-proxy"
}

// normalizeGitCommit reserve 7 characters for gitCommit
func normalizeGitCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}

	return commit
}

// Info contains version information.
type Info struct {
	GitVersion string `json:"gitVersion"`
	GitCommit  string `json:"gitCommit"`
	BuildDate  string `json:"buildDate"`
	GoVersion  string `json:"goVersion"`
	Compiler   string `json:"compiler"`
	Platform   string `json:"platform"`
}

// Get returns the overall codebase version.
func Get() Info {
	return Info{
		GitVersion: gitVersion,
		GitCommit:  normalizeGitCommit(gitCommit),
		BuildDate:  buildDate,
		GoVersion:  runtime.Version(),
		Compiler:   runtime.Compiler,
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
