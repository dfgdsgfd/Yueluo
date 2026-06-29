package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

const (
	SystemUpdateStatusPending   = "pending"
	SystemUpdateStatusRunning   = "running"
	SystemUpdateStatusSucceeded = "succeeded"
	SystemUpdateStatusFailed    = "failed"
	SystemUpdateStatusSkipped   = "skipped"

	SystemUpdateFrontendStartModeStart      = "start"
	SystemUpdateFrontendStartModeBuildStart = "build_start"
)

type SystemUpdateService struct {
	db  *gorm.DB
	cfg config.Config
}

type SystemUpdateConfigInput struct {
	FrontendRepoURL      string `json:"frontend_repo_url"`
	BackendRepoURL       string `json:"backend_repo_url"`
	GithubToken          string `json:"github_token"`
	ClearGithubToken     bool   `json:"clear_github_token"`
	FrontendBranch       string `json:"frontend_branch"`
	BackendBranch        string `json:"backend_branch"`
	FrontendReleaseTag   string `json:"frontend_release_tag"`
	BackendReleaseTag    string `json:"backend_release_tag"`
	FrontendArtifactURL  string `json:"frontend_artifact_url"`
	BackendArtifactURL   string `json:"backend_artifact_url"`
	FrontendAssetPattern string `json:"frontend_asset_pattern"`
	BackendAssetPattern  string `json:"backend_asset_pattern"`
	FrontendInstallDir   string `json:"frontend_install_dir"`
	BackendInstallPath   string `json:"backend_install_path"`
	FrontendStartMode    string `json:"frontend_start_mode"`
	FrontendSourceDir    string `json:"frontend_source_dir"`
	BackendSourceDir     string `json:"backend_source_dir"`
	ArtifactDir          string `json:"artifact_dir"`
	FrontendBuildCommand string `json:"frontend_build_command"`
	BackendBuildCommand  string `json:"backend_build_command"`
}

type SystemUpdateRunInput struct {
	Frontend           bool   `json:"frontend"`
	Backend            bool   `json:"backend"`
	Action             string `json:"action,omitempty"`
	SkipGit            bool   `json:"skip_git"`
	FrontendReleaseTag string `json:"frontend_release_tag,omitempty"`
	BackendReleaseTag  string `json:"backend_release_tag,omitempty"`
}

type SystemUpdateConfigView struct {
	ID                   int64  `json:"id"`
	FrontendRepoURL      string `json:"frontend_repo_url"`
	BackendRepoURL       string `json:"backend_repo_url"`
	GithubTokenSet       bool   `json:"github_token_set"`
	GithubTokenMasked    string `json:"github_token_masked"`
	FrontendBranch       string `json:"frontend_branch"`
	BackendBranch        string `json:"backend_branch"`
	FrontendReleaseTag   string `json:"frontend_release_tag"`
	BackendReleaseTag    string `json:"backend_release_tag"`
	FrontendArtifactURL  string `json:"frontend_artifact_url"`
	BackendArtifactURL   string `json:"backend_artifact_url"`
	FrontendAssetPattern string `json:"frontend_asset_pattern"`
	BackendAssetPattern  string `json:"backend_asset_pattern"`
	FrontendInstallDir   string `json:"frontend_install_dir"`
	BackendInstallPath   string `json:"backend_install_path"`
	FrontendStartMode    string `json:"frontend_start_mode"`
	FrontendStartCommand string `json:"frontend_start_command"`
	FrontendSourceDir    string `json:"frontend_source_dir"`
	BackendSourceDir     string `json:"backend_source_dir"`
	ArtifactDir          string `json:"artifact_dir"`
	FrontendBuildCommand string `json:"frontend_build_command"`
	BackendBuildCommand  string `json:"backend_build_command"`
	UpdatedAt            string `json:"updated_at,omitempty"`
}

type SystemUpdateStatusPayload struct {
	Config      SystemUpdateConfigView  `json:"config"`
	Environment SystemUpdateEnvironment `json:"environment"`
	Current     SystemUpdateCurrentInfo `json:"current"`
	Checks      []SystemUpdateCheck     `json:"checks,omitempty"`
	Components  []SystemUpdateCheck     `json:"component_checks,omitempty"`
	LastJob     *SystemUpdateJobView    `json:"last_job,omitempty"`
}

