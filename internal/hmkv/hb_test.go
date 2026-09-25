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
		{
			name:      "-Z is reserved as the short form of --preset",
			args:      []string{"-Z", "Fast 1080p30"},
			expectErr: true,
		},
		{
			name:      "-o with an attached value is reserved",
			args:      []string{"-o/tmp/elsewhere.mkv"},
			expectErr: true,
		},
		{
			name:      "-i with an attached value is reserved",
			args:      []string{"-i/tmp/other.mkv"},
			expectErr: true,
		},
		{
			name:      "-Z with an attached value is reserved",
			args:      []string{"-ZOther"},
			expectErr: true,
		},
		{
			// getopt reads -O as a flag without a value and continues with the
			// rest of the token, so this is -O followed by -o /tmp/elsewhere.mkv.
			name:      "-o clustered behind a flag without a value is reserved",
			args:      []string{"-Oo/tmp/elsewhere.mkv"},
			expectErr: true,
		},
		{
			name:      "-o clustered behind -g is reserved",
			args:      []string{"-go/tmp/elsewhere.mkv"},
			expectErr: true,
		},
		{
			// HandBrakeCLI accepts any unambiguous prefix of a long option.
			name:      "an abbreviated --input is reserved",
			args:      []string{"--inp", "/tmp/other.mkv"},
			expectErr: true,
		},
		{
			name:      "an abbreviated --output in inline form is reserved",
			args:      []string{"--outp=/tmp/elsewhere.mkv"},
			expectErr: true,
		},
		{
			name:      "--queue-import-file is reserved",
			args:      []string{"--queue-import-file", "queue.json"},
			expectErr: true,
		},
		{
			name:      "--preset-import-gui is reserved",
			args:      []string{"--preset-import-gui"},
			expectErr: true,
		},
		{
			// The file extension comes from output_file_format, so a different
			// container would end up in a file named for another one.
			name:      "--format is reserved",
			args:      []string{"--format", "av_mp4"},
			expectErr: true,
		},
		{
			name:      "-f is reserved",
			args:      []string{"-f", "av_mp4"},
			expectErr: true,
		},
		{
			name:      "an abbreviated --format is reserved",
			args:      []string{"--form=av_mp4"},
			expectErr: true,
		},
		{
			// A short option token is rejected as soon as it contains a reserved
			// letter anywhere, since telling an attached value apart from a
			// cluster would need HandBrakeCLI's full option table. The fix on the
			// user's side is to pass the value as its own token: "-E", "opus".
			name:      "a harmless attached value containing a reserved letter is rejected",
			args:      []string{"-Eopus"},
			expectErr: true,
		},
		{
			name:      "an attached value without reserved letters is allowed",
			args:      []string{"-Eac3"},
			expectErr: false,
		},
		{
			name:      "a negative number as a value is allowed",
			args:      []string{"--gain", "-10"},
			expectErr: false,
		},
		{
			name:      "long options that merely share a prefix with reserved ones are allowed",
			args:      []string{"--previews", "10:0", "--optimize", "--inline-parameter-sets"},
			expectErr: false,
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
