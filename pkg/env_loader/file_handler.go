package env_loader

import (
	"fmt"
	"os"
	"strings"
)

const (
	BUFF_SIZE = 256
)

type FileHandler struct {
	Filename string
	Fd       *os.File
}

func Open(filename string, handler *FileHandler) error {
	file, err := os.Open(fmt.Sprintf("./%s", filename))
	if err != nil {
		return err
	}

	handler.Filename = filename
	handler.Fd = file

	return nil
}

func (h *FileHandler) Read() (error, map[string]string) {
	data := make([]byte, 0)
	buff := make([]byte, BUFF_SIZE)

	n, err := h.Fd.Read(buff)

	for n == BUFF_SIZE {
		if err != nil {
			return err, nil
		}

		data = append(data, buff...)
		n, err = h.Fd.Read(buff)
	}

	data = append(data, buff[:n]...)

	return nil, h.unmarshalToMap(string(data))
}

func (h *FileHandler) unmarshalToMap(s string) map[string]string {
	result := make(map[string]string)
	for _, row := range strings.Split(s, "\n") {
		splittedRow := strings.Split(row, "=")
		result[splittedRow[0]] = splittedRow[1]
	}
	return result
}

func (h *FileHandler) Close() {
	_ = h.Fd.Close()
}
