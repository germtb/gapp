package scaffold

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:templates
var templateFS embed.FS

type Framework string

const (
	FrameworkReact   Framework = "react"
	FrameworkVanilla Framework = "vanilla"
)

type ProjectConfig struct {
	Name          string
	Module        string
	Framework     Framework
	GapClientPath string // absolute path to @gap/client package
	GapReactPath  string // absolute path to @gap/react package (react only)
	GapServerPath string // absolute path to gap server Go module
}

// templateFile maps a template path to an output path.
type templateFile struct {
	src string // path within embed.FS (relative to templates/<prefix>/)
	dst string // output path relative to project dir
}

var sharedFiles = []templateFile{
	{"proto/service.proto", "proto/service.proto"},
	{"server/go.mod.tmpl", "server/go.mod"},
	{"server/main.go.tmpl", "server/main.go"},
	{"client/src/rpc.ts.tmpl", "client/src/rpc.ts"},
	{"client/src/rpcTypes.ts.tmpl", "client/src/rpcTypes.ts"},
	{"client/src/preload.ts.tmpl", "client/src/preload.ts"},
	{"client/src/stores/ItemStore.ts.tmpl", "client/src/stores/ItemStore.ts"},
	{"gap-codegen.sh.tmpl", "gap-codegen.sh"},
	{"Dockerfile.tmpl", "Dockerfile"},
}

var reactFiles = []templateFile{
	{"client/package.json.tmpl", "client/package.json"},
	{"client/tsconfig.json.tmpl", "client/tsconfig.json"},
	{"client/vite.config.ts.tmpl", "client/vite.config.ts"},
	{"client/index.html.tmpl", "client/index.html"},
	{"client/src/main.tsx.tmpl", "client/src/main.tsx"},
	{"client/src/routes/HomeRoute.tsx.tmpl", "client/src/routes/HomeRoute.tsx"},
}

var vanillaFiles = []templateFile{
	{"client/package.json.tmpl", "client/package.json"},
	{"client/tsconfig.json.tmpl", "client/tsconfig.json"},
	{"client/vite.config.ts.tmpl", "client/vite.config.ts"},
	{"client/index.html.tmpl", "client/index.html"},
	{"client/src/main.ts.tmpl", "client/src/main.ts"},
	{"client/src/routes/HomeRoute.ts.tmpl", "client/src/routes/HomeRoute.ts"},
}

func filesForFramework(fw Framework) []struct {
	prefix string
	files  []templateFile
} {
	fwFiles := reactFiles
	fwPrefix := "react"
	if fw == FrameworkVanilla {
		fwFiles = vanillaFiles
		fwPrefix = "vanilla"
	}
	return []struct {
		prefix string
		files  []templateFile
	}{
		{"shared", sharedFiles},
		{fwPrefix, fwFiles},
	}
}

// Generate creates a new gap project in the given directory.
// Returns the list of created files (relative to dir).
func Generate(config ProjectConfig, dir string) ([]string, error) {
	if config.Framework == "" {
		config.Framework = FrameworkReact
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating project directory: %w", err)
	}

	// Create server/generated directory
	if err := os.MkdirAll(filepath.Join(dir, "server", "generated"), 0755); err != nil {
		return nil, fmt.Errorf("creating server/generated: %w", err)
	}

	// Create client/src/generated directory
	if err := os.MkdirAll(filepath.Join(dir, "client", "src", "generated"), 0755); err != nil {
		return nil, fmt.Errorf("creating client/src/generated: %w", err)
	}

	var created []string

	for _, group := range filesForFramework(config.Framework) {
		for _, f := range group.files {
			outPath := filepath.Join(dir, f.dst)

			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				return nil, fmt.Errorf("creating directory for %s: %w", f.dst, err)
			}

			content, err := templateFS.ReadFile("templates/" + group.prefix + "/" + f.src)
			if err != nil {
				return nil, fmt.Errorf("reading template %s/%s: %w", group.prefix, f.src, err)
			}

			rendered, err := renderTemplate(f.src, string(content), config)
			if err != nil {
				return nil, fmt.Errorf("rendering template %s: %w", f.src, err)
			}

			perm := os.FileMode(0644)
			if strings.HasSuffix(f.dst, ".sh") {
				perm = 0755
			}

			if err := os.WriteFile(outPath, []byte(rendered), perm); err != nil {
				return nil, fmt.Errorf("writing %s: %w", f.dst, err)
			}

			created = append(created, f.dst)
		}
	}

	return created, nil
}

// ProtoPackage returns the project name sanitized for use as a protobuf package name
// (hyphens replaced with underscores).
func (c ProjectConfig) ProtoPackage() string {
	return strings.ReplaceAll(c.Name, "-", "_")
}

func renderTemplate(name, content string, data ProjectConfig) (string, error) {
	tmpl, err := template.New(name).Delims("<<", ">>").Parse(content)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