type SystemUpdateCurrentInfo struct {
	Backend  SystemUpdateVersionInfo  `json:"backend"`
	Frontend *SystemUpdateVersionInfo `json:"frontend,omitempty"`
}

type SystemUpdateReleaseOptionsPayload struct {
	Frontend      []SystemUpdateReleaseOption `json:"frontend"`
	Backend       []SystemUpdateReleaseOption `json:"backend"`
	FrontendError string                      `json:"frontend_error,omitempty"`
	BackendError  string                      `json:"backend_error,omitempty"`
}

type SystemUpdateReleaseOption struct {
	TagName        string                           `json:"tag_name"`
	Name           string                           `json:"name"`
	HTMLURL        string                           `json:"html_url"`
	TargetCommit   string                           `json:"target_commit"`
	CommitHash     string                           `json:"commit_hash"`
	CreatedAt      string                           `json:"created_at"`
	PublishedAt    string                           `json:"published_at"`
	MatchingAssets []SystemUpdateReleaseAssetOption `json:"matching_assets"`
	Assets         []SystemUpdateReleaseAssetOption `json:"assets"`
}

type SystemUpdateVersionInfo struct {
	VersionTag  string `json:"version_tag"`
	CommitHash  string `json:"commit_hash"`
	BuildTime   string `json:"build_time"`
	GitHubRunID string `json:"github_run_id"`
	Source      string `json:"source"`
	Path        string `json:"path,omitempty"`
}

type SystemUpdateReleaseAssetOption struct {
	Name        string `json:"name"`
	SizeBytes   int64  `json:"size_bytes"`
	DownloadURL string `json:"download_url"`
	UpdatedAt   string `json:"updated_at"`
	SHA256      string `json:"sha256,omitempty"`
	Matched     bool   `json:"matched"`
}

type SystemUpdateEnvironment struct {
	OS           string                   `json:"os"`
	Arch         string                   `json:"arch"`
	CWD          string                   `json:"cwd"`
	ToolStatuses []SystemUpdateToolStatus `json:"tools"`
}

type SystemUpdateToolStatus struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
	Message string `json:"message,omitempty"`
}

type SystemUpdateCheck struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

type SystemUpdateArtifactView struct {
	Kind            string `json:"kind"`
	Name            string `json:"name"`
	SourceURL       string `json:"source_url,omitempty"`
	CachePath       string `json:"cache_path,omitempty"`
	InstallPath     string `json:"install_path,omitempty"`
	SizeBytes       int64  `json:"size_bytes"`
	SHA256          string `json:"sha256"`
	ReleaseTag      string `json:"release_tag,omitempty"`
	GitHubUpdatedAt string `json:"github_updated_at,omitempty"`
	DownloadedAt    string `json:"downloaded_at,omitempty"`
	InstalledAt     string `json:"installed_at,omitempty"`
}

type SystemUpdateJobView struct {
	ID            int64                      `json:"id"`
	Action        string                     `json:"action"`
	Status        string                     `json:"status"`
	FrontendState string                     `json:"frontend_state"`
	BackendState  string                     `json:"backend_state"`
	Progress      int                        `json:"progress"`
	CurrentStep   string                     `json:"current_step"`
	ArtifactPaths []string                   `json:"artifact_paths"`
	Artifacts     []SystemUpdateArtifactView `json:"artifacts"`
	Logs          string                     `json:"logs"`
	ErrorMessage  string                     `json:"error_message,omitempty"`
	StartedAt     string                     `json:"started_at,omitempty"`
	FinishedAt    string                     `json:"finished_at,omitempty"`
	CreatedAt     string                     `json:"created_at,omitempty"`
	UpdatedAt     string                     `json:"updated_at,omitempty"`
}

type systemUpdateArtifactSource struct {
	Kind            string
	Name            string
	URL             string
	AssetID         int64
	ReleaseTag      string
	SizeBytes       int64
	GitHubUpdatedAt string
}

type githubReleaseResponse struct {
	TagName      string                      `json:"tag_name"`
	Name         string                      `json:"name"`
	HTMLURL      string                      `json:"html_url"`
	TargetCommit string                      `json:"target_commitish"`
	CreatedAt    string                      `json:"created_at"`
	PublishedAt  string                      `json:"published_at"`
	Assets       []githubReleaseAssetPayload `json:"assets"`
}

