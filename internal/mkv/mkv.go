package mkv

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// TINFO:id,code,value
// TINFO:0,2,0,"Breaking Bad: Season 1: Disc 1"
// TINFO:0,8,0,"7"
// TINFO:0,9,0,"0:58:06"
// TINFO:0,10,0,"12.5 GB"
// TINFO:0,11,0,"13472686080"
// TINFO:0,16,0,"00763.mpls"
// TINFO:0,25,0,"1"
// TINFO:0,26,0,"262"
// TINFO:0,27,0,"Breaking_Bad_Season_1_Disc_1_t00.mkv"
// TINFO:0,28,0,"eng"
// TINFO:0,29,0,"English"
// TINFO:0,30,0,"Breaking Bad: Season 1: Disc 1 - 7 chapter(s) , 12.5 GB"
// TINFO:0,31,6120,"<b>Title information</b><br>"
// TINFO:0,33,0,"0"

// 2 - Disc Title
// 8 - Number of chapters in file
// 9 - Length of file in seconds
// 10 - File size in GB
// 11 - File size in bytes
// 27 - File name
// 28 - audio short code
// 29 - audio long code
type TitleInfo struct {
	// Index on disc
	Index     int
	DiscTitle string
	Chapters  int
	Length    string
	FileSize  string
	FileName  string
}

func RipTitle(ctx context.Context, title *TitleInfo, destDir string) error {

	cmdOut, err := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("makemkvcon mkv disc:0 %d %s", title.Index, destDir)).Output()

	if err != nil {
		return err
	}

	success := strings.Contains(string(cmdOut), "Copy complete. 1 titles saved.")

	if !success {
		return fmt.Errorf("an error occurred while ripping title from disc")
	}

	return nil
}

func GetTitlesFromDisc() ([]TitleInfo, error) {
	titles := make([]TitleInfo, 0)

	// Run the command to get the output
	cmdOut, err := exec.Command("sh", "-c", "makemkvcon -r info disc:0").Output()

	if err != nil {
		return titles, fmt.Errorf("error running command: %v", err)
	}

	// Parse the output by splitting into lines
	lines := strings.Split(string(cmdOut), "\n")

	// Temporary variables to hold extracted data for each title
	titleData := make(map[int]*TitleInfo)

	for _, line := range lines {
		// Extract the title index (e.g., TINFO:0, TINFO:1)
		if strings.HasPrefix(line, "TINFO:") {
			parts := strings.SplitN(line, ",", 4)
			if len(parts) < 4 {
				continue
			}
			index, _ := strconv.Atoi(strings.TrimPrefix(parts[0], "TINFO:"))
			code := parts[1]
			value := strings.Trim(parts[3], "\"")

			// Ensure the titleData map has an entry for this title index
			if titleData[index] == nil {
				titleData[index] = &TitleInfo{
					Index: index,
				}
			}

			// Populate the relevant field based on the code
			switch code {
			case "2": // Disc Title
				titleData[index].DiscTitle = value
			case "8": // Number of Chapters
				if chapters, err := strconv.Atoi(value); err == nil {
					titleData[index].Chapters = chapters
				}
			case "9": // Length
				titleData[index].Length = value
			case "10": // File Size
				titleData[index].FileSize = value
			case "27": // File Name
				titleData[index].FileName = value
			}
		}
	}

	// Convert the map to a slice
	for _, title := range titleData {
		titles = append(titles, *title)
	}

	// Sort the titles by FileName
	sort.Slice(titles, func(i, j int) bool {
		return titles[i].FileName < titles[j].FileName
	})

	return titles, nil
}