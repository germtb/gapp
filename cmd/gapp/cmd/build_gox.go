package cmd

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/germtb/goli"
	"github.com/germtb/gox"
)

type BuildStepProps struct {
	Label   string
	Success bool
	Err     string
}

func BuildStep(props BuildStepProps) gox.VNode {
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

func RunBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	outputFlag := fs.String("o", "build", "Output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	outputDir := *outputFlag

	// Validate project structure
	if _, err := os.Stat("server/main.go"); os.IsNotExist(err) {
		goli.Print(BuildStep(BuildStepProps{Label: "Validate project", Success: false, Err: "server/main.go not found"}))
		return fmt.Errorf("not a gapp project (server/main.go not found)")
	}
	if _, err := os.Stat("client/package.json"); os.IsNotExist(err) {
		goli.Print(BuildStep(BuildStepProps{Label: "Validate project", Success: false, Err: "client/package.json not found"}))
		return fmt.Errorf("not a gapp project (client/package.json not found)")
	}
	goli.Print(BuildStep(BuildStepProps{Label: "Validate project", Success: true, Err: ""}))

	// Create temp dir
	tmpDir := fmt.Sprintf(".gapp-build-tmp-%d", rand.Int())
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		goli.Print(BuildStep(BuildStepProps{Label: "Create temp directory", Success: false, Err: err.Error()}))
		return err
	}
	cleanup := func() { os.RemoveAll(tmpDir) }

	// Step 1: npm run build in client/
	npmCmd := exec.Command("npm", "run", "build")
	npmCmd.Dir = "client"
	npmCmd.Stderr = os.Stderr
	if out, err := npmCmd.Output(); err != nil {
		cleanup()
		errMsg := string(out)
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			errMsg = string(exitErr.Stderr)
		}
		goli.Print(BuildStep(BuildStepProps{Label: "Build client (npm run build)", Success: false, Err: errMsg}))
		return fmt.Errorf("client build failed: %w", err)
	}
	goli.Print(BuildStep(BuildStepProps{Label: "Build client (npm run build)", Success: true, Err: ""}))

	// Step 2: go build in server/
	serverBin := filepath.Join(tmpDir, "server")
	goCmd := exec.Command("go", "build", "-o", mustAbs(serverBin), ".")
	goCmd.Dir = "server"
	goCmd.Stderr = os.Stderr
	if out, err := goCmd.Output(); err != nil {
		cleanup()
		errMsg := string(out)
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			errMsg = string(exitErr.Stderr)
		}
		goli.Print(BuildStep(BuildStepProps{Label: "Build server (go build)", Success: false, Err: errMsg}))
		return fmt.Errorf("server build failed: %w", err)
	}
	goli.Print(BuildStep(BuildStepProps{Label: "Build server (go build)", Success: true, Err: ""}))

	// Step 3: Copy server/public/ → tmpDir/public/
	srcPublic := filepath.Join("server", "public")
	dstPublic := filepath.Join(tmpDir, "public")
	if err := copyDir(srcPublic, dstPublic); err != nil {
		cleanup()
		goli.Print(BuildStep(BuildStepProps{Label: "Copy public assets", Success: false, Err: err.Error()}))
		return fmt.Errorf("copying public dir: %w", err)
	}
	goli.Print(BuildStep(BuildStepProps{Label: "Copy public assets", Success: true, Err: ""}))

	// Step 4: Atomic swap
	os.RemoveAll(outputDir)
	if err := os.Rename(tmpDir, outputDir); err != nil {
		cleanup()
		goli.Print(BuildStep(BuildStepProps{Label: "Finalize output", Success: false, Err: err.Error()}))
		return fmt.Errorf("rename failed: %w", err)
	}

	runCmd := "    cd " + outputDir + " && ./server"
	goli.Print(gox.Element("box", gox.Props{"direction": "column"},
		gox.Element("box", gox.Props{"direction": "row"},
			gox.Element("text", gox.Props{"color": "green"},
				gox.V("✓")),
			gox.Element("text", gox.Props{"bold": true},
				gox.V(" Build complete → "+outputDir+"/"))),
		gox.Element("text", nil,
			gox.V("")),
		gox.Element("text", gox.Props{"dim": true},
			gox.V("  Run with:")),
		gox.Element("text", gox.Props{"dim": true},
			gox.V(runCmd))))

	return nil
}

func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