type githubReleaseAssetPayload struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	UpdatedAt          string `json:"updated_at"`
	Digest             string `json:"digest"`
}

type systemUpdateRunner struct {
	service   SystemUpdateService
	job       domain.SystemUpdateJob
	logs      strings.Builder
	paths     []string
	artifacts []SystemUpdateArtifactView
}

func NewSystemUpdateService(db *gorm.DB) SystemUpdateService {
	return SystemUpdateService{db: db}
}

func NewSystemUpdateServiceWithConfig(db *gorm.DB, cfg config.Config) SystemUpdateService {
	return SystemUpdateService{db: db, cfg: cfg}
}

func (s SystemUpdateService) Status(ctx context.Context) (*SystemUpdateStatusPayload, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	last, err := s.lastJob(ctx)
	if err != nil {
		return nil, err
	}
	return &SystemUpdateStatusPayload{
		Config:      systemUpdateConfigView(cfg),
		Environment: systemUpdateEnvironment(ctx),
		Current:     systemUpdateCurrentInfo(cfg),
		Components:  s.componentChecks(ctx, false),
		LastJob:     last,
	}, nil
}

func (s SystemUpdateService) Check(ctx context.Context) (*SystemUpdateStatusPayload, error) {
	status, err := s.Status(ctx)
	if err != nil {
		return nil, err
	}
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	status.Checks = append(status.Checks, writableDirCheck("artifact_dir", "下载缓存目录", cfg.ArtifactDir))
	status.Checks = append(status.Checks, writableDirCheck("frontend_install_dir", "前端安装目录", cfg.FrontendInstallDir))
	status.Checks = append(status.Checks, writableDirCheck("backend_install_dir", "后端安装目录", filepath.Dir(resolveSystemUpdatePath(cfg.BackendInstallPath))))
	status.Checks = append(status.Checks, artifactSourceCheck(ctx, cfg, "frontend"))
	status.Checks = append(status.Checks, artifactSourceCheck(ctx, cfg, "backend"))
	status.Components = s.componentChecks(ctx, true)
	return status, nil
}

func (s SystemUpdateService) ReleaseOptions(ctx context.Context) (*SystemUpdateReleaseOptionsPayload, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	payload := &SystemUpdateReleaseOptionsPayload{}
	if strings.TrimSpace(cfg.FrontendArtifactURL) == "" {
		if strings.TrimSpace(cfg.FrontendRepoURL) == "" {
			payload.FrontendError = "请先在「更新配置」中填写「前端 GitHub 仓库」地址并保存"
		} else {
			payload.Frontend, err = listSystemUpdateReleaseOptions(ctx, cfg.FrontendRepoURL, cfg.FrontendAssetPattern, cfg.GithubToken)
			if err != nil {
				payload.FrontendError = err.Error()
			}
		}
	}
	if strings.TrimSpace(cfg.BackendArtifactURL) == "" {
		if strings.TrimSpace(cfg.BackendRepoURL) == "" {
			payload.BackendError = "请先在「更新配置」中填写「后端 GitHub 仓库」地址并保存"
		} else {
			payload.Backend, err = listSystemUpdateReleaseOptions(ctx, cfg.BackendRepoURL, cfg.BackendAssetPattern, cfg.GithubToken)
			if err != nil {
				payload.BackendError = err.Error()
			}
		}
	}
	return payload, nil
}

