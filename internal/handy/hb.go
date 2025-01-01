package handy

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

func parseVideoEncoders(text string) []string {
	var encoders []string
	lines := strings.Split(text, "\n")
	start := false

	for _, line := range lines {
		if strings.Contains(line, "Select video encoder:") {
			start = true
			continue
		}
		if start {
			if strings.TrimSpace(line) == "" {
				break
			}
			encoder := strings.TrimSpace(line)
			if encoder != "" {
				encoders = append(encoders, encoder)
			}
		}
	}
	return encoders
}
