package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
)

var delay = flag.Int("d", 500, "start delay in milliseconds")
var sleep = flag.Int("s", 1000, "sleep in milliseconds")

func main() {
	flag.Parse()

	watcher, err := newWatcher()
	if err != nil {
		log.Fatal(err)
	}

	println := color.New(color.FgYellow).PrintlnFunc()

	var cmd *exec.Cmd
	startTimer := time.NewTimer(0)
	var lastEvent time.Time
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			if time.Now().Before(lastEvent.Add(time.Duration(*sleep) * time.Millisecond)) {
				continue
			}
			println(time.Now(), event.Name, "changed")
			startTimer.Reset(time.Duration(*delay) * time.Millisecond)
		case err := <-watcher.Errors:
			log.Println("error:", err)
		case <-startTimer.C:
			killProcess(cmd)
			cmd = startCommand(flag.Args()...)
			lastEvent = time.Now()
		}
	}
}

func newWatcher() (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	watcher.Add(".")
	i := 0
	fn := func(path string, f os.FileInfo, err error) error {
		if f != nil && f.IsDir() {
			if !strings.HasPrefix(path, ".") && !strings.Contains(path, "/.") {
				i++
				return watcher.Add(path)
			}
		}
		return nil
	}
	err = filepath.Walk(".", fn)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Watching %v dirs\n", i)
	return watcher, nil
}

func startCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("setsid", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Start()

	if err != nil {
		log.Fatal(err)
	}
	return cmd
}

func killProcess(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	err := cmd.Wait()
	if err != nil {
		log.Println(err)
	}
	cmd = nil
}
