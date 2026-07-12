package hmkv

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

func GetHandBrakeCLIExecutable() (string, error) {

	handBrakeCLIExecutableName := "HandBrakeCLI"

	if runtime.GOOS == "windows" {
		handBrakeCLIExecutableName = fmt.Sprintf("%s%s", handBrakeCLIExecutableName, ".exe")
	}

	_, err := exec.LookPath(handBrakeCLIExecutableName)

	if err == nil {
		return handBrakeCLIExecutableName, nil
	}

	// Check if the executable is in the homedir/.handymkv/bin
	user, err := user.Current()

	if err != nil {
		return "", fmt.Errorf("could not get current user: %w", err)
	}

	path := filepath.Join(user.HomeDir, "handymkv", "bin", handBrakeCLIExecutableName)

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
	DiscId                      int      `json:"-"`
	MKVOutputPath               string   `json:"-"`
	HandBrakeOutputPath         string   `json:"-"`
	RippedFileSizeBytes         int64    `json:"-"`
	EncodedFileSizeBytes        int64    `json:"-"`
	RippingDuration             string   `json:"-"`
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

// Builds the full HandBrakeCLI argument list for an encode.
func buildEncodeArgs(params *EncodingParams) []string {
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

	return args
}

func (hb *HandBrakeCLI) encode(ctx context.Context,
	params *EncodingParams,
	onProgress func(percent int)) error {
	args := buildEncodeArgs(params)

	cmd := exec.CommandContext(ctx, hb.executable,
		args...,
	)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()

	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start HandBrakeCLI: %w", err)
	}

	// Capture stderr in background while parsing stdout for progress
	var stderrBuf bytes.Buffer
	stderrDone := make(chan struct{})
	go func() {
		io.Copy(&stderrBuf, stderr)
		close(stderrDone)
	}()

	// Format: "Encoding: task 1 of 1, 20.06 % (421.16 fps, avg 413.85 fps, ETA 00h02m29s)"
	progressRegex := regexp.MustCompile(`Encoding:.*?(\d+\.?\d*)\s*%`)
	scanner := bufio.NewScanner(stdout)

	// Custom split function to yield on \r (carriage return) for real-time progress
	// HandBrake uses \r for in-place updates, default scanner waits for \n
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		for i, b := range data {
			if b == '\r' || b == '\n' {
				return i + 1, data[0:i], nil
			}
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

	for scanner.Scan() {
		line := scanner.Text()
		if matches := progressRegex.FindStringSubmatch(line); len(matches) > 1 {
			if percent, err := strconv.ParseFloat(matches[1], 64); err == nil {
				onProgress(int(percent))
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading HandBrakeCLI output: %w", err)
	}

	// Wait for stderr capture to complete
	<-stderrDone

	if err := cmd.Wait(); err != nil {
		stderrOutput := stderrBuf.String()
		return NewExternalProcessError(
			fmt.Errorf("encoding failed for %s: %w", params.MKVOutputPath, err),
			fmt.Sprintf("HandBrakeCLI Error Output:\n%s", stderrOutput),
		)
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
