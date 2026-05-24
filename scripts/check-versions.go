// check-versions.go 校验 apps 仓库 yaml 中声明的镜像版本是否真实存在
//
// 用法：
//
//	go run scripts/check-versions.go [flags]
//
// Flags：
//
//	--apps-dir <path>   apps 根目录，默认 "."（脚本同级的仓库根目录）
//	--app <key>         只校验指定 app
//	--arch <arch>       只校验指定架构（amd64/arm64）
//	--strict            严格模式：对 deprecated 也校验，超时更短
//	--timeout <sec>     单个 manifest 请求超时，默认 30s
//
// 设计参见 .kiro/specs/app-image-version-reliability/design.md §3.1-3.2。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"gopkg.in/yaml.v3"
)

// ---------- 数据结构 ----------

// appYAML 直接对应 apps-repo 中的 app.yaml，仅声明本工具关心的字段。
type appYAML struct {
	Key            string    `yaml:"key"`
	Name           string    `yaml:"name"`
	AppType        string    `yaml:"app_type"`
	DefaultVersion string    `yaml:"default_version"`
	Architectures  []string  `yaml:"architectures"`
	Versions       yaml.Node `yaml:"versions"`
}

// versionEntry 对应 versions[] 数组的一项（兼容字符串与对象）。
type versionEntry struct {
	Display       string   `yaml:"display"`
	Tag           string   `yaml:"tag"`
	Digest        string   `yaml:"digest"`
	Image         string   `yaml:"image"`
	Architectures []string `yaml:"architectures"`
	Rolling       bool     `yaml:"rolling"`
	Deprecated    bool     `yaml:"deprecated"`
	DownloadURL   string   `yaml:"download_url"`
	FallbackURL   string   `yaml:"fallback_url"`
	SHA256        string   `yaml:"sha256"`

	// legacy=true 表示该条目来自旧的字符串数组格式（versions: ["8.0"]）。
	// 此时 Tag = Display 是占位值，不是真实 Docker Hub tag，无从校验，统一跳过。
	legacy bool `yaml:"-"`
}

// validationResult 单次校验结果。
type validationResult struct {
	AppKey   string
	YAMLPath string // 相对仓库根目录的路径，便于 GitHub Actions annotation
	Display  string
	Image    string // 完整 ref（含 tag），仅供日志展示
	Arch     string
	Status   string // ok / fail / skip
	Reason   string
	Warning  string // 非阻塞警告（例如 deprecated 非 strict 模式跳过）
}

// ---------- 命令行参数 ----------

type cliFlags struct {
	appsDir string
	appKey  string
	arch    string
	strict  bool
	timeout time.Duration
}

func parseFlags() cliFlags {
	var f cliFlags
	var timeoutSec int
	flag.StringVar(&f.appsDir, "apps-dir", ".", "apps 根目录路径")
	flag.StringVar(&f.appKey, "app", "", "只校验指定 app（key 名）")
	flag.StringVar(&f.arch, "arch", "", "只校验指定架构（amd64/arm64），留空表示全部")
	flag.BoolVar(&f.strict, "strict", false, "严格模式：对 deprecated 版本也校验，超时更短")
	flag.IntVar(&timeoutSec, "timeout", 30, "单个 manifest 请求超时（秒）")
	flag.Parse()

	if f.strict && timeoutSec > 15 {
		timeoutSec = 15
	}
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	f.timeout = time.Duration(timeoutSec) * time.Second
	return f
}

// ---------- 入口 ----------

