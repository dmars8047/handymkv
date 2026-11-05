package hmkv

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetEncodingFileName(t *testing.T) {
	tests := []struct {
		name             string
		fileName         string
		outputFileFormat string
		want             string
	}{
		{
			name:             "no format set keeps mkv",
			fileName:         "movie.mkv",
			outputFileFormat: "",
			want:             "movie.mkv",
		},
		{
			name:             "format=mkv keeps mkv",
			fileName:         "movie.mkv",
			outputFileFormat: "mkv",
			want:             "movie.mkv",
		},
		{
			name:             "format=mp4 replaces extension",
			fileName:         "movie.mkv",
			outputFileFormat: "mp4",
			want:             "movie.mp4",
		},
		{
			name:             "format=webm replaces extension",
			fileName:         "movie.mkv",
			outputFileFormat: "webm",
			want:             "movie.webm",
		},
		{
			name:             "spaces replaced with underscores",
			fileName:         "Star Trek TNG.mkv",
			outputFileFormat: "",
			want:             "Star_Trek_TNG.mkv",
		},
		{
			name:             "spaces replaced and extension changed",
			fileName:         "Star Trek TNG.mkv",
			outputFileFormat: "mp4",
			want:             "Star_Trek_TNG.mp4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := &TitleInfo{FileName: tt.fileName}
			config := &handyMKVConfig{
				EncodeConfig: EncodingParams{
					OutputFileFormat: tt.outputFileFormat,
				},
			}
			got := title.GetEncodingFileName(config)
			if got != tt.want {
				t.Errorf("GetEncodingFileName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubdirectory(t *testing.T) {
	tests := []struct {
		name             string
		discTitle        string
		discId           int
		prependDiscToSub bool
		want             string
	}{
		{
			name:             "no prepend replaces spaces",
			discTitle:        "Star Trek TNG",
			discId:           0,
			prependDiscToSub: false,
			want:             "Star_Trek_TNG",
		},
		{
			name:             "prepend disc 0",
			discTitle:        "Star Trek TNG",
			discId:           0,
			prependDiscToSub: true,
			want:             "HMKV_DISC_0__Star_Trek_TNG",
		},
		{
			name:             "prepend disc 2",
			discTitle:        "Movie Title",
			discId:           2,
			prependDiscToSub: true,
			want:             "HMKV_DISC_2__Movie_Title",
		},
		{
			name:             "no spaces in title",
			discTitle:        "Aliens",
			discId:           1,
			prependDiscToSub: false,
			want:             "Aliens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := &TitleInfo{
				DiscTitle:        tt.discTitle,
				DiscId:           tt.discId,
				PrependDiscToSub: tt.prependDiscToSub,
			}
			got := title.Subdirectory()
			if got != tt.want {
				t.Errorf("Subdirectory() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetTitlesFromDiscParsesSourceInfo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell script as fake makemkvcon")
	}

	script := filepath.Join(t.TempDir(), "makemkvcon")
	output := `DRV:0,2,999,12,"BD-RE HL-DT-ST","STAR TREK TNG S4 D2","/dev/sr0"
TINFO:0,8,0,"5"
TINFO:0,9,0,"0:45:10"
TINFO:0,11,0,"1000"
TINFO:0,16,0,"00800.mpls"
TINFO:0,24,0,"1"
TINFO:0,26,0,"1,2"
TINFO:0,27,0,"title_t00.mkv"
TINFO:1,24,0,"2"
TINFO:1,27,0,"title_t01.mkv"
`
	content := "#!/bin/sh\ncat <<'EOF'\n" + output + "EOF\n"
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	titles, err := NewMakeMKV(script).getTitlesFromDisc(0)
	if err != nil {
		t.Fatal(err)
	}

	if len(titles) != 2 {
		t.Fatalf("got %d titles", len(titles))
	}

	if titles[0].SourceFile != "00800.mpls" || titles[0].SegmentMap != "1,2" || titles[0].DevicePath != "/dev/sr0" {
		t.Errorf("unexpected title 0: %+v", titles[0])
	}

	// Without a source file name (DVD) the source title id is used.
	if titles[1].SourceFile != "2" {
		t.Errorf("unexpected title 1 source: %q", titles[1].SourceFile)
	}
}
