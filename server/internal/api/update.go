package api

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// RestartHook 由 main 注入：更新完成后用它把进程换成新二进制。
var RestartHook func()

const (
	defaultGatewayURL = "https://zupu-update.lijiacore.com:4440/jigong/latest.json"
	githubReleasesAPI = "https://api.github.com/repos/chunlong-li/jigong/releases/latest"
)

type updatePackage struct {
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type updateInfo struct {
	Version   string                    `json:"version"`
	Notes     string                    `json:"notes"`
	ReleaseAt string                    `json:"releaseAt"`
	Source    string                    `json:"source"`
	Packages  map[string]updatePackage  `json:"packages"`
}

func platformKey() string { return runtime.GOOS + "-" + runtime.GOARCH }

func updateSource() string {
	if v := strings.TrimSpace(os.Getenv("JIGONG_UPDATE_URL")); v != "" {
		return v
	}
	return defaultGatewayURL
}

// fetchGatewayJSON 读取自有网关/任意静态 latest.json。
// 兼容两种格式：{version,packages:{...}} 与 {version,url,size,sha256}（单平台）。
func fetchGatewayJSON(url string) (*updateInfo, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "jigong/"+Version)
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("更新源返回 %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	info := &updateInfo{Source: url, Packages: map[string]updatePackage{}}
	info.Version = strings.TrimSpace(fmt.Sprint(raw["version"]))
	info.Notes = strings.TrimSpace(fmt.Sprint(raw["notes"]))
	if info.Notes == "map[]" {
		info.Notes = ""
	}
	info.ReleaseAt = strings.TrimSpace(fmt.Sprint(raw["releaseAt"]))
	if info.ReleaseAt == "map[]" {
		info.ReleaseAt = ""
	}

	// 多平台格式
	if pk, ok := raw["packages"].(map[string]any); ok {
		for k, v := range pk {
			m, ok := v.(map[string]any)
			if !ok {
				continue
			}
			info.Packages[k] = updatePackage{
				URL:    strings.TrimSpace(fmt.Sprint(m["url"])),
				Size:   toInt64(m["size"]),
				SHA256: strings.ToLower(strings.TrimSpace(fmt.Sprint(m["sha256"]))),
			}
		}
	}
	// 单平台格式（随手写一个 latest.json 也能用）
	if len(info.Packages) == 0 && raw["url"] != nil {
		info.Packages[platformKey()] = updatePackage{
			URL:    strings.TrimSpace(fmt.Sprint(raw["url"])),
			Size:   toInt64(raw["size"]),
			SHA256: strings.ToLower(strings.TrimSpace(fmt.Sprint(raw["sha256"]))),
		}
	}
	if info.Version == "" {
		return nil, fmt.Errorf("latest.json 缺少 version")
	}
	return info, nil
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	}
	return 0
}

// fetchGitHubLatest 回落：直接读 GitHub Release。
func fetchGitHubLatest() (*updateInfo, error) {
	req, _ := http.NewRequest("GET", githubReleasesAPI, nil)
	req.Header.Set("User-Agent", "jigong/"+Version)
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub 返回 %d", resp.StatusCode)
	}
	var rel struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
			Size int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	info := &updateInfo{
		Version: strings.TrimSpace(rel.TagName),
		Notes:   rel.Body,
		Source:  "github",
		Packages: map[string]updatePackage{},
	}
	want := map[string]string{
		"linux-amd64":   "-linux-amd64.tar.gz",
		"linux-arm64":   "-linux-arm64.tar.gz",
		"windows-amd64": "-windows-amd64.zip",
	}
	for _, a := range rel.Assets {
		for k, suffix := range want {
			if strings.HasSuffix(a.Name, suffix) && !strings.Contains(a.Name, "compose") {
				if _, exists := info.Packages[k]; !exists {
					info.Packages[k] = updatePackage{URL: a.URL, Size: a.Size}
				}
			}
		}
	}
	return info, nil
}

func versionParts(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	segs := strings.Split(v, ".")
	out := make([]int, 0, len(segs))
	for _, s := range segs {
		n := 0
		for _, c := range s {
			if c < '0' || c > '9' {
				break
			}
			n = n*10 + int(c-'0')
		}
		out = append(out, n)
	}
	return out
}

func versionLess(a, b string) bool {
	pa, pb := versionParts(a), versionParts(b)
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			return x < y
		}
	}
	return false
}

// checkUpdate 查询更新；网关优先，失败回落 GitHub。
func (s *Server) checkUpdate(w http.ResponseWriter, r *http.Request) {
	var info *updateInfo
	var errs []string

	if info, err := fetchGatewayJSON(updateSource()); err == nil {
		writeJSON(w, 200, map[string]any{
			"current":    Version,
			"latest":     info.Version,
			"has_update": versionLess(Version, info.Version),
			"notes":      info.Notes,
			"release_at": info.ReleaseAt,
			"source":     info.Source,
			"package":    info.Packages[platformKey()],
			"platform":   platformKey(),
		})
		return
	} else {
		errs = append(errs, "网关: "+err.Error())
	}

	info, err := fetchGitHubLatest()
	if err != nil {
		errs = append(errs, "GitHub: "+err.Error())
		fail(w, 502, "检查更新失败（"+strings.Join(errs, "；")+"）")
		return
	}
	writeJSON(w, 200, map[string]any{
		"current":    Version,
		"latest":     info.Version,
		"has_update": versionLess(Version, info.Version),
		"notes":      info.Notes,
		"release_at": info.ReleaseAt,
		"source":     "github",
		"package":    info.Packages[platformKey()],
		"platform":   platformKey(),
	})
}