func (s SystemUpdateService) SaveConfig(ctx context.Context, input SystemUpdateConfigInput) (*SystemUpdateConfigView, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	defaults := defaultSystemUpdateConfig()
	now := time.Now()
	cfg.FrontendRepoURL = strings.TrimSpace(input.FrontendRepoURL)
	cfg.BackendRepoURL = strings.TrimSpace(input.BackendRepoURL)
	if strings.TrimSpace(input.GithubToken) != "" {
		cfg.GithubToken = strings.TrimSpace(input.GithubToken)
	} else if input.ClearGithubToken {
		cfg.GithubToken = ""
	}
	cfg.FrontendBranch = defaultIfBlank(input.FrontendBranch, defaults.FrontendBranch)
	cfg.BackendBranch = defaultIfBlank(input.BackendBranch, defaults.BackendBranch)
	cfg.FrontendReleaseTag = defaultIfBlank(input.FrontendReleaseTag, defaults.FrontendReleaseTag)
	cfg.BackendReleaseTag = defaultIfBlank(input.BackendReleaseTag, defaults.BackendReleaseTag)
	cfg.FrontendArtifactURL = strings.TrimSpace(input.FrontendArtifactURL)
	cfg.BackendArtifactURL = strings.TrimSpace(input.BackendArtifactURL)
	cfg.FrontendAssetPattern = defaultIfBlank(input.FrontendAssetPattern, defaults.FrontendAssetPattern)
	cfg.BackendAssetPattern = defaultIfBlank(input.BackendAssetPattern, defaults.BackendAssetPattern)
	cfg.FrontendInstallDir = defaultIfBlank(input.FrontendInstallDir, defaults.FrontendInstallDir)
	cfg.BackendInstallPath = defaultIfBlank(input.BackendInstallPath, defaults.BackendInstallPath)
	cfg.FrontendStartMode = normalizeFrontendStartMode(input.FrontendStartMode)
	cfg.FrontendSourceDir = strings.TrimSpace(input.FrontendSourceDir)
	cfg.BackendSourceDir = strings.TrimSpace(input.BackendSourceDir)
	cfg.ArtifactDir = defaultIfBlank(input.ArtifactDir, defaults.ArtifactDir)
	cfg.FrontendBuildCommand = strings.TrimSpace(input.FrontendBuildCommand)
	cfg.BackendBuildCommand = strings.TrimSpace(input.BackendBuildCommand)
	cfg.UpdatedAt = &now
	if cfg.ID == 0 {
		cfg.CreatedAt = now
		if err := s.db.WithContext(ctx).Create(&cfg).Error; err != nil {
			return nil, err
		}
		return ptr(systemUpdateConfigView(cfg)), nil
	}
	if err := s.db.WithContext(ctx).Save(&cfg).Error; err != nil {
		return nil, err
	}
	return ptr(systemUpdateConfigView(cfg)), nil
}

func (s SystemUpdateService) Run(ctx context.Context, input SystemUpdateRunInput) (*SystemUpdateJobView, error) {
	if _, err := s.loadConfig(ctx); err != nil {
		return nil, err
	}
	action := normalizeSystemUpdateAction(input.Action)
	if action == "ffmpeg_test" || action == "geoip_update" {
		return s.runComponentAction(ctx, action)
	}
	if !input.Frontend && !input.Backend {
		input.Frontend = true
		input.Backend = true
	}
	now := time.Now()
	job := domain.SystemUpdateJob{
		Action:        "deploy_artifacts",
		Status:        SystemUpdateStatusRunning,
		FrontendState: ternaryString(input.Frontend, SystemUpdateStatusPending, SystemUpdateStatusSkipped),
		BackendState:  ternaryString(input.Backend, SystemUpdateStatusPending, SystemUpdateStatusSkipped),
		Progress:      0,
		CurrentStep:   "等待下载产物",
		StartedAt:     &now,
		CreatedAt:     now,
		UpdatedAt:     &now,
	}
	if err := s.db.WithContext(ctx).Create(&job).Error; err != nil {
		return nil, err
	}
	go s.runJob(context.Background(), job.ID, input)
	view := systemUpdateJobView(job)
	return &view, nil
}

func (s SystemUpdateService) runComponentAction(ctx context.Context, action string) (*SystemUpdateJobView, error) {
	now := time.Now()
	step := "等待组件检查"
	if action == "geoip_update" {
		step = "等待 GeoIP 数据库更新"
	}
	job := domain.SystemUpdateJob{
		Action:        action,
		Status:        SystemUpdateStatusRunning,
		FrontendState: SystemUpdateStatusSkipped,
		BackendState:  SystemUpdateStatusSkipped,
		Progress:      0,
		CurrentStep:   step,
		StartedAt:     &now,
		CreatedAt:     now,
		UpdatedAt:     &now,
	}
	if err := s.db.WithContext(ctx).Create(&job).Error; err != nil {
		return nil, err
	}
	go s.runComponentJob(context.Background(), job.ID, action)
	view := systemUpdateJobView(job)
	return &view, nil
}

