package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/liweiming-nova/open_code/errors"
	"github.com/liweiming-nova/open_code/pkg/downloader"
	"github.com/liweiming-nova/open_code/pkg/semver"
	"github.com/liweiming-nova/open_code/pkg/sum"
	"github.com/liweiming-nova/open_code/pkg/zip"
	"github.com/wsshow/selfupdate"
)

// Release represents a software version release.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a downloadable asset in a release.
type Asset struct {
	Name               string `json:"name"`
	ContentType        string `json:"content_type"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// IsCompressedFile checks if the asset is a compressed file.
func (a Asset) IsCompressedFile() bool {
	return a.ContentType == "application/zip" || a.ContentType == "application/x-gzip"
}

type Updater struct{}

// NewUpdater creates a new Updater instance.
func NewUpdater() *Updater {
	return new(Updater)
}

// CheckForUpdates verifies if newer version exists.
func (up Updater) CheckForUpdates(current *semver.Version, owner, repo string) (rel *Release, yes bool, err error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if !IsHttpSuccess(resp.StatusCode) {
		return nil, false, fmt.Errorf("URL %q is unreachable", url)
	}

	var latest Release
	if err = json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		return nil, false, err
	}

	latestVersion := semver.New(latest.TagName)
	if latestVersion == nil {
		return nil, false, err
	}
	if latestVersion.GreaterThan(current) {
		return &latest, true, nil
	}
	return nil, false, nil
}

// Apply performs version update to specified release.
func (up Updater) Apply(rel *Release,
	findAsset func([]Asset) (idx int),
	findChecksum func([]Asset) (algo sum.Algorithm, expectedChecksum string, err error),
) error {
	idx := findAsset(rel.Assets)
	if idx < 0 {
		return errors.ErrAssetNotFound
	}

	algo, expectedChecksum, err := findChecksum(rel.Assets)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", strconv.FormatInt(time.Now().UnixNano(), 10))
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	downloadURL := rel.Assets[idx].BrowserDownloadURL
	srcFilename := filepath.Join(tmpDir, deriveFilename(downloadURL))
	dstFilename := srcFilename

	proxyStr := os.Getenv("FEIKONG_PROXY_URL")
	var proxyFunc func(*http.Request) (*url.URL, error)
	if proxyStr != "" {
		proxyURL, parseErr := url.Parse(proxyStr)
		if parseErr != nil {
			return fmt.Errorf("invalid FEIKONG_PROXY_URL: %w", parseErr)
		}
		proxyFunc = http.ProxyURL(proxyURL)
	} else {
		proxyFunc = http.ProxyFromEnvironment
	}

	transport := &http.Transport{
		Proxy:                 proxyFunc,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Minute,
	}

	dl := downloader.New(downloadURL, srcFilename, httpClient)
	var lastProgress float64 = -1
	dl.OnProgress(func(loaded, total int64, rate string) {
		if total <= 0 {
			fmt.Printf("\r已下载 %s | %s    ", formatFileSize(float64(loaded)), rate)
			return
		}

		progress := float64(loaded) / float64(total) * 100
		if lastProgress >= 0 && progress-lastProgress < 0.5 && progress < 100 {
			return
		}
		lastProgress = progress

		barWidth := 40
		filledWidth := int(progress / 100 * float64(barWidth))
		if filledWidth < 0 {
			filledWidth = 0
		}
		if filledWidth > barWidth {
			filledWidth = barWidth
		}
		bar := strings.Repeat("#", filledWidth) + strings.Repeat("-", barWidth-filledWidth)
		fmt.Printf("\r[%s] %.2f%% | %s/%s | %s    ",
			bar, progress, formatFileSize(float64(loaded)), formatFileSize(float64(total)), rate)
	})

	if err := dl.Start(); err != nil {
		fmt.Printf("下载失败: %v\n", err)
		return err
	}
	fmt.Println()

	fmt.Printf("基于 %s 校验文件完整性...\n", algo)
	if err = sum.VerifyFile(algo, expectedChecksum, srcFilename); err != nil {
		return err
	}
	fmt.Println("文件完整性校验通过")

	if rel.Assets[idx].IsCompressedFile() {
		if dstFilename, err = up.unarchive(srcFilename, tmpDir); err != nil {
			return err
		}
	}

	dstFile, err := os.Open(dstFilename)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	return selfupdate.Apply(dstFile, selfupdate.Options{})
}

// unarchive extracts compressed files to target directory and returns first extracted file.
func (up Updater) unarchive(srcFile, dstDir string) (dstFile string, err error) {
	if err = zip.Unzip(srcFile, dstDir, func(processed, total int, fileName string, isDir bool) {
		fmt.Printf("解压中... %d/%d 文件: %s\n", processed, total, fileName)
	}); err != nil {
		return "", err
	}

	fis, _ := os.ReadDir(dstDir)
	for _, fi := range fis {
		if strings.HasSuffix(fi.Name(), ".md") ||
			strings.HasSuffix(fi.Name(), ".zip") ||
			strings.HasSuffix(fi.Name(), "LICENSE") {
			continue
		}
		return filepath.Join(dstDir, fi.Name()), nil
	}
	return "", nil
}

// IsHttpSuccess determines if the HTTP status code indicates successful response.
func IsHttpSuccess(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
}

func deriveFilename(downloadURL string) string {
	if parsed, err := url.Parse(downloadURL); err == nil {
		name := filepath.Base(parsed.Path)
		if name != "" && name != "." && name != "/" {
			return name
		}
	}
	name := filepath.Base(downloadURL)
	if name == "" || name == "." || name == "/" {
		return "download.bin"
	}
	return name
}

// formatFileSize converts file size in bytes to human-readable string.
func formatFileSize(fileSize float64) string {
	const (
		KB = 1024.0
		MB = KB * 1024.0
		GB = MB * 1024.0
	)

	switch {
	case fileSize >= GB:
		return fmt.Sprintf("%.2f GB", fileSize/GB)
	case fileSize >= MB:
		return fmt.Sprintf("%.2f MB", fileSize/MB)
	case fileSize >= KB:
		return fmt.Sprintf("%.2f KB", fileSize/KB)
	default:
		return fmt.Sprintf("%.2f B", fileSize)
	}
}
