package buildinfo

import (
	"runtime/debug"
	"strings"
)

var (
	VersionTag  = "unknown"
	CommitHash  = "unknown"
	BuildTime   = "unknown"
	GitHubRunID = "unknown"
)

type InfoPayload struct {
	VersionTag  string `json:"version_tag"`
	CommitHash  string `json:"commit_hash"`
	BuildTime   string `json:"build_time"`
	GitHubRunID string `json:"github_run_id"`
	Source      string `json:"source"`
}

func Info() InfoPayload {
	info := InfoPayload{
		VersionTag:  strings.TrimSpace(VersionTag),
		CommitHash:  strings.TrimSpace(CommitHash),
		BuildTime:   strings.TrimSpace(BuildTime),
		GitHubRunID: strings.TrimSpace(GitHubRunID),
		Source:      "ldflags",
	}
	if info.VersionTag == "" {
		info.VersionTag = "unknown"
	}
	if info.CommitHash == "" {
		info.CommitHash = "unknown"
	}
	if info.BuildTime == "" {
		info.BuildTime = "unknown"
	}
	if info.GitHubRunID == "" {
		info.GitHubRunID = "unknown"
	}
	if info.CommitHash == "unknown" {
		if revision := vcsRevision(); revision != "" {
			info.CommitHash = revision
			info.Source = "go-buildinfo"
		}
	}
	return info
}

func vcsRevision() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range bi.Settings {
		if setting.Key == "vcs.revision" && strings.TrimSpace(setting.Value) != "" {
			return strings.TrimSpace(setting.Value)
		}
	}
	return ""
}
