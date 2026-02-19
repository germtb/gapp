package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/germtb/gapp/cli/internal/codegen"
	"github.com/germtb/gapp/cli/scaffold"
)

func TestInitGeneratesReactProject(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "testapp")

	config := scaffold.ProjectConfig{
		Name:      "testapp",
		Module:    "testapp",
		Framework: scaffold.FrameworkReact,
	}

	files, err := scaffold.Generate(config, projectDir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No files generated")
	}

	// Verify all expected files exist
	expectedFiles := []string{
		"proto/service.proto",
		"server/go.mod",
		"server/main.go",
		"client/package.json",
		"client/tsconfig.json",
		"client/vite.config.ts",
		"client/index.html",
		"client/src/main.tsx",
		"client/src/rpc.ts",
		"client/src/rpcTypes.ts",
		"client/src/preload.ts",
		"client/src/stores/ItemStore.ts",
		"client/src/routes/HomeRoute.tsx",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(projectDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file not found: %s", f)
		}
	}

	// Verify directories exist
	expectedDirs := []string{
		"server/generated",
		"client/src/generated",
	}
	for _, d := range expectedDirs {
		path := filepath.Join(projectDir, d)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			t.Errorf("Expected directory not found: %s", d)
		} else if !info.IsDir() {
			t.Errorf("Expected directory but found file: %s", d)
		}
	}

	// Verify proto file contains correct package name
	protoContent, err := os.ReadFile(filepath.Join(projectDir, "proto/service.proto"))
	if err != nil {
		t.Fatalf("Failed to read proto file: %v", err)
	}
	if !strings.Contains(string(protoContent), "package testapp;") {
		t.Error("Proto file does not contain correct package name")
	}

	// Verify go.mod contains correct module name
	goModContent, err := os.ReadFile(filepath.Join(projectDir, "server/go.mod"))
	if err != nil {
		t.Fatalf("Failed to read server/go.mod: %v", err)
	}
	if !strings.Contains(string(goModContent), "module testapp/server") {
		t.Error("go.mod does not contain correct module name")
	}

	// Verify server/main.go has correct import
	mainContent, err := os.ReadFile(filepath.Join(projectDir, "server/main.go"))
	if err != nil {
		t.Fatalf("Failed to read server/main.go: %v", err)
	}
	if !strings.Contains(string(mainContent), `pb "testapp/server/generated"`) {
		t.Error("server/main.go does not contain correct import path")
	}

	// Verify react deps in package.json
	pkgContent, err := os.ReadFile(filepath.Join(projectDir, "client/package.json"))
	if err != nil {
		t.Fatalf("Failed to read client/package.json: %v", err)
	}
	if !strings.Contains(string(pkgContent), `"react"`) {
		t.Error("React package.json should contain react dependency")
	}
	if !strings.Contains(string(pkgContent), `"@gapp/react"`) {
		t.Error("React package.json should contain @gapp/react dependency")
	}
}

func TestInitGeneratesVanillaProject(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "testapp")

	config := scaffold.ProjectConfig{
		Name:      "testapp",
		Module:    "testapp",
		Framework: scaffold.FrameworkVanilla,
	}

	files, err := scaffold.Generate(config, projectDir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No files generated")
	}

	// Verify vanilla-specific files exist
	vanillaFiles := []string{
		"client/src/main.ts",
		"client/src/routes/HomeRoute.ts",
	}
	for _, f := range vanillaFiles {
		path := filepath.Join(projectDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected vanilla file not found: %s", f)
		}
	}

	// Verify react-specific files do NOT exist
	reactFiles := []string{
		"client/src/main.tsx",
		"client/src/routes/HomeRoute.tsx",
	}
	for _, f := range reactFiles {
		path := filepath.Join(projectDir, f)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("React file should not exist in vanilla project: %s", f)
		}
	}

	// Verify no react in package.json
	pkgContent, err := os.ReadFile(filepath.Join(projectDir, "client/package.json"))
	if err != nil {
		t.Fatalf("Failed to read client/package.json: %v", err)
	}
	if strings.Contains(string(pkgContent), `"react"`) {
		t.Error("Vanilla package.json should not contain react dependency")
	}
	if strings.Contains(string(pkgContent), `"@gapp/react"`) {
		t.Error("Vanilla package.json should not contain @gapp/react dependency")
	}

	// Verify no jsx in tsconfig.json
	tsconfigContent, err := os.ReadFile(filepath.Join(projectDir, "client/tsconfig.json"))
	if err != nil {
		t.Fatalf("Failed to read client/tsconfig.json: %v", err)
	}
	if strings.Contains(string(tsconfigContent), `"jsx"`) {
		t.Error("Vanilla tsconfig.json should not contain jsx option")
	}

	// Verify shared files still exist
	sharedFiles := []string{
		"proto/service.proto",
		"server/go.mod",
		"server/main.go",
		"client/src/rpc.ts",
		"client/src/stores/ItemStore.ts",
	}
	for _, f := range sharedFiles {
		path := filepath.Join(projectDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected shared file not found: %s", f)
		}
	}

	// Verify index.html references .ts not .tsx
	indexContent, err := os.ReadFile(filepath.Join(projectDir, "client/index.html"))
	if err != nil {
		t.Fatalf("Failed to read client/index.html: %v", err)
	}
	if !strings.Contains(string(indexContent), `/src/main.ts"`) {
		t.Error("Vanilla index.html should reference /src/main.ts")
	}
	if strings.Contains(string(indexContent), `.tsx`) {
		t.Error("Vanilla index.html should not reference .tsx")
	}
}

