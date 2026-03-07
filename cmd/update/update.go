package update

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
	"github.com/liweiming-nova/open_code/errors"
	"github.com/liweiming-nova/open_code/pkg/semver"
	"github.com/liweiming-nova/open_code/pkg/sum"
	"github.com/liweiming-nova/open_code/version"
	"github.com/pterm/pterm"
	ucli "github.com/urfave/cli/v3"
)

// UpdateCommand creates the update sub command.
func UpdateCommand() *ucli.Command {
	return &ucli.Command{
		Name:  "update",
		Usage: "检查更新并升级到最新版本",
		Action: func(ctx context.Context, cmd *ucli.Command) error {
			if err := godotenv.Load(); err != nil {
				fmt.Println("加载 .env 文件失败，请确认该文件存在")
				fmt.Println("可以先执行 generate env 生成示例配置")
				return nil
			}

			return SelfUpdate("liweiming-nova", "open_code")
		},
	}
}

func SelfUpdate(owner string, repo string) (err error) {
	up := NewUpdater()
	info := version.Get()

	latest, yes, err := up.CheckForUpdates(semver.MustNew(info.Version), owner, repo)
	if err != nil {
		return err
	}
	if !yes {
		pterm.Info.Printfln("当前已是最新版本: %s", info.Version)
		return nil
	}

	pterm.Info.Printfln("发现新版本: %s，正在下载更新...", latest.TagName)
	if err = up.Apply(latest, findAsset, findChecksum); err != nil {
		return err
	}
	pterm.Success.Printfln("版本升级成功，当前版本: %s", latest.TagName)
	return nil
}

func findAsset(items []Asset) (idx int) {
	ext := "zip"
	suffix := fmt.Sprintf("%s_%s.%s", CapitalizeOS(), GetNormalizedArch(), ext)
	for i := range items {
		if strings.HasSuffix(items[i].BrowserDownloadURL, suffix) {
			return i
		}
	}
	return -1
}

func findChecksum(items []Asset) (algo sum.Algorithm, expectedChecksum string, err error) {
	ext := "zip"
	suffix := fmt.Sprintf("%s_%s.%s", CapitalizeOS(), GetNormalizedArch(), ext)

	var checksumFileURL string
	for i := range items {
		if items[i].Name == "checksums.txt" {
			checksumFileURL = items[i].BrowserDownloadURL
			break
		}
	}
	if checksumFileURL == "" {
		return sum.SHA256, "", errors.ErrChecksumFileNotFound
	}

	resp, err := http.Get(checksumFileURL)
	if err != nil {
		return sum.SHA256, "", err
	}
	defer resp.Body.Close()

	if !IsHttpSuccess(resp.StatusCode) {
		return "", "", fmt.Errorf("URL %q is unreachable", checksumFileURL)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasSuffix(line, suffix) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		return sum.SHA256, fields[0], nil
	}
	if err = scanner.Err(); err != nil {
		return sum.SHA256, "", err
	}
	return sum.SHA256, "", errors.ErrChecksumFileNotFound
}

func CapitalizeOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "MacOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	default:
		return runtime.GOOS
	}
}

func GetNormalizedArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x64"
	case "386":
		return "x86"
	case "arm64":
		return "arm64"
	case "arm":
		return "arm"
	default:
		return runtime.GOARCH
	}
}
