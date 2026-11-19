package version

import (
	"fmt"
	"runtime/debug"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func String() string {
	if Version != "dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		settings := make(map[string]string, len(info.Settings))
		for _, s := range info.Settings {
			settings[s.Key] = s.Value
		}
		if rev := settings["vcs.revision"]; rev != "" {
			v := rev
			if len(v) > 7 {
				v = v[:7]
			}
			if settings["vcs.modified"] == "true" {
				v += "-dirty"
			}
			return "dev-" + v
		}
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return Version
}

func Full() string {
	return fmt.Sprintf("%s (commit %s, built %s)", String(), Commit, Date)
}
