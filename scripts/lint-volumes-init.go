// lint-volumes-init.go 检查 yaml 是否为已知"非 root 容器"声明了 volumes_init
//
// 用法：
//   go run scripts/lint-volumes-init.go [--apps-dir=.] [--strict]
//
// 严格模式（CI 必跑）：缺失 volumes_init 直接退出码 1
// 普通模式（本地预检）：仅 warning
//
// 维护清单：当增加新应用且镜像以非 root 用户运行时，加到 nonRootImages map 即可。
// CI 跑这个 lint 防止"忘了配 volumes_init 导致用户安装时 Permission denied"。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// nonRootImages 已知会以非 root 用户运行的官方镜像名（不含 tag）→ 期望 UID:GID
//
// 增加新应用时维护此 map。如果不确定 UID，运行：
//   docker run --rm <image> sh -c "id" | head -1
//
// 注意：返回 root（UID 0）的镜像 ENTRYPOINT 内会 gosu/su-exec 切换，但 docker run
// 默认是 root；用 docker inspect --format='{{.Config.User}}' 也无法准确判断。
// 唯一可靠的判断方式是看官方文档或测试容器实际写文件的 owner。
var nonRootImages = map[string]string{
	"postgres":         "70:70",     // postgres 用户
	"mysql":            "999:999",   // mysql 用户
	"mongo":            "999:999",   // mongodb 用户
	"redis":            "999:1000",  // redis 用户
	"mariadb":          "999:999",   // mysql 用户
	"elasticsearch":    "1000:0",
	"opensearch":       "1000:0",
	"prometheus":       "65534:65534",
	"grafana":          "472:472",
}

type appYAML struct {
	Key         string             `yaml:"key"`
	Versions    yaml.Node          `yaml:"versions"`
	VolumesInit []volumeInitSpec   `yaml:"volumes_init"`
}

type volumeInitSpec struct {
	Path string `yaml:"path"`
	UID  int    `yaml:"uid"`
	GID  int    `yaml:"gid"`
	Mode string `yaml:"mode"`
}

type versionEntry struct {
	Image string `yaml:"image"`
}

func main() {
	appsDir := flag.String("apps-dir", ".", "apps 仓库根目录")
	strict := flag.Bool("strict", false, "严格模式：缺失 volumes_init 退出码 1")
	flag.Parse()

	root, err := filepath.Abs(*appsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "解析 apps-dir 失败:", err)
		os.Exit(2)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取 apps-dir 失败:", err)
		os.Exit(2)
	}

	type problem struct {
		appKey    string
		yamlPath  string
		image     string
		expected  string
	}
	var problems []problem

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == "scripts" {
			continue
		}
		yamlPath := filepath.Join(root, name, "app.yaml")
		if _, err := os.Stat(yamlPath); err != nil {
			continue
		}

		data, err := os.ReadFile(yamlPath)
		if err != nil {
			continue
		}
		var app appYAML
		if err := yaml.Unmarshal(data, &app); err != nil {
			continue
		}

		// 收集 versions[] 中所有 image 名
		images := collectImages(&app.Versions)

		// 检查每个 image：是否在 nonRootImages 名单
		for _, img := range images {
			expected, isNonRoot := nonRootImages[img]
			if !isNonRoot {
				continue
			}
			// 是非 root 镜像 → 必须有 volumes_init
			if len(app.VolumesInit) == 0 {
				problems = append(problems, problem{
					appKey:   app.Key,
					yamlPath: filepath.Join(name, "app.yaml"),
					image:    img,
					expected: expected,
				})
				break // 同一应用只报一次
			}
		}
	}

	if len(problems) == 0 {
		fmt.Println("✓ 所有非 root 镜像应用都声明了 volumes_init")
		return
	}

	sort.Slice(problems, func(i, j int) bool {
		return problems[i].appKey < problems[j].appKey
	})

	severity := "WARNING"
	if *strict {
		severity = "ERROR"
	}
	for _, p := range problems {
		// GitHub Actions annotation 格式
		fmt.Printf("::%s file=%s,line=1::应用 %s 使用非 root 镜像 %s（建议 UID:GID = %s），但缺少 volumes_init 声明；这会导致容器启动时无法写入挂载目录\n",
			strings.ToLower(severity), p.yamlPath, p.appKey, p.image, p.expected)
	}

	fmt.Fprintf(os.Stderr, "\n%d 个应用缺失 volumes_init\n", len(problems))
	if *strict {
		os.Exit(1)
	}
}

// collectImages 从 versions[] 中抽取所有声明的 image 名（去重）
func collectImages(versions *yaml.Node) []string {
	if versions == nil || versions.Kind != yaml.SequenceNode {
		return nil
	}
	seen := map[string]bool{}
	for _, node := range versions.Content {
		if node.Kind != yaml.MappingNode {
			continue
		}
		var v versionEntry
		if err := node.Decode(&v); err != nil {
			continue
		}
		if v.Image != "" && !seen[v.Image] {
			seen[v.Image] = true
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
