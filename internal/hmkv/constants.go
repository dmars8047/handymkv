package hmkv

import (
	"errors"
)

var defaultPossibleEncoderValues []string = []string{
	"svt_av1",
	"svt_av1_10bit",
	"x264",
	"x264_10bit",
	"nvenc_h264",
	"x265",
	"x265_10bit",
	"x265_12bit",
	"nvenc_h265",
	"nvenc_h265_10bit",
	"mpeg4",
	"mpeg2",
	"VP8",
	"VP9",
	"VP9_10bit",
	"theora",
}

var ErrInvalidInput = errors.New("invalid input")
var ErrTitlesDiscRead = errors.New("cannot read titles from disc")

type ExternalProcessError struct {
	Err          error
	ProcessOuput string
}

func NewExternalProcessError(err error, output string) *ExternalProcessError {
	return &ExternalProcessError{
		Err:          err,
		ProcessOuput: output,
	}
}

func (e *ExternalProcessError) Error() string {
	return e.Err.Error()
}
