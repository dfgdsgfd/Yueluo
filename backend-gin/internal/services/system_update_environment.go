package services

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/buildinfo"
	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

func systemUpdateEnvironment(ctx context.Context) SystemUpdateEnvironment {
	cwd, _ := os.Getwd()
	return SystemUpdateEnvironment{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		CWD:          cwd,
		ToolStatuses: []SystemUpdateToolStatus{},
	}
}

func (s SystemUpdateService) componentChecks(ctx context.Context, runCommands bool) []SystemUpdateCheck {
	return []SystemUpdateCheck{
		systemUpdatePostgresStorageCheck(s.db),
		ffmpegExecutableCheck(ctx, "ffmpeg", s.cfg.Video.FFmpegPath, runCommands),
		ffmpegExecutableCheck(ctx, "ffprobe", s.cfg.Video.FFprobePath, runCommands),
		geoIPCountryDatabaseCheck(s.cfg.GeoIP),
	}
}

func systemUpdatePostgresStorageCheck(db *gorm.DB) SystemUpdateCheck {
	if db == nil {
		return SystemUpdateCheck{Key: "system_update_storage", Label: "更新配置持久化", Status: SystemUpdateStatusFailed, Message: "数据库连接不可用"}
	}
	if !db.Migrator().HasTable(&domain.SystemUpdateConfig{}) || !db.Migrator().HasTable(&domain.SystemUpdateJob{}) {
		return SystemUpdateCheck{Key: "system_update_storage", Label: "更新配置持久化", Status: SystemUpdateStatusFailed, Message: "PostgreSQL 更新配置表或任务表不存在"}
	}
	return SystemUpdateCheck{Key: "system_update_storage", Label: "更新配置持久化", Status: SystemUpdateStatusSucceeded, Message: "配置和任务保存在 PostgreSQL"}
}

func ffmpegExecutableCheck(ctx context.Context, key string, rawPath string, runCommand bool) SystemUpdateCheck {
	label := strings.ToUpper(key[:1]) + key[1:]
	path, err := executablePath(rawPath)
	if err != nil {
		return SystemUpdateCheck{Key: key, Label: label, Status: SystemUpdateStatusFailed, Message: ffmpegPermissionHint(rawPath, err), Path: rawPath}
	}
	if !runCommand {
		return SystemUpdateCheck{Key: key, Label: label, Status: SystemUpdateStatusSucceeded, Message: "文件可访问", Path: path}
	}
	runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, path, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return SystemUpdateCheck{Key: key, Label: label, Status: SystemUpdateStatusFailed, Message: ffmpegPermissionHint(path, err) + outputSummary(output), Path: path}
	}
	firstLine := firstOutputLine(output)
	if firstLine == "" {
		firstLine = "运行成功"
	}
	return SystemUpdateCheck{Key: key, Label: label, Status: SystemUpdateStatusSucceeded, Message: firstLine, Path: path}
}

func geoIPCountryDatabaseCheck(cfg config.GeoIPConfig) SystemUpdateCheck {
	path := resolveSystemUpdatePath(defaultIfBlank(cfg.CountryMMDBPath, "data/geoip/Country.mmdb"))
	info, err := os.Stat(path)
	if err != nil {
		return SystemUpdateCheck{Key: "geoip_country_mmdb", Label: "GeoIP 国家库", Status: SystemUpdateStatusFailed, Message: err.Error(), Path: path}
	}
	if info.IsDir() {
		return SystemUpdateCheck{Key: "geoip_country_mmdb", Label: "GeoIP 国家库", Status: SystemUpdateStatusFailed, Message: "路径是目录，不是 Country.mmdb 文件", Path: path}
	}
	return SystemUpdateCheck{Key: "geoip_country_mmdb", Label: "GeoIP 国家库", Status: SystemUpdateStatusSucceeded, Message: fmt.Sprintf("Country.mmdb 可读，%s", formatSystemUpdateBytes(info.Size())), Path: path}
}

