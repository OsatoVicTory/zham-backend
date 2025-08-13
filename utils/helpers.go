package utils

import (
	"io"
	"os"
)

func DeleteFile(filePath string) error {
	if _, err := os.Stat(filePath); err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil {
		return err
	}

	return nil
}

func MoveFile(sourcePath string, destinationPath string) error {
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}

	destFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	err = srcFile.Close()
	if err != nil {
		return err
	}

	err = os.Remove(sourcePath)
	if err != nil {
		return err
	}

	return nil
}
