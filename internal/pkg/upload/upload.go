package upload

import (
	"errors"
	"io"
	"mime/multipart"
)

var ErrFileTooLarge = errors.New("file too large")

func ReadFile(form *multipart.Reader, field string, maxSize int64) ([]byte, error) {
	for {
		part, err := form.NextPart()
		if err != nil {
			return nil, err
		}
		if part.FormName() != field {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(part, maxSize+1))
		if err != nil {
			return nil, err
		}
		if int64(len(data)) > maxSize {
			return nil, ErrFileTooLarge
		}
		return data, nil
	}
}
