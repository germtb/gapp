package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/germtb/goli"
	"github.com/germtb/gox"

	"github.com/germtb/gap/cli/scaffold"
)

type InitResultProps struct {
	Name      string
	Framework scaffold.Framework
	Files     []string
}

func InitResult(props InitResultProps) gox.VNode {
	return gox.Element("box", gox.Props{"direction": "column"},
		gox.Element("box", gox.Props{"direction": "row"},
			gox.Element("text", gox.Props{"color": "green"},
				gox.V("✓")),
			gox.Element("text", nil,
				gox.V(" Created "+props.Name+"/ ("+string(props.Framework)+")"))),
		gox.V(gox.Map(props.Files, func(f string) gox.VNode {
			return gox.Element("box", gox.Props{"direction": "row"},
				gox.Element("text", gox.Props{"dim": true},
					gox.V("    "+f)))
		})),
		gox.Element("text", nil,
			gox.V("")),
		gox.Element("text", gox.Props{"bold": true},
			gox.V("  Next steps:")),
		gox.Element("box", gox.Props{"direction": "row"},
			gox.Element("text", gox.Props{"dim": true},
				gox.V("    cd ")),
			gox.Element("text", gox.Props{"color": "cyan"},
				gox.V(props.Name))),
		gox.Element("box", gox.Props{"direction": "row"},
			gox.Element("text", gox.Props{"dim": true},
				gox.V("    gap run"))))
}

type InitHintProps struct {
	Name string
}

func InitHint(props InitHintProps) gox.VNode {
	name := props.Name
	if name == "" {
		name = "<name>"
	}
	return gox.Element("box", gox.Props{"direction": "column"},
		gox.Element("box", gox.Props{"direction": "row"},
			gox.Element("text", gox.Props{"color": "red"},
				gox.V("✗")),
			gox.Element("text", nil,
				gox.V(" Missing --framework flag"))),
		gox.Element("text", nil,
			gox.V("")),
		gox.Element("text", gox.Props{"dim": true},
			gox.V("  gap init "+name+" --framework react    # React + TypeScript")),
		gox.Element("text", gox.Props{"dim": true},
			gox.V("  gap init "+name+" --framework vanilla  # Plain TypeScript")),
		gox.Element("text", gox.Props{"dim": true},
			gox.V("  gap init "+name+" -y                   # Default (react)")))
}

type InitErrorProps struct {
	Err error
}

func InitError(props InitErrorProps) gox.VNode {
	return gox.Element("box", gox.Props{"direction": "row"},
		gox.Element("text", gox.Props{"color": "red"},
			gox.V("✗")),
		gox.Element("text", nil,
			gox.V(" "+props.Err.Error())))
}

