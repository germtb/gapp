package cmd

import (
	"bufio"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/germtb/goli"
	"github.com/germtb/gox"
)

type LogPaneProps struct {
	Title string
	Lines goli.Accessor[[]string]
}

func LogPane(props LogPaneProps) gox.VNode {
	lines := props.Lines()

	return gox.Element("box", gox.Props{"direction": "column", "grow": 1, "border": "rounded", "overflow": "hidden"},
		gox.Element("box", gox.Props{"direction": "row"},
			gox.Element("text", gox.Props{"bold": true},
				gox.V(" "+props.Title+" "))),
		gox.V(gox.Map(lines, func(line string) gox.VNode {
			return gox.Element("ansi", nil,
				gox.V(line))
		})))
}

type RunAppProps struct {
	ServerLines goli.Accessor[[]string]
	ClientLines goli.Accessor[[]string]
}

func RunApp(props RunAppProps) gox.VNode {
	return gox.Element("box", gox.Props{"direction": "column", "grow": 1},
		LogPane(LogPaneProps{Title: "server", Lines: props.ServerLines}),
		LogPane(LogPaneProps{Title: "client", Lines: props.ClientLines}),
		gox.Element("text", gox.Props{"dim": true},
			gox.V(" Ctrl+C to stop")))
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		syscall.Kill(-pgid, syscall.SIGTERM)
	}
}

func RunRun(args []string) error {
	if _, err := os.Stat("server/main.go"); os.IsNotExist(err) {
		goli.Print(gox.Element("box", gox.Props{"direction": "row"},
			gox.Element("text", gox.Props{"color": "red"},
				gox.V("✗")),
			gox.Element("text", nil,
				gox.V(" Not a gapp project (server/main.go not found)"))))
		return err
	}

	serverLines, setServerLines := goli.CreateSignal([]string{})
	clientLines, setClientLines := goli.CreateSignal([]string{})

	var serverCmd *exec.Cmd
	var clientCmd *exec.Cmd
	var mu sync.Mutex

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			mu.Lock()
			defer mu.Unlock()
			killProcessGroup(serverCmd)
			killProcessGroup(clientCmd)
		})
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	go func() {
		<-sigCh
		cleanup()
		os.Exit(0)
	}()

	startSubprocess := func(name string, cmdArgs []string, dir string, setter goli.Setter[[]string], getter goli.Accessor[[]string]) *exec.Cmd {
		cmd := exec.Command(name, cmdArgs...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "FORCE_COLOR=1")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		r, w, err := os.Pipe()
		if err != nil {
			setter([]string{"Failed to create pipe: " + err.Error()})
			return nil
		}
		cmd.Stdout = w
		cmd.Stderr = w

		if err := cmd.Start(); err != nil {
			setter([]string{"Failed to start: " + err.Error()})
			r.Close()
			w.Close()
			return nil
		}
		w.Close()

		setter([]string{"Starting " + name + " ..."})

		go func() {
			scanner := bufio.NewScanner(r)
			scanner.Buffer(make([]byte, 64*1024), 64*1024)
			for scanner.Scan() {
				line := scanner.Text()
				goli.SetWith(setter, func(prev []string) []string {
					next := append(prev, line)
					if len(next) > 500 {
						next = next[len(next)-500:]
					}
					return next
				}, getter)
			}
			r.Close()
		}()

		go func() {
			cmd.Wait()
			time.Sleep(50 * time.Millisecond)
			goli.SetWith(setter, func(prev []string) []string {
				return append(prev, "Process exited")
			}, getter)
		}()

		return cmd
	}

	goli.Run(func() gox.VNode {
		return RunApp(RunAppProps{ServerLines: serverLines, ClientLines: clientLines})
	}, goli.RunOptions{
		OnMount: func(app *goli.App) {
			go func() {
				ticker := time.NewTicker(50 * time.Millisecond)
				defer ticker.Stop()
				for range ticker.C {
					app.Rerender()
				}
			}()

			serverCmd = startSubprocess("go", []string{"run", "."}, "server", setServerLines, serverLines)
			clientCmd = startSubprocess("./node_modules/.bin/vite", nil, "client", setClientLines, clientLines)
		},
		OnUnmount: func() {
			signal.Stop(sigCh)
			cleanup()
		},
	})

	return nil
}
