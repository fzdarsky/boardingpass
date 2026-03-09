// Package version provides build and version information injected at compile time via ldflags.
package version

import (
	"fmt"
	"runtime"
)

var (
	// commitFromGit is a constant representing the source version that
	// generated this build. It should be set during build via -ldflags.
	commitFromGit string
	// versionFromGit is a constant representing the version tag that
	// generated this build. It should be set during build via -ldflags.
	versionFromGit = "unknown"
	// major version
	majorFromGit string
	// minor version
	minorFromGit string
	// patch version
	patchFromGit string
	// build date, output of $(date +'%Y%m%d')
	buildDate string
	// state of git tree, either "clean" or "dirty"
	gitTreeState string
)

// Info contains versioning information about the binary.
type Info struct {
	Major        string `json:"major"`
	Minor        string `json:"minor"`
	Patch        string `json:"patch"`
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	GitTreeState string `json:"gitTreeState"`
	BuildDate    string `json:"buildDate"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`
}

func (info Info) String() string {
	if info.GitTreeState != "clean" {
		return fmt.Sprintf("%s-%s", info.GitVersion, info.GitTreeState)
	}
	return info.GitVersion
}

// Get returns the version information populated at build time.
func Get() Info {
	return Info{
		Major:        majorFromGit,
		Minor:        minorFromGit,
		Patch:        patchFromGit,
		GitCommit:    commitFromGit,
		GitVersion:   versionFromGit,
		GitTreeState: gitTreeState,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