// applyUpdate 下载并安装更新：替换 data/runtime 下的二进制与前端，然后重启进程。
func (s *Server) applyUpdate(w http.ResponseWriter, r *http.Request) {
	info, err := fetchGatewayJSON(updateSource())
	if err != nil {
		info, err = fetchGitHubLatest()
	}
	if err != nil {
		fail(w, 502, "获取更新信息失败："+err.Error())
		return
	}
	if !versionLess(Version, info.Version) {
		fail(w, 400, "当前已是最新版本（"+Version+"）")
		return
	}
	pkg, ok := info.Packages[platformKey()]
	if !ok || pkg.URL == "" {
		fail(w, 400, "该平台（"+platformKey()+"）暂无更新包")
		return
	}

	tmpDir := filepath.Join(s.dataDir, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		fail(w, 500, "创建临时目录失败："+err.Error())
		return
	}
	archive := filepath.Join(tmpDir, "update.pkg")
	if err := downloadFile(pkg.URL, archive, pkg.Size, pkg.SHA256); err != nil {
		fail(w, 500, "下载更新包失败："+err.Error())
		return
	}
	defer os.Remove(archive)

	extractDir := filepath.Join(tmpDir, "extract")
	_ = os.RemoveAll(extractDir)
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		fail(w, 500, err.Error())
		return
	}
	if err := extractArchive(archive, extractDir); err != nil {
		fail(w, 500, "解压失败："+err.Error())
		return
	}
	newBin, newWeb, err := locatePayload(extractDir)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}

	runtimeDir := filepath.Join(s.dataDir, "runtime")
	if err := installUpdate(runtimeDir, newBin, newWeb); err != nil {
		fail(w, 500, "安装更新失败："+err.Error())
		return
	}

	writeJSON(w, 200, map[string]any{
		"status":  "ok",
		"version": info.Version,
		"message": "已更新到 " + info.Version + "，服务即将重启",
	})

	// 给响应一点时间，然后换进程
	go func() {
		time.Sleep(1200 * time.Millisecond)
		if RestartHook != nil {
			RestartHook()
		}
		os.Exit(0)
	}()
}

func downloadFile(url, dest string, wantSize int64, wantSHA string) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "jigong/"+Version)
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("下载返回 %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), resp.Body)
	if err != nil {
		return err
	}
	if wantSize > 0 && n != wantSize {
		return fmt.Errorf("大小不符：期望 %d，实际 %d", wantSize, n)
	}
	if wantSHA != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if !strings.EqualFold(got, wantSHA) {
			return fmt.Errorf("校验失败：sha256 不匹配")
		}
	}
	return nil
}

func extractArchive(archive, dest string) error {
	if strings.HasSuffix(archive, ".zip") {
		return extractZip(archive, dest)
	}
	return extractTarGz(archive, dest)
}

func extractTarGz(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(hdr.Name)
		if strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			continue
		}
		target := filepath.Join(dest, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
			_ = os.Chmod(target, os.FileMode(hdr.Mode)&0o777)
		}
	}
}

func extractZip(archive, dest string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, zf := range zr.File {
		name := filepath.Clean(zf.Name)
		if strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			continue
		}
		target := filepath.Join(dest, name)
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return err
		}
		out.Close()
		rc.Close()
	}
	return nil
}

// locatePayload 在解压目录里找二进制与 web 目录（支持带 jigong/ 前缀）。
func locatePayload(dir string) (bin string, web string, err error) {
	binName := "jigong"
	if runtime.GOOS == "windows" {
		binName = "jigong.exe"
	}
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if bin == "" && info.Mode().IsRegular() && info.Name() == binName {
			bin = path
			return nil
		}
		if web == "" && info.IsDir() && info.Name() == "web" {
			if _, e := os.Stat(filepath.Join(path, "index.html")); e == nil {
				web = path
			}
		}
		return nil
	})
	if bin == "" {
		return "", "", fmt.Errorf("更新包里没找到 %s", binName)
	}
	if web == "" {
		return "", "", fmt.Errorf("更新包里没找到 web/")
	}
	return bin, web, nil
}

// installUpdate 原子替换 <data>/runtime/{jigong,web}。
func installUpdate(runtimeDir, newBin, newWeb string) error {
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return err
	}
	targetBin := filepath.Join(runtimeDir, "jigong")
	if runtime.GOOS == "windows" {
		targetBin += ".exe"
	}
	tmpBin := targetBin + ".new"
	if err := copyFile(newBin, tmpBin); err != nil {
		return err
	}
	_ = os.Chmod(tmpBin, 0o755)

	targetWeb := filepath.Join(runtimeDir, "web")
	oldWeb := filepath.Join(runtimeDir, "web.old")
	_ = os.RemoveAll(oldWeb)
	if _, err := os.Stat(targetWeb); err == nil {
		if err := os.Rename(targetWeb, oldWeb); err != nil {
			return err
		}
	}
	if err := copyDir(newWeb, targetWeb); err != nil {
		_ = os.Rename(oldWeb, targetWeb)
		return err
	}
	_ = os.RemoveAll(oldWeb)

	if err := os.Rename(tmpBin, targetBin); err != nil {
		return err
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}