func RunInit(args []string) error {
	var name, module, framework string
	var skipConfirm bool

	// Parse args manually so flags can appear before or after the name
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--module":
			i++
			if i < len(args) {
				module = args[i]
			}
		case "--framework":
			i++
			if i < len(args) {
				framework = args[i]
			}
		case "-y":
			skipConfirm = true
		default:
			if strings.HasPrefix(args[i], "-") {
				goli.Print(InitError(InitErrorProps{Err: fmt.Errorf("unknown flag: %s", args[i])}))
				return fmt.Errorf("unknown flag: %s", args[i])
			}
			if name == "" {
				name = args[i]
			}
		}
	}

	if name == "" {
		goli.Print(InitError(InitErrorProps{Err: fmt.Errorf("usage: gap init <name> --framework react|vanilla")}))
		return fmt.Errorf("missing project name")
	}

	if module == "" {
		module = name
	}

	dir := filepath.Join(".", name)
	if _, err := os.Stat(dir); err == nil {
		goli.Print(InitError(InitErrorProps{Err: fmt.Errorf("directory %s already exists", name)}))
		return fmt.Errorf("directory %s already exists", name)
	}

	// Determine framework
	var fw scaffold.Framework
	switch framework {
	case "react":
		fw = scaffold.FrameworkReact
	case "vanilla":
		fw = scaffold.FrameworkVanilla
	case "":
		if skipConfirm {
			fw = scaffold.FrameworkReact
		} else {
			goli.Print(InitHint(InitHintProps{Name: name}))
			return fmt.Errorf("missing --framework flag")
		}
	default:
		goli.Print(InitError(InitErrorProps{Err: fmt.Errorf("unknown framework %q (use react or vanilla)", framework)}))
		return fmt.Errorf("unknown framework %q", framework)
	}

	// Resolve gap package paths from the gap binary location
	gapClientPath, gapReactPath, gapServerPath := resolveGapPackages()

	config := scaffold.ProjectConfig{
		Name:          name,
		Module:        module,
		Framework:     fw,
		GapClientPath: gapClientPath,
		GapReactPath:  gapReactPath,
		GapServerPath: gapServerPath,
	}

	files, err := scaffold.Generate(config, dir)
	if err != nil {
		goli.Print(InitError(InitErrorProps{Err: err}))
		return err
	}

	// Run npm install in client/
	goli.Print(gox.Element("box", gox.Props{"direction": "row"},
		gox.Element("text", gox.Props{"dim": true},
			gox.V("  Installing client dependencies..."))))
	npmCmd := exec.Command("npm", "install")
	npmCmd.Dir = filepath.Join(dir, "client")
	npmCmd.Stdout = nil
	npmCmd.Stderr = os.Stderr
	if err := npmCmd.Run(); err != nil {
		goli.Print(gox.Element("box", gox.Props{"direction": "row"},
			gox.Element("text", gox.Props{"color": "yellow"},
				gox.V("!")),
			gox.Element("text", nil,
				gox.V(" npm install failed: "+err.Error()))))
	}

	// Run codegen
	goli.Print(gox.Element("box", gox.Props{"direction": "row"},
		gox.Element("text", gox.Props{"dim": true},
			gox.V("  Running codegen..."))))
	if err := RunCodegen([]string{"--proto", filepath.Join(dir, "proto", "service.proto"), "--go-out", filepath.Join(dir, "server", "generated"), "--ts-out", filepath.Join(dir, "client", "src", "generated"), "--routes-dir", filepath.Join(dir, "client", "src", "routes"), "--preload-out", filepath.Join(dir, "server", "generated", "preload_routes.go")}); err != nil {
		goli.Print(gox.Element("box", gox.Props{"direction": "row"},
			gox.Element("text", gox.Props{"color": "yellow"},
				gox.V("!")),
			gox.Element("text", nil,
				gox.V(" codegen failed: "+err.Error()))))
	}

	// Run go mod tidy for server (after codegen so generated packages exist)
	goli.Print(gox.Element("box", gox.Props{"direction": "row"},
		gox.Element("text", gox.Props{"dim": true},
			gox.V("  Resolving server dependencies..."))))
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = filepath.Join(dir, "server")
	tidyCmd.Stdout = nil
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		goli.Print(gox.Element("box", gox.Props{"direction": "row"},
			gox.Element("text", gox.Props{"color": "yellow"},
				gox.V("!")),
			gox.Element("text", nil,
				gox.V(" go mod tidy failed: "+err.Error()))))
	}

	goli.Print(InitResult(InitResultProps{Name: name, Framework: fw, Files: files}))
	return nil
}

// resolveGapPackages finds the @gap/client and @gap/react packages
// relative to the gap binary location (gap/cli/ -> gap/client/, gap/react/)
func resolveGapPackages() (clientPath, reactPath, serverPath string) {
	exe, err := os.Executable()
	if err != nil {
		return "", "", ""
	}
	// Resolve symlinks
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", "", ""
	}
	// Binary is at gap/cli/gap-cli, so gap repo root is gap/cli/..
	cliDir := filepath.Dir(exe)
	gapRoot := filepath.Dir(cliDir)

	clientDir := filepath.Join(gapRoot, "client")
	reactDir := filepath.Join(gapRoot, "react")
	serverDir := filepath.Join(gapRoot, "server")

	// Verify this is a dev checkout (not just an installed binary) by checking
	// for expected files in each package directory.
	if _, err := os.Stat(filepath.Join(clientDir, "package.json")); err == nil {
		clientPath = clientDir
	}
	if _, err := os.Stat(filepath.Join(reactDir, "package.json")); err == nil {
		reactPath = reactDir
	}
	if _, err := os.Stat(filepath.Join(serverDir, "go.mod")); err == nil {
		serverPath = serverDir
	}
	return
}