func main() {
	flags := parseFlags()

	rootAbs, err := filepath.Abs(flags.appsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析 apps-dir 失败: %v\n", err)
		os.Exit(2)
	}

	yamlPaths, err := discoverAppYAMLs(rootAbs, flags.appKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "扫描 app.yaml 失败: %v\n", err)
		os.Exit(2)
	}
	if len(yamlPaths) == 0 {
		fmt.Println("未发现任何 app.yaml")
		return
	}

	// 解析所有 yaml，构造校验任务。
	var tasks []validationTask
	for _, p := range yamlPaths {
		ts, err := buildTasksForYAML(rootAbs, p, flags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "解析 %s 失败: %v\n", p, err)
			// 解析失败也算一次失败，便于 CI 阻塞
			tasks = append(tasks, validationTask{
				appKey:   filepath.Base(filepath.Dir(p)),
				yamlPath: relPath(rootAbs, p),
				display:  "-",
				kind:     taskKindError,
				reason:   err.Error(),
			})
			continue
		}
		tasks = append(tasks, ts...)
	}

	// 排序，让输出可预测便于 review。
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].appKey != tasks[j].appKey {
			return tasks[i].appKey < tasks[j].appKey
		}
		if tasks[i].display != tasks[j].display {
			return tasks[i].display < tasks[j].display
		}
		return tasks[i].arch < tasks[j].arch
	})

	results := runTasks(tasks, flags)

	// 输出可读结果 + GitHub annotations。
	printResults(results)
	if isGitHubActions() {
		emitGitHubAnnotations(results)
	}

	// 计算退出码。
	hasFail := false
	for _, r := range results {
		if r.Status == "fail" {
			hasFail = true
			break
		}
	}
	if hasFail {
		os.Exit(1)
	}
}

// ---------- 扫描 yaml ----------

// 跳过的目录名（位于 apps 仓库根目录直接子目录时跳过）。
var skipDirs = map[string]bool{
	"scripts":      true,
	".github":      true,
	".git":         true,
	"node_modules": true,
}

