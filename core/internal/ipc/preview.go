package ipc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"
)

const (
	previewDefaultBytes = 16 * 1024
	previewMaxBytes     = 256 * 1024
)

type previewResult struct {
	Text      string `json:"text"`
	Binary    bool   `json:"binary"`
	Truncated bool   `json:"truncated"`
	Size      int64  `json:"size"`
	Lines     int    `json:"lines"`
}

func HandlePreview(_ context.Context, w *ConnWriter, req Request) {
	path := ParamString(req.Params, "path")
	if !filepath.IsAbs(path) {
		RespondError(w, req.ID, "path is not an absolute path")
		return
	}
	limit := ParamInt(req.Params, "maxBytes")
	switch {
	case limit <= 0:
		limit = previewDefaultBytes
	case limit > previewMaxBytes:
		limit = previewMaxBytes
	}
	result, err := readPreview(path, limit)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	Respond(w, req.ID, result)
}

func readPreview(path string, limit int) (previewResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return previewResult{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return previewResult{}, err
	}
	if info.IsDir() {
		return previewResult{}, errors.New("path is a directory")
	}

	buf := make([]byte, limit+1)
	n, err := io.ReadFull(file, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return previewResult{}, err
	}
	data := buf[:n]
	result := previewResult{Size: info.Size(), Truncated: n > limit}
	if result.Truncated {
		data = data[:limit]
	}
	if bytes.IndexByte(data, 0) >= 0 {
		result.Binary = true
		return result, nil
	}
	if result.Truncated {
		data = trimPartialRune(data)
	}
	if !utf8.Valid(data) {
		result.Binary = true
		return result, nil
	}
	result.Text = string(data)
	result.Lines = bytes.Count(data, []byte{'\n'})
	if len(data) > 0 && data[len(data)-1] != '\n' {
		result.Lines++
	}
	return result, nil
}

func trimPartialRune(data []byte) []byte {
	for cut := 0; cut < utf8.UTFMax && cut < len(data); cut++ {
		if utf8.Valid(data[len(data)-1-cut:]) {
			return data
		}
		if utf8.RuneStart(data[len(data)-1-cut]) {
			return data[:len(data)-1-cut]
		}
	}
	return data
}
