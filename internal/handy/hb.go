package handy

import (
	"bufio"
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

func encode(ctx context.Context, params *EncodingParams) error {
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

// Calls HandBrakeCLI --help and parses the output to get a list of possible encoders
func getPossibleEncoders() ([]string, error) {
	var encoders []string

	cmd := exec.Command("HandBrakeCLI", "--help")

	output, err := cmd.Output()

	if err != nil {
		return encoders, fmt.Errorf("handbrakecli failure: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	inEncoderSection := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, "Select video encoder:") {
			inEncoderSection = true
			continue
		}

		if inEncoderSection {
			if line == "" || strings.HasPrefix(line, "--") {
				break
			}
			encoders = append(encoders, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading command output: %w", err)
	}

	return encoders, nil
}
