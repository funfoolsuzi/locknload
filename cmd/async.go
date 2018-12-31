package cmd

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"sync"

	"github.com/fsnotify/fsnotify"
)

var (
	exitEventsLoop = make(chan struct{})
	exitErrorsLoop = make(chan struct{})
	buildExecCmd   *exec.Cmd
	runExecCmd     *exec.Cmd
)

func observeFileWatcherEvents(fw *fsnotify.Watcher, exit <-chan struct{}, wg *sync.WaitGroup) {
	if fw == nil {
		log.Println("input fileWatcher shouldn't be nil")
		os.Exit(1)
	}
	log.Println("Starting fsnotify Events Loop")
EventsLoop:
	for {
		select {
		case evt, ok := <-fw.Events:
			if !ok {
				log.Printf("Failed receiving event from filewatcher Events channel")
				continue
			}

			// filter out Chmod event
			if evt.Op == fsnotify.Chmod {
				continue
			}

			if evt.Op == fsnotify.Create { // something has been created
				if isDir, err := isPathDir(evt.Name); err != nil { // Failed to detect whether newly created file/dir is dir. crash it
					log.Printf("Error determining newlly added file/dir(%s) is directory. %v", evt.Name, err)
					os.Exit(1)
				} else if isDir { // new file/dir is a DIR
					if err = fw.Add(evt.Name); err != nil {
						log.Printf("Error adding newly created directory under watch. %v", err)
						os.Exit(1)
					}
					continue
				}
			}

			if evt.Op&fsnotify.Remove != fsnotify.Remove && evt.Name[len(evt.Name)-3:] != ".go" {
				// if this change is not a delete or doesn't have ".go", skip!
				continue
			}

			// if it is reached here, it is a go file change
			buildWaitGroup := &sync.WaitGroup{}
			rebuild(buildWaitGroup)
			buildWaitGroup.Wait()
			restart()

		case <-exit:
			break EventsLoop
		}
	}

	if wg != nil {
		wg.Done()
	}
}

func observeFileWatcherErrors(errs <-chan error, exit <-chan struct{}, wg *sync.WaitGroup) {
	log.Println("Starting fsnotify Errors Loop")
ErrorsLoop:
	for {
		select {
		case err, ok := <-errs:
			if !ok {
				log.Printf("Failed receiving error from filewatcher Errors channel")
			}
			log.Printf("Error received from filewatcher Errors channel. %v", err)
			// TODO: speciall error handling
		case <-exit:
			break ErrorsLoop
		}
	}

	if wg != nil {
		wg.Done()
	}
}

func rebuild(wg *sync.WaitGroup) {
	wg.Add(1)
	if buildExecCmd != nil && buildExecCmd.Process != nil && buildExecCmd.ProcessState == nil { // there is process currently running, killable
		if errKill := buildExecCmd.Process.Kill(); errKill != nil {
			log.Panicf("Error killing running build while awaiting to rebuild. %v", errKill)
		}
		log.Println("Killed previous built")
	}
	buildExecCmd = exec.Command("go", "build", "-o", buildOutputPath, path.Join(targetDirPath, targetBuildFilePath))

	// copy build stdout to current stdout
	output, err := buildExecCmd.StdoutPipe()
	if err != nil {
		log.Panicf("Error getting stdout from target app rebuild. %v", err)
	}
	go io.Copy(os.Stdout, output)

	log.Printf("Running: %v", buildExecCmd.Args)
	if err = buildExecCmd.Start(); err != nil {
		log.Panicf("Error starting target app rebuild. %v", err)
	}
	go func() {
		buildExecCmd.Wait()
		log.Println("Built successfully.")
		wg.Done()
	}()
}

func restart() {
	if runExecCmd != nil && runExecCmd.Process != nil && runExecCmd.ProcessState == nil {
		if errKill := runExecCmd.Process.Kill(); errKill != nil {
			log.Panicf("Error killing running app for restarting. %v", errKill)
		}
		runExecCmd.Wait()
	}
	log.Println("Killed running app")
	runExecCmd = exec.Command(buildOutputPath)

	// copy app stdout to current stdout
	output, err := runExecCmd.StdoutPipe()
	if err != nil {
		log.Panicf("Error getting stdout from target build. %v", err)
	}
	go io.Copy(os.Stdout, output)

	stdE, err := runExecCmd.StderrPipe()
	if err != nil {
		log.Panicf("Error getting stderr for initla run. %v", err)
	}
	go io.Copy(os.Stderr, stdE)

	log.Printf("Running: %v", runExecCmd.Args)
	if err = runExecCmd.Start(); err != nil {
		log.Panicf("Error restarting app. %v", err)
	}
	log.Printf("App restarted. current number of goroutines: %d", runtime.NumGoroutine())
}
