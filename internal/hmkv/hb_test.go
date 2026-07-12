package hmkv

import (
	"errors"
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
		{
			// Extra arguments must come after --preset. HandBrakeCLI applies an
			// imported preset first and lets later flags override it, so ordering
			// is what makes the override work at all.
			name: "extra arguments are appended after the preset",
			params: EncodingParams{
				MKVOutputPath:       "in.mkv",
				HandBrakeOutputPath: "out.mkv",
				Preset:              "MyPreset",
				PresetFile:          "preset.json",
				ExtraHandBrakeArgs:  []string{"-a", "1,2", "--mixdown", "mono,5point1"},
			},
			expected: []string{
				"--input", "in.mkv",
				"--output", "out.mkv",
				"--preset-import-file", "preset.json",
				"--preset", "MyPreset",
				"-a", "1,2",
				"--mixdown", "mono,5point1",
			},
		},
		{
			name: "extra arguments also apply to simplified settings",
			params: EncodingParams{
				MKVOutputPath:       "in.mkv",
				HandBrakeOutputPath: "out.mkv",
				Encoder:             "x265",
				Quality:             20,
				ExtraHandBrakeArgs:  []string{"--audio-fallback", "ac3"},
			},
			expected: []string{
				"--input", "in.mkv",
				"--output", "out.mkv",
				"--encoder", "x265",
				"--quality", "20",
				"--audio-fallback", "ac3",
			},
		},
		{
			name: "no extra arguments leaves the argument list untouched",
			params: EncodingParams{
				MKVOutputPath:       "in.mkv",
				HandBrakeOutputPath: "out.mkv",
				Preset:              "MyPreset",
				ExtraHandBrakeArgs:  []string{},
			},
			expected: []string{
				"--input", "in.mkv",
				"--output", "out.mkv",
				"--preset", "MyPreset",
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

func TestValidateExtraHandBrakeArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			name:      "nil arguments are valid",
			args:      nil,
			expectErr: false,
		},
		{
			name:      "track selection and encoder overrides are allowed",
			args:      []string{"-a", "1,2", "-E", "ca_aac,ca_aac", "-B", "160,640"},
			expectErr: false,
		},
		{
			// The check does not track which tokens are flags and which are values,
			// so a value that happens to equal a reserved flag is rejected too. That
			// is deliberate: no HandBrakeCLI option takes "-i" or "--output" as its
			// value, and refusing loudly beats silently redirecting an encode.
			name:      "a reserved flag is rejected even in value position",
			args:      []string{"--mixdown", "mono", "--encoder-preset", "-i"},
			expectErr: true,
		},
		{
			name:      "--output is reserved",
			args:      []string{"--output", "/tmp/elsewhere.mkv"},
			expectErr: true,
		},
		{
			name:      "-o is reserved",
			args:      []string{"-o", "/tmp/elsewhere.mkv"},
			expectErr: true,
		},
		{
			name:      "--input is reserved",
			args:      []string{"--input", "/tmp/other.mkv"},
			expectErr: true,
		},
		{
			name:      "--preset is reserved",
			args:      []string{"--preset", "Fast 1080p30"},
			expectErr: true,
		},
		{
			name:      "--preset-import-file is reserved",
			args:      []string{"--preset-import-file", "other.json"},
			expectErr: true,
		},
		{
			name:      "a reserved flag in inline form is reserved",
			args:      []string{"--output=/tmp/elsewhere.mkv"},
			expectErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateExtraHandBrakeArgs(test.args)

			if test.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}

				if !errors.Is(err, ErrReservedHandBrakeArg) {
					t.Errorf("expected ErrReservedHandBrakeArg, got %v", err)
				}

				return
			}

			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}
