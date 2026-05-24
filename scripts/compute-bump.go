// compute-bump.go 根据 git diff（HEAD~1..HEAD）计算 version.json 的下一个 patch 版本号，
// 并生成中文 changelog。在 GitHub Actions 环境下把结果写入 GITHUB_OUTPUT，
// 本地直接打到 stdout。
//
// 用法：
//
//	go run scripts/compute-bump.go [flags]
//
// Flags：
//
//	--repo <path>   仓库根目录，默认 ".."（脚本同级 scripts/ 上一层）
//	--from <ref>    起始 ref，默认 HEAD~1
//	--to <ref>      结束 ref，默认 HEAD
//
// 输出（写入 GITHUB_OUTPUT 的格式与 stdout 保持一致）：
//
//	version=1.0.2
//	changelog=本次更新涉及 N 个应用：[app1, app2, ...]。详见 git log。
//
// 设计参见 .kiro/specs/app-image-version-reliability/design.md §3.4。
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// versionFile 对应 apps 仓库根目录的 version.json。
type versionFile struct {
	Version   string `json:"version"`
	UpdatedAt string `json:"updated_at"`
	Changelog string `json:"changelog"`
}

func main() {
	repoDir := flag.String("repo", "..", "仓库根目录路径（相对脚本目录）")
	fromRef := flag.String("from", "HEAD~1", "git diff 起始 ref")
	toRef := flag.String("to", "HEAD", "git diff 结束 ref")
	flag.Parse()

	rootAbs, err := filepath.Abs(*repoDir)
	if err != nil {
		fail("解析仓库根目录失败: %v", err)
	}

	// 读取 version.json，计算下一个 patch 版本。
	currentVer, err := readVersion(filepath.Join(rootAbs, "version.json"))
	if err != nil {
		fail("读取 version.json 失败: %v", err)
	}
	nextVer, err := bumpPatch(currentVer)
	if err != nil {
		fail("计算下一个版本号失败: %v", err)
	}

	// 通过 git diff 收集变更涉及的应用 key。
	changedApps, err := collectChangedApps(rootAbs, *fromRef, *toRef)
	if err != nil {
		fail("git diff 失败: %v", err)
	}

	changelog := buildChangelog(changedApps)

	// 输出到 GITHUB_OUTPUT 或 stdout。
	if path := os.Getenv("GITHUB_OUTPUT"); path != "" {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			fail("打开 GITHUB_OUTPUT 失败: %v", err)
		}
		defer f.Close()
		fmt.Fprintf(f, "version=%s\n", nextVer)
		// changelog 可能含换行；GitHub Actions 多行输出需要 heredoc 形式。
		fmt.Fprintf(f, "changelog<<EOF\n%s\nEOF\n", changelog)
	}

	// 同时打到 stdout，便于本地调试。
	fmt.Printf("version=%s\n", nextVer)
	fmt.Printf("changelog=%s\n", changelog)
}

// readVersion 加载 version.json 并返回 .version 字段。
// 兼容 UTF-8 BOM。
func readVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	// 跳过可能存在的 UTF-8 BOM。
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	var v versionFile
	if err := json.Unmarshal(data, &v); err != nil {
		return "", err
	}
	if v.Version == "" {
		return "", fmt.Errorf("version 字段为空")
	}
	return v.Version, nil
}

// bumpPatch 把 SemVer "x.y.z" 的 z 加 1。非 SemVer 直接报错。
func bumpPatch(v string) (string, error) {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("非法 SemVer: %s（要求 x.y.z）", v)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("patch 段非数字: %s", parts[2])
	}
	parts[2] = strconv.Itoa(patch + 1)
	return strings.Join(parts, "."), nil
}

// collectChangedApps 通过 git diff 找出受影响的应用 key（首层目录名）。
// 仅识别 app.yaml 和 docker-compose.yml 的改动；scripts/、.github/ 等改动不算。
func collectChangedApps(rootAbs, from, to string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", from, to)
	cmd.Dir = rootAbs
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	set := map[string]struct{}{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 必须是 <appKey>/app.yaml 或 <appKey>/docker-compose.yml。
		parts := strings.SplitN(line, "/", 2)
		if len(parts) != 2 {
			continue
		}
		key, file := parts[0], parts[1]
		// 跳过非应用目录。
		if key == "" || strings.HasPrefix(key, ".") || key == "scripts" {
			continue
		}
		if file != "app.yaml" && file != "docker-compose.yml" {
			continue
		}
		set[key] = struct{}{}
	}

	apps := make([]string, 0, len(set))
	for k := range set {
		apps = append(apps, k)
	}
	sort.Strings(apps)
	return apps, nil
}

// buildChangelog 产生中文一句话 changelog。
func buildChangelog(apps []string) string {
	if len(apps) == 0 {
		return "维护性更新（不涉及应用 yaml/compose 模板变更）。详见 git log。"
	}
	return fmt.Sprintf("本次更新涉及 %d 个应用：%s。详见 git log。", len(apps), strings.Join(apps, ", "))
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
