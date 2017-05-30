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
	"gopkg.in/fsnotify.v1"
)

var delay = flag.Int("d", 500, "execution delay in milliseconds")
var sleep = flag.Int("w", 1000, "wait after execution in milliseconds")
var execDir = flag.Bool("c", false, "execute in changed directory")

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
	var lastFileName string
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			if time.Now().Before(lastEvent.Add(time.Duration(*sleep) * time.Millisecond)) {
				continue
			}
			lastFileName = event.Name
			println(time.Now(), event.Name, "changed")
			startTimer.Reset(time.Duration(*delay) * time.Millisecond)
		case err := <-watcher.Errors:
			log.Println("error:", err)
		case <-startTimer.C:
			killProcess(cmd)
			wd, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			cwd := filepath.Join(wd, lastFileName)
			if lastFileName != "" {
				cwd = filepath.Dir(cwd)
			}
			cmd = startCommand(cwd, flag.Args()...)
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

func startCommand(cwd string, args ...string) *exec.Cmd {
	cmd := exec.Command("setsid", args...)
	if *execDir {
		cmd.Dir = cwd
	}
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
