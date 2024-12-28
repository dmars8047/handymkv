package hnd

import (
	"errors"
)

const (
	INPUT_DIR_NAME  = "Input"
	OUTPUT_DIR_NAME = "Output"
	DEFAULT_QUALITY = 20     // 18
	DEFAULT_ENCODER = "h264" //nvenc_h264
)

var PossibleEncoderValues map[string]struct{} = map[string]struct{}{
	"svt_av1":          {},
	"svt_av1_10bit":    {},
	"x264":             {},
	"x264_10bit":       {},
	"nvenc_h264":       {},
	"x265":             {},
	"x265_10bit":       {},
	"x265_12bit":       {},
	"nvenc_h265":       {},
	"nvenc_h265_10bit": {},
	"mpeg4":            {},
	"mpeg2":            {},
	"VP8":              {},
	"VP9":              {},
	"VP9_10bit":        {},
	"theora":           {},
}

var ErrInvalidInput = errors.New("invalid input")
var ErrTitlesDiscRead = errors.New("cannot read titles from disc")
