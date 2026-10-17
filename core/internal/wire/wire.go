package wire

import (
	"errors"

	"github.com/AvengeMedia/dankgo/files"
	"github.com/AvengeMedia/dankgo/ipc"
	"github.com/AvengeMedia/dankgo/ipc/params"
)

type errorResponse struct {
	ID    int    `json:"id,omitempty"`
	Error string `json:"error"`
	Code  string `json:"code"`
}

func RespondError(w *ipc.ConnWriter, id int, err error) {
	_ = w.WriteResponse(errorResponse{ID: id, Error: message(err), Code: string(files.CodeOf(err))})
}

func message(err error) string {
	var coded *files.Error
	if errors.As(err, &coded) && coded.Err != nil {
		return coded.Err.Error()
	}
	return err.Error()
}

func Invalid(msg string) error {
	return &files.Error{Code: files.CodeInvalid, Err: errors.New(msg)}
}

func StringList(p map[string]any, key string) []string {
	if value := params.StringOpt(p, key, ""); value != "" {
		return []string{value}
	}
	return params.StringSlice(p, key)
}