func artifactSourceCheck(ctx context.Context, cfg domain.SystemUpdateConfig, kind string) SystemUpdateCheck {
	label := "前端产物"
	if kind == "backend" {
		label = "后端产物"
	}
	source, err := resolveSystemUpdateArtifactSource(ctx, cfg, kind)
	if err != nil {
		return SystemUpdateCheck{Key: kind + "_artifact", Label: label, Status: SystemUpdateStatusFailed, Message: err.Error()}
	}
	message := source.Name
	if source.ReleaseTag != "" {
		message += " / " + source.ReleaseTag
	}
	return SystemUpdateCheck{Key: kind + "_artifact", Label: label, Status: SystemUpdateStatusSucceeded, Message: message, Path: source.URL}
}

func writableDirCheck(key, label, rawPath string) SystemUpdateCheck {
	path := resolveSystemUpdatePath(rawPath)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return SystemUpdateCheck{Key: key, Label: label, Status: SystemUpdateStatusFailed, Message: err.Error(), Path: path}
	}
	testFile := filepath.Join(path, ".write-check")
	if err := os.WriteFile(testFile, []byte("ok"), 0o644); err != nil {
		return SystemUpdateCheck{Key: key, Label: label, Status: SystemUpdateStatusFailed, Message: err.Error(), Path: path}
	}
	_ = os.Remove(testFile)
	return SystemUpdateCheck{Key: key, Label: label, Status: SystemUpdateStatusSucceeded, Message: "目录可写", Path: path}
}