func discoverAppYAMLs(root string, only string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if skipDirs[e.Name()] {
			continue
		}
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if only != "" && e.Name() != only {
			continue
		}
		yamlPath := filepath.Join(root, e.Name(), "app.yaml")
		if _, err := os.Stat(yamlPath); err == nil {
			paths = append(paths, yamlPath)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

// ---------- 解析 yaml & 构造任务 ----------

// taskKind 区分不同形态的校验任务。
type taskKind int

const (
	taskKindManifest    taskKind = iota // 普通容器镜像 manifest 校验
	taskKindRolling                     // rolling 仅校验 image 名可解析
	taskKindCMS                         // CMS 类 HTTP HEAD 校验
	taskKindSkip                        // 直接跳过（带原因）
	taskKindError                       // 解析或前置错误，直接计为 fail
	taskKindDeprecated                  // deprecated 跳过（非 strict 时）
)

type validationTask struct {
	appKey   string
	yamlPath string // 相对仓库根目录路径
	display  string
	tag      string
	digest   string
	image    string // 已替换占位符的完整 ref（image:tag），用于 manifest 校验
	imageRaw string // 不含 tag 的 image 名
	arch     string
	kind     taskKind
	urls     []string // CMS 任务用：download_url + fallback_url
	reason   string   // 跳过/错误时的原因
}

func buildTasksForYAML(root, yamlPath string, flags cliFlags) ([]validationTask, error) {
	rel := relPath(root, yamlPath)
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("读取 yaml: %w", err)
	}

	var app appYAML
	if err := yaml.Unmarshal(data, &app); err != nil {
		return nil, fmt.Errorf("yaml 解析: %w", err)
	}

	entries, err := parseVersionEntries(&app)
	if err != nil {
		return nil, err
	}

	// docker-compose.yml 中第一个 image 行（用于旧字符串格式回填 image 主名）。
	composeImage := readComposeImage(filepath.Join(filepath.Dir(yamlPath), "docker-compose.yml"))

	appKey := app.Key
	if appKey == "" {
		appKey = filepath.Base(filepath.Dir(yamlPath))
	}

	var tasks []validationTask
	for _, entry := range entries {
		// 决定 image 主名：entry.Image > compose 中的首条 image。
		imageRaw := entry.Image
		if imageRaw == "" {
			imageRaw = composeImage
		}

		// CMS 类：走 HTTP HEAD。
		if strings.EqualFold(app.AppType, "cms") {
			urls := []string{}
			if entry.DownloadURL != "" {
				urls = append(urls, entry.DownloadURL)
			}
			if entry.FallbackURL != "" {
				urls = append(urls, entry.FallbackURL)
			}
			if len(urls) == 0 {
				// 没有显式 URL，跳过（保持兼容 wordpress 隐式 URL 模板）。
				tasks = append(tasks, validationTask{
					appKey:   appKey,
					yamlPath: rel,
					display:  entry.Display,
					arch:     "-",
					kind:     taskKindSkip,
					reason:   "cms 应用未声明 download_url，跳过 HTTP 校验",
				})
				continue
			}
			tasks = append(tasks, validationTask{
				appKey:   appKey,
				yamlPath: rel,
				display:  entry.Display,
				arch:     "-",
				kind:     taskKindCMS,
				urls:     urls,
			})
			continue
		}

		// 容器类：必须有 image 主名，否则只能 skip。
		if imageRaw == "" {
			tasks = append(tasks, validationTask{
				appKey:   appKey,
				yamlPath: rel,
				display:  entry.Display,
				arch:     "-",
				kind:     taskKindSkip,
				reason:   "yaml 未声明 image 字段，且 docker-compose.yml 未提取到 image",
			})
			continue
		}

		// 旧字符串格式（versions: ["8.0"]）：Tag 是占位值，不是真实 Docker Hub tag，
		// 无从校验，跳过并发警告，等 Task 8 升级 yaml 后再校验。
		if entry.legacy {
			tasks = append(tasks, validationTask{
				appKey:   appKey,
				yamlPath: rel,
				display:  entry.Display,
				arch:     "-",
				kind:     taskKindSkip,
				reason:   "旧字符串格式（无 image/tag 字段），跳过 manifest 校验",
			})
			continue
		}

		// deprecated 非 strict 模式跳过。
		if entry.Deprecated && !flags.strict {
			tasks = append(tasks, validationTask{
				appKey:   appKey,
				yamlPath: rel,
				display:  entry.Display,
				arch:     "-",
				kind:     taskKindDeprecated,
				reason:   "deprecated 版本，非 strict 模式跳过",
			})
			continue
		}

		// 拼出 ref：占位符替换成 entry.Tag/entry.Display。
		// 旧字符串格式时 tag=display，且 image 取自 compose（含占位符），同样能正确替换。
		ref := buildImageRef(imageRaw, entry)

		// rolling：只校验 ref 可解析。
		if entry.Rolling {
			tasks = append(tasks, validationTask{
				appKey:   appKey,
				yamlPath: rel,
				display:  entry.Display,
				image:    ref,
				imageRaw: imageRaw,
				arch:     "-",
				kind:     taskKindRolling,
			})
			continue
		}

		// 多架构：entry > app > [amd64]。
		archs := pickArchitectures(entry.Architectures, app.Architectures)
		if flags.arch != "" {
			// CLI 限定单架构：取交集。
			matched := []string{}
			for _, a := range archs {
				if a == flags.arch {
					matched = append(matched, a)
				}
			}
			if len(matched) == 0 {
				// 该版本不支持指定架构，跳过。
				tasks = append(tasks, validationTask{
					appKey:   appKey,
					yamlPath: rel,
					display:  entry.Display,
					arch:     flags.arch,
					kind:     taskKindSkip,
					reason:   fmt.Sprintf("版本 %s 不支持架构 %s，跳过", entry.Display, flags.arch),
				})
				continue
			}
			archs = matched
		}

		for _, arch := range archs {
			tasks = append(tasks, validationTask{
				appKey:   appKey,
				yamlPath: rel,
				display:  entry.Display,
				tag:      entry.Tag,
				digest:   entry.Digest,
				image:    ref,
				imageRaw: imageRaw,
				arch:     arch,
				kind:     taskKindManifest,
			})
		}
	}

	return tasks, nil
}

// parseVersionEntries 兼容 versions 字段的字符串数组与对象数组形态。
func parseVersionEntries(app *appYAML) ([]versionEntry, error) {
	if app.Versions.Kind == 0 {
		// 字段为空（旧 yaml 可能完全没声明），返回空列表。
		return nil, nil
	}
	if app.Versions.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("versions 必须是数组")
	}
	out := make([]versionEntry, 0, len(app.Versions.Content))
	for i, n := range app.Versions.Content {
		switch n.Kind {
		case yaml.ScalarNode:
			v := strings.TrimSpace(n.Value)
			if v == "" {
				return nil, fmt.Errorf("versions[%d] 是空字符串", i)
			}
			out = append(out, versionEntry{Display: v, Tag: v, legacy: true})
		case yaml.MappingNode:
			var e versionEntry
			if err := n.Decode(&e); err != nil {
				return nil, fmt.Errorf("versions[%d] 解析失败: %w", i, err)
			}
			if e.Display == "" {
				return nil, fmt.Errorf("versions[%d] 缺少 display 字段", i)
			}
			if e.Tag == "" {
				e.Tag = e.Display
			}
			out = append(out, e)
		default:
			return nil, fmt.Errorf("versions[%d] 必须是字符串或对象", i)
		}
	}
	return out, nil
}

