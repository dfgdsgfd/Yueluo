package services

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

func resolveSystemUpdateArtifactSource(ctx context.Context, cfg domain.SystemUpdateConfig, kind string) (systemUpdateArtifactSource, error) {
	if kind == "frontend" {
		if strings.TrimSpace(cfg.FrontendArtifactURL) != "" {
			return directArtifactSource("frontend", cfg.FrontendArtifactURL, cfg.FrontendReleaseTag), nil
		}
		return githubReleaseAsset(ctx, cfg.FrontendRepoURL, cfg.FrontendReleaseTag, cfg.FrontendAssetPattern, cfg.GithubToken, "frontend")
	}
	if strings.TrimSpace(cfg.BackendArtifactURL) != "" {
		return directArtifactSource("backend", cfg.BackendArtifactURL, cfg.BackendReleaseTag), nil
	}
	return githubReleaseAsset(ctx, cfg.BackendRepoURL, cfg.BackendReleaseTag, cfg.BackendAssetPattern, cfg.GithubToken, "backend")
}

func directArtifactSource(kind, rawURL, tag string) systemUpdateArtifactSource {
	name := "artifact"
	if parsed, err := url.Parse(strings.TrimSpace(rawURL)); err == nil {
		if base := path.Base(parsed.Path); base != "." && base != "/" && base != "" {
			name = base
		}
	}
	return systemUpdateArtifactSource{Kind: kind, Name: name, URL: strings.TrimSpace(rawURL), ReleaseTag: normalizeReleaseTag(tag)}
}

func githubReleaseAsset(ctx context.Context, repoURL, releaseTag, assetPattern, token, kind string) (systemUpdateArtifactSource, error) {
	owner, repo, err := parseGitHubRepo(repoURL)
	if err != nil {
		return systemUpdateArtifactSource{}, err
	}
	tag := normalizeReleaseTag(releaseTag)
	release, err := fetchGitHubRelease(ctx, owner, repo, tag, token)
	if err != nil {
		return systemUpdateArtifactSource{}, err
	}
	pattern := strings.TrimSpace(assetPattern)
	names := make([]string, 0, len(release.Assets))
	for _, asset := range release.Assets {
		names = append(names, asset.Name)
		if wildcardMatch(pattern, asset.Name) {
			dlURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/assets/%d", owner, repo, asset.ID)
			return systemUpdateArtifactSource{
				Kind:            kind,
				Name:            asset.Name,
				URL:             dlURL,
				AssetID:         asset.ID,
				ReleaseTag:      defaultIfBlank(release.TagName, tag),
				SizeBytes:       asset.Size,
				GitHubUpdatedAt: asset.UpdatedAt,
			}, nil
		}
	}
	sort.Strings(names)
	return systemUpdateArtifactSource{}, fmt.Errorf("release %s has no asset matching %q; assets: %s", defaultIfBlank(release.TagName, tag), pattern, strings.Join(names, ", "))
}

func listSystemUpdateReleaseOptions(ctx context.Context, repoURL, assetPattern, token string) ([]SystemUpdateReleaseOption, error) {
	owner, repo, err := parseGitHubRepo(repoURL)
	if err != nil {
		return nil, err
	}
	releases, err := fetchGitHubReleases(ctx, owner, repo, token)
	if err != nil {
		return nil, err
	}
	options := make([]SystemUpdateReleaseOption, 0, len(releases))
	for _, release := range releases {
		hashes := fetchReleaseSHA256Sums(ctx, release.Assets, token)
		option := systemUpdateReleaseOption(release, assetPattern, hashes)
		if len(option.MatchingAssets) > 0 {
			options = append(options, option)
		}
	}
	return options, nil
}

func fetchGitHubRelease(ctx context.Context, owner, repo, tag, token string) (githubReleaseResponse, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	if tag != "latest" {
		endpoint = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, url.PathEscape(tag))
	}
	var release githubReleaseResponse
	if err := getGitHubJSON(ctx, endpoint, token, &release); err != nil {
		return release, err
	}
	return release, nil
}

