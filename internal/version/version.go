package version

import (
	"fmt"
	"runtime/debug"
)

// String renders the build information embedded by the Go toolchain:
// the module version stamped by `go install`, otherwise the VCS revision
// of a local `go build`. A binary without build metadata reports
// "(unknown)".
func String() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.GoVersion == "" {
		return "(unknown)"
	}
	v := info.Main.Version
	if v == "" || v == "(devel)" {
		v = devel(info)
	}
	return fmt.Sprintf("%s (%s)", v, info.GoVersion)
}

// devel describes a locally built binary from the VCS settings the
// toolchain stamps in when building inside a repository. It falls back to
// the plain "(devel)" of a build outside version control.
func devel(info *debug.BuildInfo) string {
	rev := buildSetting(info, "vcs.revision")
	if rev == "" {
		return "(devel)"
	}
	v := "devel+" + rev[:min(7, len(rev))]
	if buildSetting(info, "vcs.modified") == "true" {
		v += "-dirty"
	}
	return v
}

func buildSetting(info *debug.BuildInfo, key string) string {
	for _, s := range info.Settings {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}
