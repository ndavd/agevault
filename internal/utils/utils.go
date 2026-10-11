package utils

import (
	"os"
)

func Exists(path string) (exists bool, isDir bool) {
	info, err := os.Stat(path)
	exists = true
	if err != nil {
		exists = false
		return
	}
	isDir = info.IsDir()
	return
}

type MatcherFunc func(filename string) bool

func FileMatchInCwd(match MatcherFunc) (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir, err := os.Open(currentDir)
	if err != nil {
		return "", err
	}
	defer dir.Close()
	files, err := dir.ReadDir(-1)
	if err != nil {
		return "", err
	}
	for _, file := range files {
		if !file.IsDir() && match(file.Name()) {
			return file.Name(), nil
		}
	}
	return "", nil
}
