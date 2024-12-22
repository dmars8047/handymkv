package hb

import (
	"context"
	"fmt"
	"os/exec"
)

const (
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

type EncodingParams struct {
	TitleIndex        int      `json:"-"`
	InputFilePath     string   `json:"-"`
	OutputFilePath    string   `json:"-"`
	Encoder           string   `json:"encoder"`
	Quality           int      `json:"quality"`
	SubtitleLanguages []string `json:"subtitle_languages"`
	AudioLanguages    []string `json:"audio_languages"`
}

func Encode(ctx context.Context, params *EncodingParams) error {
	cmd := exec.CommandContext(ctx, "HandBrakeCLI",
		"--input", params.InputFilePath,
		"--output", params.OutputFilePath,
		"--encoder", params.Encoder,
		"--quality", fmt.Sprintf("%d", params.Quality),
		"--subtitle-lang-list", "eng,jpn",
		"--all-subtitles",
		"--audio-lang-list", "eng",
		"--all-audio",
	)

	// Run the command and wait for it to complete
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("HandBrakeCLI failed: %v", err)
	}

	return nil
}

// func Encode(ctx context.Context, params *EncodingParams) error {
// 	handbrakeCommand := exec.CommandContext(ctx, "HandBrakeCLI",
// 		"--input", params.InputFilePath,
// 		"--output", params.OutputFilePath,
// 		"--encoder", params.Encoder,
// 		"--quality", fmt.Sprintf("%d", params.Quality),
// 		"--subtitle-lang-list", "eng,jpn",
// 		"--all-subtitles",
// 		"--audio-lang-list", "eng",
// 		"--all-audio",
// 	)

// 	reader, err := handbrakeCommand.StdoutPipe()

// 	if err != nil {
// 		return err
// 	}

// 	// Start the command
// 	if err := handbrakeCommand.Start(); err != nil {
// 		return err
// 	}

// 	// Read the output from the pipe
// 	buf := make([]byte, 1024)

// 	for {
// 		n, err := reader.Read(buf)
// 		if n > 0 {
// 			fmt.Print(string(buf[:n]))
// 		}
// 		if err != nil {
// 			break
// 		}
// 	}

// 	if err := handbrakeCommand.Wait(); err != nil {
// 		return err
// 	}

// 	return nil
// }
