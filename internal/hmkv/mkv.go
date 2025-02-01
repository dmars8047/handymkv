package hmkv

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// 2 - Disc Title
// 8 - Number of chapters in file
// 9 - Length of file in seconds
// 10 - File size (GB)
// 11 - File size (Bytes)
// 27 - File name
// 28 - Audio Short Code
// 29 - Audio Long Code
type TitleInfo struct {
	// Index on disc
	Index     int
	DiscTitle string
	DiscId    int
	Chapters  int
	Length    string
	FileSize  string
	FileName  string
}

func ripTitle(ctx context.Context, title *TitleInfo, destDir string, discId int) error {
	cmd := exec.CommandContext(ctx, "makemkvcon", "mkv", fmt.Sprintf("disc:%d", discId), fmt.Sprintf("%d", title.Index), destDir)

	cmdOut, err := cmd.Output()
	if err != nil {
		return err
	}

	success := strings.Contains(string(cmdOut), "Copy complete. 1 titles saved.")

	if !success {
		// write cmdOut to a log file in the dest dir
		logFilePath := filepath.Join(destDir, "rip_err.log")
		f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil {
			return fmt.Errorf("ripping title from disc was not successful - an error occured while creating log file: %w", err)
		}

		defer f.Close()

		if _, err := f.WriteString(string(cmdOut)); err != nil {
			return fmt.Errorf("failed to write to log file: %w", err)
		}

		return fmt.Errorf("ripping title from disc was not successful - mkv error details can be found in log file %s", logFilePath)
	}

	return nil
}

func getTitlesFromDisc(discId int) ([]TitleInfo, error) {
	titles := make([]TitleInfo, 0)

	// Run the command to get the output
	cmdOut, err := exec.Command("makemkvcon", "-r", "info", fmt.Sprintf("disc:%d", discId)).Output()

	if err != nil {
		return titles, fmt.Errorf("error running command: %w", err)
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

			// Windows carriage return fix
			if runtime.GOOS == "windows" {
				value = strings.TrimRight(value, "\"\r")
			}

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
		title.DiscId = discId
		titles = append(titles, *title)
	}

	// Sort the titles by FileName
	sort.Slice(titles, func(i, j int) bool {
		return titles[i].FileName < titles[j].FileName
	})

	return titles, nil
}

func getTitles(discId int) ([]TitleInfo, error) {
	titles, err := getTitlesFromDisc(discId)

	if err != nil {
		return titles, fmt.Errorf("an error occurred while reading titles from disc - %w", err)
	}

	if len(titles) < 1 {
		return titles, NewDiscError(discId, "no titles found on disc")
	}

	return titles, nil
}

// makemkvcon -r --cache=1 info disc:9999

// Example Output of makemkvcon:

// DRV:0,2,999,12,"BD-RE HL-DT-ST BD-RE  BH16NS40 1.05 KLZK7UI0426","STAR TREK TNG S4 D2","/dev/sr0"
// DRV:1,256,999,0,"","",""
// DRV:2,256,999,0,"","",""
// DRV:3,256,999,0,"","",""
// DRV:4,256,999,0,"","",""
// DRV:5,256,999,0,"","",""
// DRV:6,256,999,0,"","",""
// DRV:7,256,999,0,"","",""
// DRV:8,256,999,0,"","",""
// DRV:9,256,999,0,"","",""
// DRV:10,256,999,0,"","",""
// DRV:11,256,999,0,"","",""
// DRV:12,256,999,0,"","",""
// DRV:13,256,999,0,"","",""
// DRV:14,256,999,0,"","",""
// DRV:15,256,999,0,"","",""

type DiscInfo struct {
	Index int
	Name  string
}

func ListDiscs() ([]DiscInfo, error) {
	cmdOut, err := exec.Command("makemkvcon", "-r", "--cache=1", "info", "disc:9999").Output()

	if err != nil {
		return nil, fmt.Errorf("error running command: %w", err)
	}

	// Parse the output by splitting into lines
	lines := strings.Split(string(cmdOut), "\n")

	// Temporary variables to hold extracted data for each title
	drives := make([]DiscInfo, 0)

	for _, line := range lines {
		// Extract the title index (e.g., TINFO:0, TINFO:1)
		if strings.HasPrefix(line, "DRV:") {
			parts := strings.Split(line, ",")

			if len(parts) != 7 || parts[5] == "\"\"" {
				continue
			}

			discIndexString := parts[0]

			discIndexString = strings.TrimPrefix(discIndexString, "DRV:")

			discIndex, err := strconv.Atoi(discIndexString)

			if err != nil {
				return nil, fmt.Errorf("error parsing disc index: %w", err)
			}

			driveName := strings.Trim(parts[5], "\"")

			drives = append(drives, DiscInfo{
				Index: discIndex,
				Name:  driveName,
			})
		}
	}

	return drives, nil
}
