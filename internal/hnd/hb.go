package hnd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type EncodingParams struct {
	TitleIndex                  int      `json:"-"`
	MKVOutputPath               string   `json:"-"`
	HandBrakeOutputPath         string   `json:"-"`
	Encoder                     string   `json:"encoder"`
	Quality                     int      `json:"quality"`
	SubtitleLanguages           []string `json:"subtitle_languages"`
	IncludeAllRelevantSubtitles bool     `json:"include_all_relevant_subtitles"`
	AudioLanguages              []string `json:"audio_languages"`
	IncludeAllRelevantAudio     bool     `json:"include_all_relevant_audio"`
}

func Encode(ctx context.Context, params *EncodingParams) error {
	var args []string = []string{
		"--input", params.MKVOutputPath,
		"--output", params.HandBrakeOutputPath,
		"--encoder", params.Encoder,
		"--quality", fmt.Sprintf("%d", params.Quality),
	}

	if len(params.SubtitleLanguages) > 0 {
		args = append(args, "--subtitle-lang-list", strings.Join(params.SubtitleLanguages, ","))

		if params.IncludeAllRelevantSubtitles {
			args = append(args, "--all-subtitles")
		}
	}

	if len(params.AudioLanguages) > 0 {
		args = append(args, "--audio-lang-list", strings.Join(params.AudioLanguages, ","))

		if params.IncludeAllRelevantAudio {
			args = append(args, "--all-audio")
		}
	}

	cmd := exec.CommandContext(ctx, "HandBrakeCLI",
		args...,
	)

	// Run the command and wait for it to complete
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("handbrakecli failure: %w", err)
	}

	return nil
}