func fetchGitHubReleases(ctx context.Context, owner, repo, token string) ([]githubReleaseResponse, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=30", owner, repo)
	var releases []githubReleaseResponse
	if err := getGitHubJSON(ctx, endpoint, token, &releases); err != nil {
		return nil, err
	}
	return releases, nil
}

func getGitHubJSON(ctx context.Context, endpoint, token string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		message := strings.TrimSpace(string(body))
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("GitHub Release 不存在或无权限访问，请先运行 GitHub Action 自动发布 Release，或确认仓库地址/访问密钥: %s", message)
		}
		if resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("GitHub API 返回 403，当前密钥无权访问该仓库。请确认: 1) Fine-grained PAT 需选择仓库并授予 Contents: Read-only 权限 2) Classic PAT 需 repo 或 public_repo 作用域 3) 密钥未过期。创建地址: https://github.com/settings/tokens。API 响应: %s", message)
		}
		return fmt.Errorf("github release lookup failed: %s %s", resp.Status, message)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func systemUpdateReleaseOption(release githubReleaseResponse, assetPattern string, hashes map[string]string) SystemUpdateReleaseOption {
	option := SystemUpdateReleaseOption{
		TagName:      release.TagName,
		Name:         release.Name,
		HTMLURL:      release.HTMLURL,
		TargetCommit: release.TargetCommit,
		CommitHash:   release.TargetCommit,
		CreatedAt:    release.CreatedAt,
		PublishedAt:  release.PublishedAt,
		Assets:       make([]SystemUpdateReleaseAssetOption, 0, len(release.Assets)),
	}
	for _, asset := range release.Assets {
		sha := assetSHA256(asset, hashes)
		item := SystemUpdateReleaseAssetOption{
			Name:        asset.Name,
			SizeBytes:   asset.Size,
			DownloadURL: asset.BrowserDownloadURL,
			UpdatedAt:   asset.UpdatedAt,
			SHA256:      sha,
			Matched:     wildcardMatch(assetPattern, asset.Name),
		}
		option.Assets = append(option.Assets, item)
		if item.Matched {
			option.MatchingAssets = append(option.MatchingAssets, item)
		}
	}
	return option
}

func fetchReleaseSHA256Sums(ctx context.Context, assets []githubReleaseAssetPayload, token string) map[string]string {
	out := map[string]string{}
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if !strings.HasPrefix(name, "sha256sums") || !strings.HasSuffix(name, ".txt") {
			continue
		}
		data, err := downloadSmallText(ctx, asset.BrowserDownloadURL, token, 256*1024)
		if err != nil {
			continue
		}
		maps.Copy(out, parseSHA256Sums(data))
	}
	return out
}

func downloadSmallText(ctx context.Context, rawURL string, token string, limit int64) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(token) != "" && isGitHubURL(rawURL) {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download checksum failed: %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func parseSHA256Sums(data string) map[string]string {
	out := map[string]string{}
	for line := range strings.SplitSeq(data, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		hash := strings.ToLower(strings.TrimSpace(fields[0]))
		if len(hash) != 64 {
			continue
		}
		name := strings.TrimPrefix(strings.TrimSpace(fields[1]), "*")
		if name != "" {
			out[name] = hash
		}
	}
	return out
}

func assetSHA256(asset githubReleaseAssetPayload, hashes map[string]string) string {
	if value := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(asset.Digest)), "sha256:"); len(value) == 64 {
		return value
	}
	if hashes == nil {
		return ""
	}
	if value := hashes[asset.Name]; value != "" {
		return value
	}
	return hashes[path.Base(asset.Name)]
}

