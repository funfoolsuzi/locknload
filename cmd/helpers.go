package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/fsnotify/fsnotify"
)

func addToWatchRecursive(fw *fsnotify.Watcher, p string) error {
	fs, err := ioutil.ReadDir(p)
	if err != nil {
		return fmt.Errorf("Error adding %s to watcher. %v", p, err)
	}

	if err = fw.Add(p); err != nil {
		return fmt.Errorf("Error adding file(%s) to watcher. %v", p, err)
	}
	for _, f := range fs {
		if f.IsDir() {
			if err = addToWatchRecursive(fw, path.Join(p, f.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

func isPathDir(p string) (ret bool, retErr error) {
	f, err := os.Open(p)
	if err != nil {
		retErr = fmt.Errorf("Error opening file while checking directory status. %v", err)
		return
	}

	info, err := f.Stat()
	if err != nil {
		retErr = fmt.Errorf("Error getting file info while checking directory stauts. %v", err)
		return
	}

	return info.IsDir(), nil
}
