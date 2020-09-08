package main

import (
	"os"
)

func fileExists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// file not found
		return false
	}

	return true
}

func rmfile(fspath string) error {
	if !fileExists(fspath) {
		return nil
	}

	err := os.Remove(fspath)
	if err != nil {
		return err
	}
	return nil
}

func createLockfile(lockPath string) error {
	lockHandle, errLock := os.Create(lockPath)
	if errLock != nil {
		return errLock
	}
	lockHandle.Close()

	return nil
}