func TestCodegenGoFromScaffoldedProject(t *testing.T) {
	// Scaffold a project
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "testapp")

	config := scaffold.ProjectConfig{
		Name:      "testapp",
		Module:    "testapp",
		Framework: scaffold.FrameworkReact,
	}

	_, err := scaffold.Generate(config, projectDir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Compile the proto
	req, err := codegen.CompileProto(
		filepath.Join(projectDir, "proto"),
		"service.proto",
	)
	if err != nil {
		t.Fatalf("CompileProto failed: %v", err)
	}

	if len(req.ProtoFile) == 0 {
		t.Fatal("No proto files in request")
	}
	if len(req.FileToGenerate) != 1 || req.FileToGenerate[0] != "service.proto" {
		t.Errorf("FileToGenerate = %v, want [service.proto]", req.FileToGenerate)
	}

	// Generate Go code (requires protoc-gen-go in PATH or via `go run`)
	_, err = exec.LookPath("protoc-gen-go")
	if err != nil {
		// Also check if `go` is available for `go run` fallback
		_, err = exec.LookPath("go")
		if err != nil {
			t.Skip("Neither protoc-gen-go nor go in PATH, skipping Go codegen test")
		}
	}

	goOut := filepath.Join(projectDir, "server", "generated")
	goResp, err := codegen.RunGoPlugin(req, "paths=source_relative")
	if err != nil {
		t.Fatalf("RunGoPlugin failed: %v", err)
	}

	written, err := codegen.WriteResponse(goResp, goOut)
	if err != nil {
		t.Fatalf("WriteResponse failed: %v", err)
	}

	if len(written) == 0 {
		t.Fatal("No Go files generated")
	}

	// Verify .pb.go file exists and contains expected content
	pbFile := filepath.Join(goOut, "service.pb.go")
	content, err := os.ReadFile(pbFile)
	if err != nil {
		t.Fatalf("Generated file not found: %v", err)
	}
	if !strings.Contains(string(content), "type Item struct") {
		t.Error("Generated Go code should contain Item struct")
	}
	if !strings.Contains(string(content), "type GetItemsRequest struct") {
		t.Error("Generated Go code should contain GetItemsRequest struct")
	}
}

func TestCodegenHashCaching(t *testing.T) {
	dir := t.TempDir()

	// Write a dummy proto file
	protoDir := filepath.Join(dir, "proto")
	os.MkdirAll(protoDir, 0755)
	os.WriteFile(filepath.Join(protoDir, "test.proto"), []byte("syntax = \"proto3\";"), 0644)

	// Hash it
	hash, err := codegen.HashFile(filepath.Join(protoDir, "test.proto"))
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}
	if hash == "" {
		t.Fatal("Hash should not be empty")
	}

	// No stored hash initially
	stored := codegen.ReadStoredHash(dir)
	if stored != "" {
		t.Error("Should have no stored hash initially")
	}

	// Write and read back
	if err := codegen.WriteHash(dir, hash); err != nil {
		t.Fatalf("WriteHash failed: %v", err)
	}

	stored = codegen.ReadStoredHash(dir)
	if stored != hash {
		t.Errorf("Stored hash = %q, want %q", stored, hash)
	}

	// Modify file, hash should change
	os.WriteFile(filepath.Join(protoDir, "test.proto"), []byte("syntax = \"proto3\";\npackage foo;"), 0644)
	newHash, _ := codegen.HashFile(filepath.Join(protoDir, "test.proto"))
	if newHash == hash {
		t.Error("Hash should change when file content changes")
	}
}