// composeImageRegexp 匹配 docker-compose.yml 第一条 image 行。
var composeImageRegexp = regexp.MustCompile(`(?m)^\s*image:\s*([^\s#]+)`)

func readComposeImage(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	m := composeImageRegexp.FindStringSubmatch(string(data))
	if len(m) < 2 {
		return ""
	}
	raw := strings.Trim(m[1], `"'`)
	// 去除 ":tag" 部分（保留模板占位符 image，方便后续替换 tag）。
	// raw 可能形如 "openresty/openresty:{{tag}}" 或 "openresty/openresty:{{version}}-alpine"。
	// 这里只截 "image" 主名（第一个冒号之前的部分）。
	if idx := strings.Index(raw, ":"); idx >= 0 {
		raw = raw[:idx]
	}
	return strings.TrimSpace(raw)
}

// buildImageRef 替换占位符并拼接为完整 ref。
// imageRaw 可能是干净的 image 主名（如 "openresty/openresty"），也可能含模板占位符
// （如 docker-compose.yml 中 "openresty/openresty:{{tag}}-alpine" 已被截断到主名时无尾巴）。
// 由于 readComposeImage 只截到第一个冒号前，这里只需要用 entry.Tag 拼。
// 如果 entry.Image 显式声明了完整 image 主名（不含 tag），同样直接拼。
func buildImageRef(imageRaw string, entry versionEntry) string {
	tag := entry.Tag
	if tag == "" {
		tag = entry.Display
	}
	// 替换罕见的 {{tag}} / {{version}} 占位符（容错：理论上 imageRaw 里不应有它们）。
	imageRaw = strings.ReplaceAll(imageRaw, "{{tag}}", tag)
	imageRaw = strings.ReplaceAll(imageRaw, "{{version}}", entry.Display)
	return imageRaw + ":" + tag
}

func pickArchitectures(entryArchs, appArchs []string) []string {
	if len(entryArchs) > 0 {
		return entryArchs
	}
	if len(appArchs) > 0 {
		return appArchs
	}
	return []string{"amd64"}
}

// ---------- 执行校验 ----------

func runTasks(tasks []validationTask, flags cliFlags) []validationResult {
	const maxConcurrent = 8
	sem := make(chan struct{}, maxConcurrent)

	results := make([]validationResult, len(tasks))
	var wg sync.WaitGroup

	for i, task := range tasks {
		i, task := i, task
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = executeTask(task, flags)
		}()
	}
	wg.Wait()

	// 保持已排序的顺序（tasks 已在 main 中排序）。
	return results
}