func downloadSystemUpdateArtifact(ctx context.Context, source systemUpdateArtifactSource, artifactDir string, token string, onProgress func(percent int)) (SystemUpdateArtifactView, error) {
	cacheDir := resolveSystemUpdatePath(artifactDir)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return SystemUpdateArtifactView{}, err
	}
	name := safeArtifactFileName(source.Name)
	targetPath := filepath.Join(cacheDir, name)
	tmpPath := targetPath + ".download"
	file, err := os.Create(tmpPath)
	if err != nil {
		return SystemUpdateArtifactView{}, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	read, shaHex, err := downloadSystemUpdateBody(ctx, source, file, token, onProgress)
	if err != nil {
		_ = file.Close()
		closed = true
		_ = os.Remove(tmpPath)
		return SystemUpdateArtifactView{}, err
	}
	if err := file.Close(); err != nil {
		closed = true
		_ = os.Remove(tmpPath)
		return SystemUpdateArtifactView{}, err
	}
	closed = true
	_ = os.Remove(targetPath)
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return SystemUpdateArtifactView{}, err
	}
	if onProgress != nil {
		onProgress(100)
	}
	return SystemUpdateArtifactView{
		Kind:            source.Kind,
		Name:            source.Name,
		SourceURL:       source.URL,
		CachePath:       targetPath,
		SizeBytes:       read,
		SHA256:          shaHex,
		ReleaseTag:      source.ReleaseTag,
		GitHubUpdatedAt: source.GitHubUpdatedAt,
		DownloadedAt:    time.Now().Format(time.RFC3339),
	}, nil
}

func streamSystemUpdateBinary(ctx context.Context, source systemUpdateArtifactSource, installPath string, token string, onProgress func(percent int)) (SystemUpdateArtifactView, error) {
	installPath = resolveSystemUpdatePath(installPath)
	if err := os.MkdirAll(filepath.Dir(installPath), 0o755); err != nil {
		return SystemUpdateArtifactView{}, err
	}
	tmpPath := installPath + ".new"
	file, err := os.Create(tmpPath)
	if err != nil {
		return SystemUpdateArtifactView{}, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	read, shaHex, err := downloadSystemUpdateBody(ctx, source, file, token, onProgress)
	if err != nil {
		_ = file.Close()
		closed = true
		_ = os.Remove(tmpPath)
		return SystemUpdateArtifactView{}, err
	}
	if err := file.Close(); err != nil {
		closed = true
		_ = os.Remove(tmpPath)
		return SystemUpdateArtifactView{}, err
	}
	closed = true
	if runtime.GOOS != "windows" {
		_ = os.Chmod(tmpPath, 0o755)
	}
	_ = os.Remove(installPath)
	if err := os.Rename(tmpPath, installPath); err != nil {
		_ = os.Remove(tmpPath)
		return SystemUpdateArtifactView{}, err
	}
	if onProgress != nil {
		onProgress(100)
	}
	return SystemUpdateArtifactView{
		Kind:            source.Kind,
		Name:            source.Name,
		SourceURL:       source.URL,
		InstallPath:     installPath,
		SizeBytes:       read,
		SHA256:          shaHex,
		ReleaseTag:      source.ReleaseTag,
		GitHubUpdatedAt: source.GitHubUpdatedAt,
		DownloadedAt:    time.Now().Format(time.RFC3339),
		InstalledAt:     time.Now().Format(time.RFC3339),
	}, nil
}

func downloadSystemUpdateBody(ctx context.Context, source systemUpdateArtifactSource, writer io.Writer, token string, onProgress func(percent int)) (int64, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return 0, "", err
	}
	if source.AssetID > 0 {
		req.Header.Set("Accept", "application/octet-stream")
	}
	if strings.TrimSpace(token) != "" && isGitHubURL(source.URL) {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusNotFound {
			if strings.TrimSpace(token) == "" && isGitHubURL(source.URL) {
				return 0, "", fmt.Errorf("下载 %s 失败: 404 找不到文件，当前未配置仓库访问密钥，私有仓库需配置密钥才能下载", source.Name)
			}
			return 0, "", fmt.Errorf("下载 %s 失败: 404 找不到文件，请确认 Release 中存在该产物，或检查仓库访问密钥是否正确", source.Name)
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return 0, "", fmt.Errorf("下载 %s 失败: %s，请检查仓库访问密钥是否有效", source.Name, resp.Status)
		}
		return 0, "", fmt.Errorf("下载 %s 失败: %s", source.Name, resp.Status)
	}
	hash := sha256.New()
	total := resp.ContentLength
	if total <= 0 && source.SizeBytes > 0 {
		total = source.SizeBytes
	}
	multi := io.MultiWriter(writer, hash)
	buffer := make([]byte, 256*1024)
	chunkCount := 0
	read := int64(0)
	lastPercent := -1
	for {
		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			if _, err := multi.Write(chunk); err != nil {
				return read, hex.EncodeToString(hash.Sum(nil)), err
			}
			read += int64(n)
			chunkCount++
			if onProgress != nil {
				percent := 0
				if total > 0 {
					percent = min(int(read*100/total), 100)
				} else {
					percent = min(chunkCount*2, 95)
				}
				if percent != lastPercent && (percent == 100 || percent-lastPercent >= 2) {
					lastPercent = percent
					onProgress(percent)
				}
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return read, hex.EncodeToString(hash.Sum(nil)), readErr
		}
	}
	return read, hex.EncodeToString(hash.Sum(nil)), nil
}