func (s SystemUpdateService) runJob(ctx context.Context, jobID int64, input SystemUpdateRunInput) {
	var job domain.SystemUpdateJob
	if err := s.db.WithContext(ctx).First(&job, jobID).Error; err != nil {
		return
	}
	runner := &systemUpdateRunner{service: s, job: job}
	defer func() {
		if recovered := recover(); recovered != nil {
			message := fmt.Sprintf("panic: %v", recovered)
			runner.log("执行异常: %s", message)
			runner.fail(ctx, message)
		}
	}()
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		runner.fail(ctx, err.Error())
		return
	}
	if strings.TrimSpace(input.FrontendReleaseTag) != "" {
		cfg.FrontendReleaseTag = strings.TrimSpace(input.FrontendReleaseTag)
	}
	if strings.TrimSpace(input.BackendReleaseTag) != "" {
		cfg.BackendReleaseTag = strings.TrimSpace(input.BackendReleaseTag)
	}
	runner.log("系统环境: %s/%s", runtime.GOOS, runtime.GOARCH)
	runner.log("下载缓存目录: %s", resolveSystemUpdatePath(cfg.ArtifactDir))
	runner.log("前端安装目录: %s", resolveSystemUpdatePath(cfg.FrontendInstallDir))
	runner.log("后端安装路径: %s", resolveSystemUpdatePath(cfg.BackendInstallPath))
	runner.log("前端 Release: %s", cfg.FrontendReleaseTag)
	runner.log("后端 Release: %s", cfg.BackendReleaseTag)
	runner.log("前端启动方式: %s", frontendStartCommand(cfg.FrontendStartMode))
	runner.update(ctx, 5, "准备下载产物")

	runErr := error(nil)
	if input.Frontend {
		runner.job.FrontendState = SystemUpdateStatusRunning
		runner.save(ctx)
		if meta, err := runner.deployFrontend(ctx, cfg, ternaryInt(input.Backend, 5, 10), ternaryInt(input.Backend, 45, 80)); err != nil {
			runner.job.FrontendState = SystemUpdateStatusFailed
			runErr = errors.Join(runErr, fmt.Errorf("frontend: %w", err))
		} else {
			runner.job.FrontendState = SystemUpdateStatusSucceeded
			runner.addArtifact(meta)
		}
	}
	if input.Backend {
		runner.job.BackendState = SystemUpdateStatusRunning
		runner.save(ctx)
		if meta, err := runner.deployBackend(ctx, cfg, ternaryInt(input.Frontend, 55, 10), ternaryInt(input.Frontend, 40, 80)); err != nil {
			runner.job.BackendState = SystemUpdateStatusFailed
			runErr = errors.Join(runErr, fmt.Errorf("backend: %w", err))
		} else {
			runner.job.BackendState = SystemUpdateStatusSucceeded
			runner.addArtifact(meta)
		}
	}
	if runErr != nil {
		runner.fail(ctx, runErr.Error())
		return
	}
	finished := time.Now()
	runner.job.Status = SystemUpdateStatusSucceeded
	runner.job.Progress = 100
	runner.job.CurrentStep = "产物已安装，等待手动重启容器"
	runner.job.FinishedAt = &finished
	runner.job.UpdatedAt = &finished
	runner.log("执行完成: 已下载并安装产物，请按容器环境手动重启服务")
	runner.save(ctx)
}

