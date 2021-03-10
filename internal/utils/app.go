package utils

import (
	"os"
	"path/filepath"
)

func AppName() string {
	return "goca"
}

var tmpDir = filepath.Join(os.TempDir(), AppName())

func TmpDir() string {
	return tmpDir
}

func InitTmpDir() error {
	_, err := os.Stat(tmpDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(tmpDir, 0764)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}
