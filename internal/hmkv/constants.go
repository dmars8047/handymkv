package hmkv

import (
	"errors"
	"fmt"
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

type DiscError struct {
	DiscId int
	Msg    string
}

func (e *DiscError) Error() string {
	return fmt.Sprintf("disc %d: %s", e.DiscId, e.Msg)
}

func NewDiscError(discId int, msg string) error {
	return &DiscError{
		DiscId: discId,
		Msg:    msg,
	}
}
