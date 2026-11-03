package app

import (
	"github.com/YanChenBai/cyrene-terminal/assets"
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"strings"
)

var fileIcons = func() map[string]*ui.SVG {
	result := map[string]*ui.SVG{}
	entries, err := assets.FS.ReadDir("file-icons")
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".svg") {
			continue
		}
		data, err := assets.FS.ReadFile("file-icons/" + entry.Name())
		if err != nil {
			panic(err)
		}
		result[strings.TrimSuffix(entry.Name(), ".svg")] = ui.MustParseSVG(data)
	}
	return result
}()

func fileIconName(name string, directory, open bool) string {
	name = strings.ToLower(name)
	if directory {
		id := "default_folder"
		switch name {
		case "node_modules":
			id = "folder_type_node"
		case "src", "internal", "cmd":
			id = "folder_type_src"
		case "scripts", "script":
			id = "folder_type_script"
		case "test", "tests", "test-results", "__tests__":
			id = "folder_type_test"
		case ".git":
			id = "folder_type_git"
		case ".github":
			id = "folder_type_github"
		case "build", "dist", "out":
			id = "folder_type_dist"
		case "config", "configs", ".config":
			id = "folder_type_config"
		case "docs", "doc":
			id = "folder_type_docs"
		case "assets", "resources", "images":
			id = "folder_type_asset"
		case "public", "static":
			id = "folder_type_public"
		}
		if open {
			id += "_opened"
		}
		return id
	}
	switch name {
	case "go.mod", "go.sum":
		return "file_type_go_package"
	case "license", "license.md", "license.txt", "copying", "copying.lesser":
		return "file_type_license"
	case ".gitignore", ".gitattributes", ".gitmodules":
		return "file_type_git"
	case ".editorconfig":
		return "file_type_editorconfig"
	case "dockerfile", "docker-compose.yml", "docker-compose.yaml":
		return "file_type_docker"
	case "package.json", "package-lock.json", ".npmrc":
		return "file_type_npm"
	case "vite.config.ts", "vite.config.js":
		return "file_type_vite"
	}
	if strings.HasSuffix(name, ".d.ts") {
		return "file_type_typescriptdef"
	}
	switch filepath.Ext(name) {
	case ".go":
		return "file_type_go_aqua"
	case ".ps1", ".psm1", ".psd1":
		return "file_type_powershell"
	case ".py", ".pyw", ".pyi":
		return "file_type_python"
	case ".js", ".mjs", ".cjs", ".jsx":
		return "file_type_js_official"
	case ".ts", ".mts", ".cts", ".tsx":
		return "file_type_typescript"
	case ".vue":
		return "file_type_vue"
	case ".json", ".jsonc", ".json5":
		return "file_type_json"
	case ".md", ".mdx":
		return "file_type_markdown"
	case ".yaml", ".yml":
		return "file_type_yaml"
	case ".toml":
		return "file_type_toml"
	case ".xml", ".svg":
		return "file_type_xml"
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".ico", ".bmp", ".avif":
		return "file_type_image"
	case ".rs":
		return "file_type_rust"
	case ".html", ".htm":
		return "file_type_html"
	case ".css", ".scss", ".less":
		return "file_type_css"
	case ".sh", ".bash", ".zsh", ".fish", ".bat", ".cmd":
		return "file_type_shell"
	case ".cs":
		return "file_type_csharp"
	case ".c", ".h":
		return "file_type_c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "file_type_cpp"
	case ".java":
		return "file_type_java"
	case ".sql":
		return "file_type_sql"
	case ".pdf":
		return "file_type_pdf"
	case ".zip", ".gz", ".tar", ".xz", ".7z", ".rar":
		return "file_type_zip"
	case ".ttf", ".otf", ".woff", ".woff2":
		return "file_type_font"
	case ".exe", ".dll", ".bin", ".so", ".dylib":
		return "file_type_binary"
	case ".txt", ".log":
		return "file_type_text"
	case ".env", ".ini", ".cfg", ".conf":
		return "file_type_config"
	}
	return "default_file"
}
func fileTypeIcon(c *ui.Context, name string, directory, open bool) {
	icon := fileIcons[fileIconName(name, directory, open)]
	if icon == nil {
		if directory {
			icon = fileIcons["default_folder"]
		} else {
			icon = fileIcons["default_file"]
		}
	}
	ui.Image(c, icon).Size(16, 16).Shrink(0)
}
