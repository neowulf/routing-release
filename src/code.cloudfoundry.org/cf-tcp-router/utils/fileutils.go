package utils

import (
	"os"
)

func CopyFile(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return WriteToFile(data, dest)
}

func WriteToFile(data []byte, fileName string) error {
	var file *os.File
	var err error
	file, err = os.Create(fileName)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	err = file.Close()
	if err != nil {
		return err
	}
	return nil
}

func FileExists(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil || os.IsExist(err)
}

func DirExists(dirName string) bool {
	info, err := os.Stat(dirName)
	return err == nil && info.IsDir()
}
