package hmkv

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

const handBrakeCLIExecutable = "HandBrakeCLI"

func GetHandBrakeCLIExecutable() (string, error) {
	_, err := exec.LookPath(handBrakeCLIExecutable)

	if err == nil {
		return handBrakeCLIExecutable, nil
	}

	// Check if the executable is in the homedir/.handymkv/bin
	user, err := user.Current()

	if err != nil {
		return "", fmt.Errorf("could not get current user: %w", err)
	}

	path := filepath.Join(user.HomeDir, "handymkv", "bin", handBrakeCLIExecutable)

	// check if the file exists using stat
	_, err = os.Stat(path)

	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("handbrakecli executable not found")
}

type HandBrakeCLI struct {
	executable string
}

func NewHandBrakeCLI(executable string) *HandBrakeCLI {
	return &HandBrakeCLI{
		executable: executable,
	}
}

type EncodingParams struct {
	TitleIndex                  int      `json:"-"`
	MKVOutputPath               string   `json:"-"`
	HandBrakeOutputPath         string   `json:"-"`
	Encoder                     string   `json:"encoder,omitempty"`
	EncoderPreset               string   `json:"encoder_preset,omitempty"`
	Quality                     int      `json:"quality,omitempty"`
	SubtitleLanguages           []string `json:"subtitle_languages,omitempty"`
	IncludeAllRelevantSubtitles bool     `json:"include_all_relevant_subtitles,omitempty"`
	AudioLanguages              []string `json:"audio_languages,omitempty"`
	IncludeAllRelevantAudio     bool     `json:"include_all_relevant_audio,omitempty"`
	OutputFileFormat            string   `json:"output_file_format,omitempty"`
	Preset                      string   `json:"handbrake_preset,omitempty"`
	PresetFile                  string   `json:"preset_file,omitempty"`
}

type HandBrakePresetFile struct {
	PresetList []HandBrakePreset `json:"PresetList"`
}

type HandBrakePreset struct {
	PresetName string `json:"PresetName"`
	FileFormat string `json:"FileFormat"`
}

func (hb *HandBrakeCLI) encode(ctx context.Context, params *EncodingParams) error {
	var args []string = []string{
		"--input", params.MKVOutputPath,
		"--output", params.HandBrakeOutputPath,
	}

	if params.Preset != "" {
		if params.PresetFile != "" {
			args = append(args, "--preset-import-file", params.PresetFile)
		}

		args = append(args, "--preset", params.Preset)
	} else {
		args = append(args, "--encoder", params.Encoder)

		if params.EncoderPreset != "" {
			args = append(args, "--encoder-preset", params.EncoderPreset)
		} else {
			args = append(args, "--quality", strconv.Itoa(params.Quality))
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
	}

	cmd := exec.CommandContext(ctx, hb.executable,
		args...,
	)

	// Capture the combined output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return NewExternalProcessError(fmt.Errorf("an error occurred while encoding %s - handbrakecli failure: %w", params.MKVOutputPath, err),
			string(fmt.Sprintf("HandBrakeCLI Output\n----------------\n%s----------------\n\n", output)))
	}

	return nil
}

func (hb *HandBrakeCLI) getPossiblePresets() ([]string, error) {
	var presets []string

	cmd := exec.Command(hb.executable, "--preset-list")

	output, err := cmd.CombinedOutput()

	if err != nil {
		return presets, fmt.Errorf("handbrakecli failure: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "        ") {
			presets = append(presets, strings.TrimSpace(line))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading command output: %w", err)
	}

	return presets, nil
}

// Calls HandBrakeCLI --help and parses the output to get a list of possible encoders
func (hb *HandBrakeCLI) getPossibleEncoders() ([]string, error) {
	var encoders []string

	cmd := exec.Command(hb.executable, "--help")

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

// Calls HandBrakeCLI --encoder-preset-list and parses the output to get a list of possible quality presets for a given encoder
func (hb *HandBrakeCLI) getPossibleEncoderPresets(encoder string) ([]string, error) {
	var qualityPresets []string

	cmd := exec.Command(hb.executable, "--encoder-preset-list", encoder)

	output, err := cmd.CombinedOutput()

	if err != nil {
		return qualityPresets, fmt.Errorf("handbrakecli failure: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "        ") {
			qualityPresets = append(qualityPresets, strings.TrimSpace(scanner.Text()))
		}
	}

	return qualityPresets, nil
}