func downloadGeoIPCountryMMDB(ctx context.Context, cfg config.GeoIPConfig, onProgress func(percent int)) (SystemUpdateArtifactView, error) {
	targetPath := resolveSystemUpdatePath(defaultIfBlank(cfg.CountryMMDBPath, "data/geoip/Country.mmdb"))
	sourceURL := defaultIfBlank(cfg.CountryMMDBURL, "https://github.com/Loyalsoldier/geoip/releases/latest/download/Country.mmdb")
	shaURL := defaultIfBlank(cfg.SHA256URL, "https://github.com/Loyalsoldier/geoip/releases/latest/download/Country.mmdb.sha256sum")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return SystemUpdateArtifactView{}, err
	}
	tmpPath := targetPath + ".download"
	file, err := os.Create(tmpPath)
	if err != nil {
		return SystemUpdateArtifactView{}, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	source := systemUpdateArtifactSource{Kind: "geoip", Name: "Country.mmdb", URL: sourceURL}
	read, shaHex, err := downloadSystemUpdateBody(ctx, source, file, "", onProgress)
	if err != nil {
		_ = file.Close()
		closed = true
		_ = os.Remove(tmpPath)
		return SystemUpdateArtifactView{}, err
	}
	if err := file.Close(); err != nil {
		closed = true
		_ = os.Remove(tmpPath)
		return SystemUpdateArtifactView{}, err
	}
	closed = true
	expected, err := downloadGeoIPSHA256(ctx, shaURL)
	if err != nil {
		_ = os.Remove(tmpPath)
		return SystemUpdateArtifactView{}, err
	}
	if expected != "" && !strings.EqualFold(expected, shaHex) {
		_ = os.Remove(tmpPath)
		return SystemUpdateArtifactView{}, fmt.Errorf("Country.mmdb sha256 mismatch: got %s want %s", shaHex, expected)
	}
	_ = os.Remove(targetPath)
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return SystemUpdateArtifactView{}, err
	}
	return SystemUpdateArtifactView{
		Kind:         "geoip",
		Name:         "Country.mmdb",
		SourceURL:    sourceURL,
		InstallPath:  targetPath,
		SizeBytes:    read,
		SHA256:       shaHex,
		DownloadedAt: time.Now().Format(time.RFC3339),
		InstalledAt:  time.Now().Format(time.RFC3339),
	}, nil
}

