package cmd

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"sync"

	"github.com/fsnotify/fsnotify"

	"github.com/spf13/cobra"
)

var (
	targetDirPath       string
	targetBuildFilePath string
	buildOutputPath     string
)

var rootCmd = &cobra.Command{
	Use:   "locknload",
	Short: "watch go file changes and rebuild go program",
	Run: func(cmd *cobra.Command, args []string) {

		log.Printf("locknload target: %s, output: %s", path.Join(targetDirPath, targetBuildFilePath), buildOutputPath)
		initialBuild := exec.Command("go", "build", "-o", buildOutputPath, path.Join(targetDirPath, targetBuildFilePath))
		log.Printf("Running: %v", initialBuild.Args)
		if output, err := initialBuild.CombinedOutput(); err != nil {
			log.Panicf("Error running initial build. %v. %s", err, output)
		}

		runExecCmd = exec.Command(buildOutputPath)
		out, err := runExecCmd.StdoutPipe()
		if err != nil {
			log.Panicf("Error getting stdout for initial run. %v", err)
		}
		go io.Copy(os.Stdout, out)
		if err := runExecCmd.Start(); err != nil {
			log.Panicf("Error starting initially. %v", err)
		}

		fw, err := fsnotify.NewWatcher()
		if err != nil {
			log.Panicf("Error creating file watcher. %v", err)
		}
		defer fw.Close()

		// add files recursively to the file watcher
		if err = addToWatchRecursive(fw, targetDirPath); err != nil {
			log.Panicf("Error adding files recursively to the watch. %v", err)
		}

		wg := &sync.WaitGroup{}

		wg.Add(2)
		go observeFileWatcherEvents(fw, exitEventsLoop, wg)
		go observeFileWatcherErrors(fw.Errors, exitErrorsLoop, wg)

		wg.Wait()
	},
}

func init() {
	const (
		flagNameTargetDir   = "target"
		flagNameBuildTarget = "build"
		flagNameBuildOutput = "output"
	)

	rootCmd.PersistentFlags().StringVarP(
		&targetDirPath,
		flagNameTargetDir,
		"t",
		"./app",
		"target directory being observed.",
	)
	rootCmd.MarkPersistentFlagRequired(flagNameTargetDir)
	rootCmd.MarkPersistentFlagFilename(flagNameTargetDir)

	rootCmd.PersistentFlags().StringVarP(
		&targetBuildFilePath,
		flagNameBuildTarget,
		"b",
		"*.go",
		"target build entry that will be rebuilt every time there is file update",
	)
	rootCmd.MarkPersistentFlagRequired(flagNameBuildTarget)

	rootCmd.PersistentFlags().StringVarP(
		&buildOutputPath,
		flagNameBuildOutput,
		"o",
		"/tmp/locknload/app",
		"where to store the temporary built app",
	)
}

// Execute will execute the main app
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Panicln(err)
	}
}