func verifySystemUpdateInstallDir(dir string) error {
	stat, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("无法访问目录: %w", err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("路径不是目录")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("无法读取目录: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}
	if !exists(filepath.Join(dir, "package.json")) {
		return fmt.Errorf("目录非空但未找到 package.json，请确认前端安装目录设置正确")
	}
	return nil
}

func defaultSystemUpdateConfig() domain.SystemUpdateConfig {
	root := detectWorkspaceRoot()
	backendRoot := detectBackendRoot(root)
	backendName := "yuem-go-" + runtime.GOOS + "-" + runtime.GOARCH + exeSuffix()
	return domain.SystemUpdateConfig{
		FrontendBranch:       "main",
		BackendBranch:        "main",
		FrontendReleaseTag:   "latest",
		BackendReleaseTag:    "latest",
		FrontendAssetPattern: "frontend-" + runtime.GOOS + "-" + runtime.GOARCH + ".zip",
		BackendAssetPattern:  backendName,
		FrontendInstallDir:   filepath.Join(backendRoot, "frontend"),
		BackendInstallPath:   filepath.Join(backendRoot, backendName),
		FrontendStartMode:    SystemUpdateFrontendStartModeStart,
		FrontendSourceDir:    filepath.Join(root, "front-end-nextjs"),
		BackendSourceDir:     backendRoot,
		ArtifactDir:          filepath.Join(backendRoot, "updates"),
	}
}

func fillSystemUpdateDefaults(cfg *domain.SystemUpdateConfig) {
	defaults := defaultSystemUpdateConfig()
	cfg.FrontendBranch = defaultIfBlank(cfg.FrontendBranch, defaults.FrontendBranch)
	cfg.BackendBranch = defaultIfBlank(cfg.BackendBranch, defaults.BackendBranch)
	cfg.FrontendReleaseTag = defaultIfBlank(cfg.FrontendReleaseTag, defaults.FrontendReleaseTag)
	cfg.BackendReleaseTag = defaultIfBlank(cfg.BackendReleaseTag, defaults.BackendReleaseTag)
	cfg.FrontendAssetPattern = defaultIfBlank(cfg.FrontendAssetPattern, defaults.FrontendAssetPattern)
	cfg.BackendAssetPattern = defaultIfBlank(cfg.BackendAssetPattern, defaults.BackendAssetPattern)
	cfg.FrontendInstallDir = defaultIfBlank(cfg.FrontendInstallDir, defaults.FrontendInstallDir)
	cfg.BackendInstallPath = defaultIfBlank(cfg.BackendInstallPath, defaults.BackendInstallPath)
	cfg.FrontendStartMode = normalizeFrontendStartMode(defaultIfBlank(cfg.FrontendStartMode, defaults.FrontendStartMode))
	cfg.FrontendSourceDir = defaultIfBlank(cfg.FrontendSourceDir, defaults.FrontendSourceDir)
	cfg.BackendSourceDir = defaultIfBlank(cfg.BackendSourceDir, defaults.BackendSourceDir)
	cfg.ArtifactDir = defaultIfBlank(cfg.ArtifactDir, defaults.ArtifactDir)
}

func detectWorkspaceRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if exists(filepath.Join(dir, "backend-gin")) && exists(filepath.Join(dir, "front-end-nextjs")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	if filepath.Base(cwd) == "backend-gin" {
		return filepath.Dir(cwd)
	}
	return cwd
}

func detectBackendRoot(workspaceRoot string) string {
	cwd, err := os.Getwd()
	if err == nil && exists(filepath.Join(cwd, "go.mod")) && exists(filepath.Join(cwd, "cmd", "api")) {
		return cwd
	}
	candidate := filepath.Join(workspaceRoot, "backend-gin")
	if exists(candidate) {
		return candidate
	}
	return cwd
}

func systemUpdateConfigView(cfg domain.SystemUpdateConfig) SystemUpdateConfigView {
	updated := ""
	if cfg.UpdatedAt != nil {
		updated = cfg.UpdatedAt.Format(time.RFC3339)
	}
	return SystemUpdateConfigView{
		ID:                   cfg.ID,
		FrontendRepoURL:      cfg.FrontendRepoURL,
		BackendRepoURL:       cfg.BackendRepoURL,
		GithubTokenSet:       strings.TrimSpace(cfg.GithubToken) != "",
		GithubTokenMasked:    maskToken(cfg.GithubToken),
		FrontendBranch:       cfg.FrontendBranch,
		BackendBranch:        cfg.BackendBranch,
		FrontendReleaseTag:   cfg.FrontendReleaseTag,
		BackendReleaseTag:    cfg.BackendReleaseTag,
		FrontendArtifactURL:  cfg.FrontendArtifactURL,
		BackendArtifactURL:   cfg.BackendArtifactURL,
		FrontendAssetPattern: cfg.FrontendAssetPattern,
		BackendAssetPattern:  cfg.BackendAssetPattern,
		FrontendInstallDir:   cfg.FrontendInstallDir,
		BackendInstallPath:   cfg.BackendInstallPath,
		FrontendStartMode:    cfg.FrontendStartMode,
		FrontendStartCommand: frontendStartCommand(cfg.FrontendStartMode),
		FrontendSourceDir:    cfg.FrontendSourceDir,
		BackendSourceDir:     cfg.BackendSourceDir,
		ArtifactDir:          cfg.ArtifactDir,
		FrontendBuildCommand: cfg.FrontendBuildCommand,
		BackendBuildCommand:  cfg.BackendBuildCommand,
		UpdatedAt:            updated,
	}
}

func systemUpdateJobView(job domain.SystemUpdateJob) SystemUpdateJobView {
	paths := []string{}
	if len(job.ArtifactPaths) > 0 {
		_ = json.Unmarshal(job.ArtifactPaths, &paths)
	}
	artifacts := []SystemUpdateArtifactView{}
	if len(job.ArtifactMeta) > 0 {
		_ = json.Unmarshal(job.ArtifactMeta, &artifacts)
	}
	view := SystemUpdateJobView{
		ID:            job.ID,
		Action:        job.Action,
		Status:        job.Status,
		FrontendState: job.FrontendState,
		BackendState:  job.BackendState,
		Progress:      job.Progress,
		CurrentStep:   job.CurrentStep,
		ArtifactPaths: paths,
		Artifacts:     artifacts,
		Logs:          job.Logs,
	}
	if job.ErrorMessage != nil {
		view.ErrorMessage = *job.ErrorMessage
	}
	if job.StartedAt != nil {
		view.StartedAt = job.StartedAt.Format(time.RFC3339)
	}
	if job.FinishedAt != nil {
		view.FinishedAt = job.FinishedAt.Format(time.RFC3339)
	}
	if !job.CreatedAt.IsZero() {
		view.CreatedAt = job.CreatedAt.Format(time.RFC3339)
	}
	if job.UpdatedAt != nil {
		view.UpdatedAt = job.UpdatedAt.Format(time.RFC3339)
	}
	return view
}

func systemUpdateCurrentInfo(cfg domain.SystemUpdateConfig) SystemUpdateCurrentInfo {
	backend := buildinfo.Info()
	current := SystemUpdateCurrentInfo{
		Backend: SystemUpdateVersionInfo{
			VersionTag:  backend.VersionTag,
			CommitHash:  backend.CommitHash,
			BuildTime:   backend.BuildTime,
			GitHubRunID: backend.GitHubRunID,
			Source:      backend.Source,
		},
	}
	if frontend := readFrontendVersionInfo(cfg.FrontendInstallDir); frontend != nil {
		current.Frontend = frontend
	}
	return current
}

func readFrontendVersionInfo(installDir string) *SystemUpdateVersionInfo {
	base := resolveSystemUpdatePath(installDir)
	for _, name := range []string{"version.json", filepath.Join("public", "version.json")} {
		candidate := filepath.Join(base, name)
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		var payload struct {
			VersionTag  string `json:"version_tag"`
			Tag         string `json:"tag"`
			CommitHash  string `json:"commit_hash"`
			BuildTime   string `json:"build_time"`
			GitHubRunID string `json:"github_run_id"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			continue
		}
		versionTag := payload.VersionTag
		if strings.TrimSpace(versionTag) == "" {
			versionTag = payload.Tag
		}
		info := &SystemUpdateVersionInfo{
			VersionTag:  defaultIfBlank(versionTag, "unknown"),
			CommitHash:  defaultIfBlank(payload.CommitHash, "unknown"),
			BuildTime:   defaultIfBlank(payload.BuildTime, "unknown"),
			GitHubRunID: defaultIfBlank(payload.GitHubRunID, "unknown"),
			Source:      "version.json",
			Path:        candidate,
		}
		return info
	}
	return nil
}

func zipStripPrefix(files []*zip.File) string {
	names := make([]string, 0, len(files))
	for _, item := range files {
		if item.FileInfo().IsDir() {
			continue
		}
		name := filepath.ToSlash(item.Name)
		if strings.TrimSpace(name) != "" {
			names = append(names, name)
		}
	}
	for _, prefix := range []string{"build/frontend/", "frontend/"} {
		all := len(names) > 0
		for _, name := range names {
			if !strings.HasPrefix(name, prefix) {
				all = false
				break
			}
		}
		if all {
			return prefix
		}
	}
	return ""
}

func safeZipDestination(root, name string) (string, error) {
	cleanRoot := filepath.Clean(root)
	destination := filepath.Clean(filepath.Join(cleanRoot, filepath.FromSlash(name)))
	if destination != cleanRoot && !strings.HasPrefix(destination, cleanRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("zip entry escapes target directory: %s", name)
	}
	return destination, nil
}

func parseGitHubRepo(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", errors.New("未配置 GitHub 仓库地址或直接产物 URL")
	}
	if after, ok := strings.CutPrefix(raw, "git@github.com:"); ok {
		value := after
		value = strings.TrimSuffix(value, ".git")
		parts := strings.Split(value, "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1], nil
		}
	}
	parsed, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(parsed.Host, "github.com") {
		return "", "", fmt.Errorf("不是有效的 GitHub 仓库地址: %s", raw)
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("不是有效的 GitHub 仓库地址: %s", raw)
	}
	repo := strings.TrimSuffix(parts[1], ".git")
	if parts[0] == "" || repo == "" {
		return "", "", fmt.Errorf("不是有效的 GitHub 仓库地址: %s", raw)
	}
	return parts[0], repo, nil
}

func wildcardMatch(pattern, name string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || pattern == "*" {
		return true
	}
	for part := range strings.SplitSeq(pattern, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		lowerName := strings.ToLower(name)
		ok, err := path.Match(part, lowerName)
		if err == nil && ok {
			return true
		}
		if !strings.ContainsAny(part, "*?[") && strings.EqualFold(part, name) {
			return true
		}
	}
	return false
}

func isGitHubURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Host)
	return host == "github.com" || strings.HasSuffix(host, ".github.com") || strings.Contains(host, "githubusercontent.com")
}

func isZipName(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".zip")
}

func isZipFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".zip")
}

func safeArtifactFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "artifact-" + time.Now().Format("20060102-150405")
	}
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return replacer.Replace(name)
}

func appendSystemUpdateLog(logs *strings.Builder, format string, args ...any) {
	logs.WriteString(time.Now().Format("15:04:05"))
	logs.WriteString(" ")
	logs.WriteString(fmt.Sprintf(format, args...))
	logs.WriteString("\n")
}

func trimSystemUpdateLogs(value string) string {
	const maxLogBytes = 120000
	if len(value) <= maxLogBytes {
		return value
	}
	return value[len(value)-maxLogBytes:]
}

func maskToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func normalizeReleaseTag(value string) string {
	return defaultIfBlank(value, "latest")
}

func normalizeFrontendStartMode(value string) string {
	switch strings.TrimSpace(value) {
	case SystemUpdateFrontendStartModeBuildStart:
		return SystemUpdateFrontendStartModeBuildStart
	default:
		return SystemUpdateFrontendStartModeStart
	}
}

func frontendStartCommand(mode string) string {
	if normalizeFrontendStartMode(mode) == SystemUpdateFrontendStartModeBuildStart {
		return "npm ci && npm run build && npm run start"
	}
	return "npm run start"
}

func resolveSystemUpdatePath(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return raw
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw)
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return filepath.Clean(raw)
	}
	return abs
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func defaultIfBlank(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func normalizeSystemUpdateAction(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "deploy_artifacts"
	}
	return value
}

func ffmpegPermissionHint(path string, err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	target := strings.TrimSpace(path)
	if target == "" {
		target = "ffmpeg"
	}
	lower := strings.ToLower(message)
	if errors.Is(err, os.ErrPermission) || strings.Contains(lower, "permission denied") || strings.Contains(lower, "access is denied") {
		return fmt.Sprintf("%s; 系统拒绝执行 %s。请在容器内确认: 1) 路径指向真实二进制而不是目录或损坏 symlink 2) 父目录有执行权限 3) 挂载点不是 noexec 4) 运行用户可执行该文件 5) 二进制架构匹配容器 6) SELinux/AppArmor 未阻止执行", message, target)
	}
	return message
}

func firstOutputLine(output []byte) string {
	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func outputSummary(output []byte) string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return ""
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if len(text) > 600 {
		text = text[:600] + "..."
	}
	return ": " + text
}

func ternaryString(ok bool, yes string, no string) string {
	if ok {
		return yes
	}
	return no
}

func ternaryInt(ok bool, yes int, no int) int {
	if ok {
		return yes
	}
	return no
}

func formatSystemUpdateBytes(value int64) string {
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	size := float64(value)
	for _, unit := range units {
		size = size / 1024
		if size < 1024 {
			return fmt.Sprintf("%.1f %s", size, unit)
		}
	}
	return fmt.Sprintf("%.1f PB", size/1024)
}

func ptr[T any](value T) *T {
	return &value
}
