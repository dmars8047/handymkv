package hmkv

import (
	"slices"
	"testing"
)

func TestBuildEncodeArgs(t *testing.T) {
	tests := []struct {
		name     string
		params   EncodingParams
		expected []string
	}{
		{
			name: "custom preset file",
			params: EncodingParams{
				MKVOutputPath:       "in.mkv",
				HandBrakeOutputPath: "out.mkv",
				Preset:              "MyPreset",
				PresetFile:          "preset.json",
			},
			expected: []string{
				"--input", "in.mkv",
				"--output", "out.mkv",
				"--preset-import-file", "preset.json",
				"--preset", "MyPreset",
			},
		},
		{
			name: "built-in preset takes no import file",
			params: EncodingParams{
				MKVOutputPath:       "in.mkv",
				HandBrakeOutputPath: "out.mkv",
				Preset:              "Fast 1080p30",
			},
			expected: []string{
				"--input", "in.mkv",
				"--output", "out.mkv",
				"--preset", "Fast 1080p30",
			},
		},
		{
			name: "simplified settings with encoder preset",
			params: EncodingParams{
				MKVOutputPath:               "in.mkv",
				HandBrakeOutputPath:         "out.mkv",
				Encoder:                     "x265_10bit",
				EncoderPreset:               "slow",
				SubtitleLanguages:           []string{"eng", "jpn"},
				IncludeAllRelevantSubtitles: true,
				AudioLanguages:              []string{"eng"},
				IncludeAllRelevantAudio:     true,
			},
			expected: []string{
				"--input", "in.mkv",
				"--output", "out.mkv",
				"--encoder", "x265_10bit",
				"--encoder-preset", "slow",
				"--subtitle-lang-list", "eng,jpn",
				"--all-subtitles",
				"--audio-lang-list", "eng",
				"--all-audio",
			},
		},
		{
			name: "simplified settings fall back to numeric quality",
			params: EncodingParams{
				MKVOutputPath:       "in.mkv",
				HandBrakeOutputPath: "out.mkv",
				Encoder:             "x265",
				Quality:             22,
			},
			expected: []string{
				"--input", "in.mkv",
				"--output", "out.mkv",
				"--encoder", "x265",
				"--quality", "22",
			},
		},
		{
			name: "language lists are omitted when empty",
			params: EncodingParams{
				MKVOutputPath:       "in.mkv",
				HandBrakeOutputPath: "out.mkv",
				Encoder:             "x264",
				Quality:             20,
				SubtitleLanguages:   []string{},
				AudioLanguages:      []string{},
			},
			expected: []string{
				"--input", "in.mkv",
				"--output", "out.mkv",
				"--encoder", "x264",
				"--quality", "20",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := buildEncodeArgs(&test.params)

			if !slices.Equal(actual, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}
