package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/germtb/goli"
	"github.com/germtb/gox"

	"github.com/germtb/gap/cli/internal/codegen"
)

type CodegenStepProps struct {
	Label   string
	Success bool
	Err     string
}

func CodegenStep(props CodegenStepProps) gox.VNode {
	if props.Success {
		return gox.Element("box", gox.Props{"direction": "row"},
			gox.Element("text", gox.Props{"color": "green"},
				gox.V("✓")),
			gox.Element("text", nil,
				gox.V(" "+props.Label)))
	}
	return gox.Element("box", gox.Props{"direction": "column"},
		gox.Element("box", gox.Props{"direction": "row"},
			gox.Element("text", gox.Props{"color": "red"},
				gox.V("✗")),
			gox.Element("text", nil,
				gox.V(" "+props.Label))),
		gox.Element("text", gox.Props{"dim": true},
			gox.V("    "+props.Err)))
}

func RunCodegen(args []string) error {
	fs := flag.NewFlagSet("codegen", flag.ExitOnError)
	protoFlag := fs.String("proto", "proto/service.proto", "Proto file path")
	goOutFlag := fs.String("go-out", "server/generated", "Go output directory")
	tsOutFlag := fs.String("ts-out", "client/src/generated", "TypeScript output directory")
	routesDirFlag := fs.String("routes-dir", "client/src/routes", "Routes directory for preload config")
	preloadOutFlag := fs.String("preload-out", "server/generated/preload_routes.go", "Preload config output path")
	forceFlag := fs.Bool("force", false, "Force codegen even if proto hasn't changed")

	if err := fs.Parse(args); err != nil {
		return err
	}

	protoFile := *protoFlag
	goOut := *goOutFlag
	tsOut := *tsOutFlag
	routesDir := *routesDirFlag
	preloadOut := *preloadOutFlag

	// Verify proto file exists
	if _, err := os.Stat(protoFile); os.IsNotExist(err) {
		goli.Print(CodegenStep(CodegenStepProps{Label: "Proto file: " + protoFile, Success: false, Err: "file not found"}))
		return fmt.Errorf("proto file not found: %s", protoFile)
	}

	protoDir := filepath.Dir(protoFile)

	// Derive project root (parent of proto/)
	projectDir := filepath.Dir(protoDir)
	if filepath.Base(protoDir) != "proto" {
		projectDir = "."
	}

	// Hash-based caching
	if !*forceFlag {
		currentHash, err := codegen.HashFile(protoFile)
		if err == nil {
			storedHash := codegen.ReadStoredHash(projectDir)
			if currentHash == storedHash {
				goli.Print(gox.Element("box", gox.Props{"direction": "row"},
					gox.Element("text", gox.Props{"color": "green"},
						gox.V("✓")),
					gox.Element("text", nil,
						gox.V(" Proto unchanged, codegen up to date (use --force to re-run)"))))
				return nil
			}
		}
	}
	protoName := filepath.Base(protoFile)

	// Ensure output directories exist
	os.MkdirAll(goOut, 0755)
	os.MkdirAll(tsOut, 0755)

	// Step 1: Compile proto with protocompile (no protoc binary needed)
	req, err := codegen.CompileProto(protoDir, protoName)
	if err != nil {
		goli.Print(CodegenStep(CodegenStepProps{Label: "Proto compilation", Success: false, Err: err.Error()}))
		return fmt.Errorf("proto compilation failed: %w", err)
	}
	goli.Print(CodegenStep(CodegenStepProps{Label: "Proto compilation", Success: true, Err: ""}))

	// Step 2: Generate Go code via protoc-gen-go
	goResp, err := codegen.RunGoPlugin(req, "paths=source_relative")
	if err != nil {
		goli.Print(CodegenStep(CodegenStepProps{Label: "Go codegen", Success: false, Err: err.Error()}))
		return fmt.Errorf("Go codegen failed: %w", err)
	}
	if _, err := codegen.WriteResponse(goResp, goOut); err != nil {
		goli.Print(CodegenStep(CodegenStepProps{Label: "Go codegen", Success: false, Err: err.Error()}))
		return fmt.Errorf("writing Go output: %w", err)
	}
	goli.Print(CodegenStep(CodegenStepProps{Label: "Go codegen → " + goOut, Success: true, Err: ""}))

	// Step 3: Generate TypeScript code via protoc-gen-ts_proto
	tsPlugin, err := findTsProtoPlugin(filepath.Dir(tsOut))
	if err != nil {
		goli.Print(CodegenStep(CodegenStepProps{Label: "TypeScript codegen", Success: false, Err: err.Error()}))
		return err
	}
	tsResp, err := codegen.RunPlugin(req, tsPlugin, "outputServices=default,esModuleInterop=true,useOptionals=messages")
	if err != nil {
		goli.Print(CodegenStep(CodegenStepProps{Label: "TypeScript codegen", Success: false, Err: err.Error()}))
		return fmt.Errorf("TypeScript codegen failed: %w", err)
	}
	if _, err := codegen.WriteResponse(tsResp, tsOut); err != nil {
		goli.Print(CodegenStep(CodegenStepProps{Label: "TypeScript codegen", Success: false, Err: err.Error()}))
		return fmt.Errorf("writing TypeScript output: %w", err)
	}
	goli.Print(CodegenStep(CodegenStepProps{Label: "TypeScript codegen → " + tsOut, Success: true, Err: ""}))

	// Step 4: Generate preload routes config
	if routesDir != "" && preloadOut != "" {
		if _, err := os.Stat(routesDir); err == nil {
			routes, err := codegen.ScanRoutes(routesDir)
			if err != nil {
				goli.Print(CodegenStep(CodegenStepProps{Label: "Preload config", Success: false, Err: err.Error()}))
				return fmt.Errorf("preload config generation failed: %w", err)
			}

			if len(routes) == 0 {
				goli.Print(CodegenStep(CodegenStepProps{Label: "Preload config — no routes with RPCs found", Success: true, Err: ""}))
			} else {
				pkgName := filepath.Base(filepath.Dir(preloadOut))
				goCode := codegen.GeneratePreloadGo(routes, pkgName)

				os.MkdirAll(filepath.Dir(preloadOut), 0755)
				if err := os.WriteFile(preloadOut, []byte(goCode), 0644); err != nil {
					goli.Print(CodegenStep(CodegenStepProps{Label: "Preload config", Success: false, Err: err.Error()}))
					return fmt.Errorf("writing preload config: %w", err)
				}
				goli.Print(CodegenStep(CodegenStepProps{Label: "Preload config → " + preloadOut, Success: true, Err: ""}))
			}
		}
	}

	// Write hash after successful codegen
	if hash, err := codegen.HashFile(protoFile); err == nil {
		codegen.WriteHash(projectDir, hash)
	}

	return nil
}

func findTsProtoPlugin(tsOutDir string) (string, error) {
	// Walk up from ts output dir to find client/node_modules
	dir := tsOutDir
	for dir != "/" && dir != "." {
		candidate := filepath.Join(dir, "node_modules", ".bin", "protoc-gen-ts_proto")
		if _, err := os.Stat(candidate); err == nil {
			abs, _ := filepath.Abs(candidate)
			return abs, nil
		}
		dir = filepath.Dir(dir)
	}
	// Also check relative to CWD
	local := filepath.Join("client", "node_modules", ".bin", "protoc-gen-ts_proto")
	if _, err := os.Stat(local); err == nil {
		abs, _ := filepath.Abs(local)
		return abs, nil
	}
	path, err := exec.LookPath("protoc-gen-ts_proto")
	if err == nil {
		return path, nil
	}
	return "", fmt.Errorf("protoc-gen-ts_proto not found. Run: cd client && npm install")
}