func executeTask(task validationTask, flags cliFlags) validationResult {
	res := validationResult{
		AppKey:   task.appKey,
		YAMLPath: task.yamlPath,
		Display:  task.display,
		Image:    task.image,
		Arch:     task.arch,
	}

	switch task.kind {
	case taskKindError:
		res.Status = "fail"
		res.Reason = task.reason
		return res

	case taskKindSkip:
		res.Status = "skip"
		res.Reason = task.reason
		return res

	case taskKindDeprecated:
		res.Status = "skip"
		res.Reason = task.reason
		res.Warning = fmt.Sprintf("%s 版本 %s 标记为 deprecated", task.appKey, task.display)
		return res

	case taskKindRolling:
		// 仅校验 ref 可解析。
		if _, err := name.ParseReference(task.image); err != nil {
			res.Status = "fail"
			res.Reason = fmt.Sprintf("rolling 镜像引用解析失败: %v", err)
			return res
		}
		res.Status = "ok"
		res.Reason = "rolling 跳过 manifest 校验"
		return res

	case taskKindCMS:
		ctx, cancel := context.WithTimeout(context.Background(), flags.timeout)
		defer cancel()
		if err := headAny(ctx, task.urls); err != nil {
			res.Status = "fail"
			res.Reason = fmt.Sprintf("CMS HTTP HEAD 失败: %v", err)
			return res
		}
		res.Status = "ok"
		return res

	case taskKindManifest:
		ctx, cancel := context.WithTimeout(context.Background(), flags.timeout)
		defer cancel()
		ref, err := name.ParseReference(task.image)
		if err != nil {
			res.Status = "fail"
			res.Reason = fmt.Sprintf("ref 解析失败: %v", err)
			return res
		}
		platform := &v1.Platform{OS: "linux", Architecture: task.arch}
		if _, err := crane.Manifest(ref.Name(), crane.WithContext(ctx), crane.WithPlatform(platform)); err != nil {
			res.Status = "fail"
			res.Reason = fmt.Sprintf("manifest 拉取失败: %v", err)
			return res
		}
		// digest 校验。
		if task.digest != "" {
			gotDigest, err := crane.Digest(ref.Name(), crane.WithContext(ctx), crane.WithPlatform(platform))
			if err != nil {
				res.Status = "fail"
				res.Reason = fmt.Sprintf("拉取 digest 失败: %v", err)
				return res
			}
			if !strings.EqualFold(gotDigest, task.digest) {
				res.Status = "fail"
				res.Reason = fmt.Sprintf("digest_mismatch: 声明=%s 实际=%s", task.digest, gotDigest)
				return res
			}
		}
		res.Status = "ok"
		return res
	}

	res.Status = "fail"
	res.Reason = "未知任务类型"
	return res
}

// headAny 依次尝试 url 列表，任意一个返回 2xx 视为通过。
func headAny(ctx context.Context, urls []string) error {
	client := &http.Client{Timeout: 0} // 由 ctx 控制超时
	var lastErr error
	for _, u := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("HEAD %s -> %d", u, resp.StatusCode)
	}
	if lastErr == nil {
		lastErr = errors.New("无可用 URL")
	}
	return lastErr
}

// ---------- 输出 ----------

func printResults(results []validationResult) {
	var pass, fail, skip int
	var warn int
	fmt.Println()
	fmt.Println("==================== 校验明细 ====================")
	for _, r := range results {
		marker := "  "
		switch r.Status {
		case "ok":
			marker = "✓ "
			pass++
		case "fail":
			marker = "✗ "
			fail++
		case "skip":
			marker = "- "
			skip++
		}
		archStr := ""
		if r.Arch != "" && r.Arch != "-" {
			archStr = "/" + r.Arch
		}
		line := fmt.Sprintf("%s%s %-12s %-12s%s", marker, r.YAMLPath, r.AppKey, r.Display, archStr)
		if r.Image != "" {
			line += "  " + r.Image
		}
		if r.Reason != "" {
			line += "  -- " + r.Reason
		}
		fmt.Println(line)
		if r.Warning != "" {
			warn++
		}
	}
	fmt.Println("=================================================")
	fmt.Printf("汇总: pass=%d  fail=%d  skip=%d  warn=%d\n\n", pass, fail, skip, warn)
}

// emitGitHubAnnotations 在 GitHub Actions 环境下输出 ::error / ::warning。
func emitGitHubAnnotations(results []validationResult) {
	for _, r := range results {
		if r.Status == "fail" {
			msg := fmt.Sprintf("%s 版本 %s", r.AppKey, r.Display)
			if r.Arch != "" && r.Arch != "-" {
				msg += "/" + r.Arch
			}
			msg += " 校验失败: " + r.Reason
			fmt.Printf("::error file=%s,line=1::%s\n", normalizeAnnotationPath(r.YAMLPath), msg)
		}
		if r.Warning != "" {
			fmt.Printf("::warning file=%s::%s\n", normalizeAnnotationPath(r.YAMLPath), r.Warning)
		}
	}
}

func isGitHubActions() bool {
	return strings.EqualFold(os.Getenv("GITHUB_ACTIONS"), "true")
}

// ---------- 工具函数 ----------

func relPath(root, target string) string {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return target
	}
	return filepath.ToSlash(rel)
}

func normalizeAnnotationPath(p string) string {
	return filepath.ToSlash(p)
}
