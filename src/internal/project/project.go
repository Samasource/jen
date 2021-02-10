package project

import (
	"fmt"
	"os"
	"path"

	"github.com/Samasource/jen/src/internal/constant"
	"github.com/Samasource/jen/src/internal/helpers"
)

// GetProjectDir returns the project's root dir. It finds it by looking for the jen.yaml file
// in current working dir and then walking up the directory structure until it reaches the
// volume's root dir. If it doesn't find it, it returns an empty string.
func GetProjectDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("finding project's root dir: %w", err)
	}

	for {
		filePath := path.Join(dir, constant.JenFileName)
		if helpers.PathExists(filePath) {
			return dir, nil
		}
		if dir == "/" {
			return "", nil
		}
		dir = path.Dir(dir)
	}
}