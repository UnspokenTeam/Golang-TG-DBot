package env_loader

import (
	"io"
	"os"
	path2 "path/filepath"
)

const BufferSize = 2048

type File struct {
	path string
	data *string
}

func (file *File) Read() error {
	f, fileOpeningError := os.Open(file.path)
	if fileOpeningError != nil {
		return fileOpeningError
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	buffer := make([]byte, BufferSize)
	byteContent := make([]byte, 0)

	for {
		n, readError := f.Read(buffer)
		if readError != nil && readError != io.EOF {
			return readError
		}

		if n == 0 {
			break
		}

		byteContent = append(byteContent, buffer[:n]...)
	}

	stringData := string(byteContent[:])
	file.data = &stringData
	return nil
}

func (file *File) GetData() *string {
	return file.data
}

func CreateFile(path string) (file File) {
	var concatedPath string
	if path2.IsAbs(path) {
		concatedPath = path
	} else {
		currentFolderPath, _ := path2.Abs(".")
		concatedPath = path2.Join(currentFolderPath, path)
	}

	file.path = concatedPath
	return file
}