func (s SystemUpdateService) runComponentJob(ctx context.Context, jobID int64, action string) {
	var job domain.SystemUpdateJob
	if err := s.db.WithContext(ctx).First(&job, jobID).Error; err != nil {
		return
	}
	runner := &systemUpdateRunner{service: s, job: job}
	defer func() {
		if recovered := recover(); recovered != nil {
			message := fmt.Sprintf("panic: %v", recovered)
			runner.log("执行异常: %s", message)
			runner.fail(ctx, message)
		}
	}()
	switch action {
	case "ffmpeg_test":
		runner.update(ctx, 10, "运行 FFmpeg 测试")
		runner.log("FFmpeg 路径: %s", defaultIfBlank(s.cfg.Video.FFmpegPath, "ffmpeg"))
		runner.log("FFprobe 路径: %s", defaultIfBlank(s.cfg.Video.FFprobePath, "ffprobe"))
		checks := []SystemUpdateCheck{
			ffmpegExecutableCheck(ctx, "ffmpeg", s.cfg.Video.FFmpegPath, true),
			ffmpegExecutableCheck(ctx, "ffprobe", s.cfg.Video.FFprobePath, true),
		}
		failed := false
		for _, check := range checks {
			runner.log("%s: %s - %s", check.Label, check.Status, check.Message)
			if check.Path != "" {
				runner.log("%s 路径: %s", check.Label, check.Path)
			}
			if check.Status != SystemUpdateStatusSucceeded {
				failed = true
			}
		}
		if failed {
			runner.fail(ctx, "FFmpeg/FFprobe 测试失败")
			return
		}
		runner.succeed(ctx, "FFmpeg 测试完成", "FFmpeg/FFprobe 均可执行")
	case "geoip_update":
		runner.update(ctx, 10, "下载 Country.mmdb")
		meta, err := downloadGeoIPCountryMMDB(ctx, s.cfg.GeoIP, func(percent int) {
			runner.update(ctx, 10+(percent*80/100), fmt.Sprintf("下载 Country.mmdb %d%%", percent))
		})
		if err != nil {
			runner.fail(ctx, err.Error())
			return
		}
		runner.addArtifact(meta)
		runner.log("GeoIP 数据库已更新: %s, %s, sha256=%s", meta.InstallPath, formatSystemUpdateBytes(meta.SizeBytes), meta.SHA256)
		runner.succeed(ctx, "GeoIP 数据库已更新", "Country.mmdb 已替换")
	default:
		runner.fail(ctx, "unsupported component action: "+action)
	}
}

func (r *systemUpdateRunner) deployFrontend(ctx context.Context, cfg domain.SystemUpdateConfig, base int, span int) (SystemUpdateArtifactView, error) {
	source, err := resolveSystemUpdateArtifactSource(ctx, cfg, "frontend")
	if err != nil {
		return SystemUpdateArtifactView{}, err
	}
	r.log("前端产物: %s", source.Name)
	r.log("前端下载地址: %s", source.URL)
	r.log("仓库密钥: %s", ternaryString(strings.TrimSpace(cfg.GithubToken) != "", "已配置", "未配置"))
	r.update(ctx, base, "下载前端产物")
	meta, err := downloadSystemUpdateArtifact(ctx, source, cfg.ArtifactDir, cfg.GithubToken, func(percent int) {
		r.update(ctx, base+(percent*span/100), fmt.Sprintf("下载前端产物 %d%%", percent))
	})
	if err != nil {
		return SystemUpdateArtifactView{}, err
	}
	r.log("前端下载完成: %s, %s, sha256=%s", meta.CachePath, formatSystemUpdateBytes(meta.SizeBytes), meta.SHA256)
	if !isZipFile(meta.CachePath) {
		return SystemUpdateArtifactView{}, errors.New("前端产物必须是 zip 文件")
	}
	installDir := resolveSystemUpdatePath(cfg.FrontendInstallDir)
	if err := verifySystemUpdateInstallDir(installDir); err != nil {
		return SystemUpdateArtifactView{}, fmt.Errorf("安装目录校验失败: %s (当前值: %s)", err, installDir)
	}
	r.update(ctx, base+span, "解压前端产物")
	files, bytesWritten, err := unzipSystemUpdateArtifact(meta.CachePath, installDir)
	if err != nil {
		return SystemUpdateArtifactView{}, err
	}
	meta.InstallPath = installDir
	meta.InstalledAt = time.Now().Format(time.RFC3339)
	r.log("前端解压完成: %s (%d 个文件, %s)", installDir, files, formatSystemUpdateBytes(bytesWritten))
	r.log("前端启动命令: %s", frontendStartCommand(cfg.FrontendStartMode))
	return meta, nil
}