func downloadGeoIPSHA256(ctx context.Context, rawURL string) (string, error) {
	data, err := downloadSmallText(ctx, rawURL, "", 4096)
	if err != nil {
		return "", err
	}
	if value := parseNamedSHA256(data, "Country.mmdb"); value != "" {
		return value, nil
	}
	fields := strings.FieldsSeq(data)
	for field := range fields {
		field = strings.ToLower(strings.TrimSpace(field))
		if len(field) == 64 {
			return field, nil
		}
	}
	return "", errors.New("Country.mmdb.sha256sum 中未找到 sha256")
}

func parseNamedSHA256(data string, name string) string {
	hashes := parseSHA256Sums(data)
	if value := hashes[name]; value != "" {
		return value
	}
	return hashes[path.Base(name)]
}

func unzipSystemUpdateArtifact(zipPath, targetDir string) (int, int64, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, 0, err
	}
	defer reader.Close()
	stripPrefix := zipStripPrefix(reader.File)
	targetDir = resolveSystemUpdatePath(targetDir)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return 0, 0, err
	}
	files := 0
	bytesWritten := int64(0)
	for _, item := range reader.File {
		name := filepath.ToSlash(item.Name)
		if stripPrefix != "" {
			name = strings.TrimPrefix(name, stripPrefix)
		}
		name = strings.TrimPrefix(name, "/")
		if name == "" {
			continue
		}
		destination, err := safeZipDestination(targetDir, name)
		if err != nil {
			return files, bytesWritten, err
		}
		if item.FileInfo().IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				return files, bytesWritten, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return files, bytesWritten, err
		}
		source, err := item.Open()
		if err != nil {
			return files, bytesWritten, err
		}
		target, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, item.FileInfo().Mode())
		if err != nil {
			_ = source.Close()
			return files, bytesWritten, err
		}
		written, copyErr := io.Copy(target, source)
		closeErr := errors.Join(target.Close(), source.Close())
		if copyErr != nil || closeErr != nil {
			return files, bytesWritten, errors.Join(copyErr, closeErr)
		}
		files++
		bytesWritten += written
	}
	return files, bytesWritten, nil
}

func installBackendArtifact(cachePath, installPath, pattern string) error {
	source := cachePath
	cleanup := func() {}
	if isZipFile(cachePath) {
		tempDir, err := os.MkdirTemp(filepath.Dir(cachePath), "backend-artifact-*")
		if err != nil {
			return err
		}
		cleanup = func() { _ = os.RemoveAll(tempDir) }
		defer cleanup()
		if _, _, err := unzipSystemUpdateArtifact(cachePath, tempDir); err != nil {
			return err
		}
		candidate, err := findBackendBinary(tempDir, pattern)
		if err != nil {
			return err
		}
		source = candidate
	}
	installPath = resolveSystemUpdatePath(installPath)
	if err := os.MkdirAll(filepath.Dir(installPath), 0o755); err != nil {
		return err
	}
	tmpPath := installPath + ".new"
	if err := copyFile(source, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(tmpPath, 0o755)
	}
	_ = os.Remove(installPath)
	if err := os.Rename(tmpPath, installPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func findBackendBinary(root, pattern string) (string, error) {
	matches := []string{}
	fallbacks := []string{}
	err := filepath.WalkDir(root, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if wildcardMatch(pattern, name) {
			matches = append(matches, current)
		}
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "yuem-go-") || lower == "yuem-go" || lower == "yuem-go.exe" {
			fallbacks = append(fallbacks, current)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) > 0 {
		sort.Strings(matches)
		return matches[0], nil
	}
	if len(fallbacks) > 0 {
		sort.Strings(fallbacks)
		return fallbacks[0], nil
	}
	return "", errors.New("backend binary not found in zip artifact")
}

func copyFile(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	return errors.Join(copyErr, closeErr)
}
