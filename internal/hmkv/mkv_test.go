package hmkv

import "testing"

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