func (r *systemUpdateRunner) deployBackend(ctx context.Context, cfg domain.SystemUpdateConfig, base int, span int) (SystemUpdateArtifactView, error) {
	source, err := resolveSystemUpdateArtifactSource(ctx, cfg, "backend")
	if err != nil {
		return SystemUpdateArtifactView{}, err
	}
	r.log("后端产物: %s", source.Name)
	r.log("后端下载地址: %s", source.URL)
	r.log("仓库密钥: %s", ternaryString(strings.TrimSpace(cfg.GithubToken) != "", "已配置", "未配置"))
	installPath := resolveSystemUpdatePath(cfg.BackendInstallPath)

	if isZipName(source.Name) {
		r.update(ctx, base, "下载后端产物")
		meta, err := downloadSystemUpdateArtifact(ctx, source, cfg.ArtifactDir, cfg.GithubToken, func(percent int) {
			r.update(ctx, base+(percent*span/100), fmt.Sprintf("下载后端产物 %d%%", percent))
		})
		if err != nil {
			return SystemUpdateArtifactView{}, err
		}
		r.log("后端下载完成: %s, %s, sha256=%s", meta.CachePath, formatSystemUpdateBytes(meta.SizeBytes), meta.SHA256)
		r.update(ctx, base+span, "安装后端产物")
		if err := installBackendArtifact(meta.CachePath, installPath, cfg.BackendAssetPattern); err != nil {
			return SystemUpdateArtifactView{}, err
		}
		meta.InstallPath = installPath
		meta.InstalledAt = time.Now().Format(time.RFC3339)
		r.log("后端安装完成: %s", installPath)
		return meta, nil
	}

	r.update(ctx, base, "下载并安装后端产物")
	meta, err := streamSystemUpdateBinary(ctx, source, installPath, cfg.GithubToken, func(percent int) {
		r.update(ctx, base+(percent*span/100), fmt.Sprintf("下载后端产物 %d%%", percent))
	})
	if err != nil {
		return SystemUpdateArtifactView{}, err
	}
	r.log("后端安装完成: %s, %s, sha256=%s", installPath, formatSystemUpdateBytes(meta.SizeBytes), meta.SHA256)
	return meta, nil
}

func (r *systemUpdateRunner) addArtifact(meta SystemUpdateArtifactView) {
	r.artifacts = append(r.artifacts, meta)
	if meta.InstallPath != "" {
		r.paths = append(r.paths, meta.InstallPath)
		return
	}
	if meta.CachePath != "" {
		r.paths = append(r.paths, meta.CachePath)
	}
}

func (r *systemUpdateRunner) update(ctx context.Context, progress int, step string) {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	r.job.Progress = progress
	r.job.CurrentStep = step
	r.save(ctx)
}

func (r *systemUpdateRunner) log(format string, args ...any) {
	appendSystemUpdateLog(&r.logs, format, args...)
	r.job.Logs = trimSystemUpdateLogs(r.logs.String())
}

func (r *systemUpdateRunner) fail(ctx context.Context, message string) {
	now := time.Now()
	r.job.Status = SystemUpdateStatusFailed
	r.job.Progress = 100
	r.job.CurrentStep = "执行失败"
	r.job.ErrorMessage = &message
	r.job.FinishedAt = &now
	r.job.UpdatedAt = &now
	r.save(ctx)
}

func (r *systemUpdateRunner) succeed(ctx context.Context, step string, message string) {
	now := time.Now()
	r.job.Status = SystemUpdateStatusSucceeded
	r.job.Progress = 100
	r.job.CurrentStep = step
	r.job.FinishedAt = &now
	r.job.UpdatedAt = &now
	if strings.TrimSpace(message) != "" {
		r.log("%s", message)
	}
	r.save(ctx)
}

func (r *systemUpdateRunner) save(ctx context.Context) {
	pathsRaw, _ := json.Marshal(r.paths)
	artifactsRaw, _ := json.Marshal(r.artifacts)
	r.job.ArtifactPaths = pathsRaw
	r.job.ArtifactMeta = artifactsRaw
	r.job.Logs = trimSystemUpdateLogs(r.logs.String())
	now := time.Now()
	r.job.UpdatedAt = &now
	_ = r.service.db.WithContext(ctx).Save(&r.job).Error
}

func (s SystemUpdateService) loadConfig(ctx context.Context) (domain.SystemUpdateConfig, error) {
	var cfg domain.SystemUpdateConfig
	err := s.db.WithContext(ctx).Order("id ASC").First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return defaultSystemUpdateConfig(), nil
	}
	if err != nil {
		return cfg, err
	}
	fillSystemUpdateDefaults(&cfg)
	return cfg, nil
}

func (s SystemUpdateService) lastJob(ctx context.Context) (*SystemUpdateJobView, error) {
	var job domain.SystemUpdateJob
	err := s.db.WithContext(ctx).Order("created_at DESC, id DESC").First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	view := systemUpdateJobView(job)
	return &view, nil
}
