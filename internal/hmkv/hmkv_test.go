package hmkv

import (
	"slices"
	"testing"
	"time"
)

// Guards the config-to-encode handoff. The encode parameters used to be
// assembled field by field, so a setting could be read from the config, shown
// by the config subcommand, and still never reach HandBrakeCLI.
func TestNewEncodingParamsCarriesConfiguredSettings(t *testing.T) {
	config := &handyMKVConfig{
		EncodeConfig: EncodingParams{
			Encoder:                     "x265_10bit",
			EncoderPreset:               "slow",
			Quality:                     22,
			SubtitleLanguages:           []string{"eng"},
			IncludeAllRelevantSubtitles: true,
			AudioLanguages:              []string{"eng", "jpn"},
			IncludeAllRelevantAudio:     true,
			OutputFileFormat:            "mkv",
			Preset:                      "MyPreset",
			PresetFile:                  "preset.json",
			ExtraHandBrakeArgs:          []string{"-a", "1,2", "--mixdown", "stereo,5point1"},
		},
	}

	title := &TitleInfo{Index: 3, DiscId: 1}

	params := newEncodingParams(config, title, "in.mkv", "out.mkv", 1234, 90*time.Second)

	if params.Encoder != "x265_10bit" {
		t.Errorf("expected encoder x265_10bit, got %q", params.Encoder)
	}

	if params.EncoderPreset != "slow" {
		t.Errorf("expected encoder preset slow, got %q", params.EncoderPreset)
	}

	if params.Quality != 22 {
		t.Errorf("expected quality 22, got %d", params.Quality)
	}

	if params.Preset != "MyPreset" || params.PresetFile != "preset.json" {
		t.Errorf("expected preset MyPreset from preset.json, got %q from %q", params.Preset, params.PresetFile)
	}

	if params.OutputFileFormat != "mkv" {
		t.Errorf("expected output file format mkv, got %q", params.OutputFileFormat)
	}

	if !slices.Equal(params.AudioLanguages, []string{"eng", "jpn"}) {
		t.Errorf("expected audio languages [eng jpn], got %v", params.AudioLanguages)
	}

	if !slices.Equal(params.SubtitleLanguages, []string{"eng"}) {
		t.Errorf("expected subtitle languages [eng], got %v", params.SubtitleLanguages)
	}

	if !params.IncludeAllRelevantAudio {
		t.Error("expected IncludeAllRelevantAudio to be carried over")
	}

	if !params.IncludeAllRelevantSubtitles {
		t.Error("expected IncludeAllRelevantSubtitles to be carried over")
	}

	if !slices.Equal(params.ExtraHandBrakeArgs, []string{"-a", "1,2", "--mixdown", "stereo,5point1"}) {
		t.Errorf("expected extra HandBrake arguments to be carried over, got %v", params.ExtraHandBrakeArgs)
	}

	// The per-title fields have no configured counterpart and must be filled in.
	if params.TitleIndex != 3 || params.DiscId != 1 {
		t.Errorf("expected title 3 of disc 1, got title %d of disc %d", params.TitleIndex, params.DiscId)
	}

	if params.MKVOutputPath != "in.mkv" || params.HandBrakeOutputPath != "out.mkv" {
		t.Errorf("expected in.mkv -> out.mkv, got %q -> %q", params.MKVOutputPath, params.HandBrakeOutputPath)
	}

	if params.RippedFileSizeBytes != 1234 {
		t.Errorf("expected ripped size 1234, got %d", params.RippedFileSizeBytes)
	}

	if params.RippingDuration != "1m30s" {
		t.Errorf("expected ripping duration 1m30s, got %q", params.RippingDuration)
	}
}

// The settings a user configures must survive all the way into the arguments
// HandBrakeCLI is actually invoked with.
func TestConfiguredExtraArgsReachHandBrake(t *testing.T) {
	config := &handyMKVConfig{
		EncodeConfig: EncodingParams{
			Preset:             "MyPreset",
			PresetFile:         "preset.json",
			ExtraHandBrakeArgs: []string{"-a", "1,2"},
		},
	}

	params := newEncodingParams(config, &TitleInfo{}, "in.mkv", "out.mkv", 0, 0)

	args := buildEncodeArgs(&params)

	expected := []string{
		"--input", "in.mkv",
		"--output", "out.mkv",
		"--preset-import-file", "preset.json",
		"--preset", "MyPreset",
		"-a", "1,2",
	}

	if !slices.Equal(args, expected) {
		t.Errorf("expected %v, got %v", expected, args)
	}
}
