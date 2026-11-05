package hmkv

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// findDiscMountPoint returns the directory where the file system of the disc in the given drive
// is mounted. devicePath is the drive path reported by makemkvcon (DRV line, last field), e.g.
// "/dev/sr0" on Linux, "/dev/rdisk4" on macOS or "E:" on Windows.
func findDiscMountPoint(devicePath string) (string, error) {
	devicePath = strings.TrimSpace(devicePath)
	if devicePath == "" {
		return "", fmt.Errorf("makemkvcon did not report a device path for the drive")
	}

	switch runtime.GOOS {
	case "windows":
		return windowsDriveRoot(devicePath)
	case "darwin":
		return darwinMountPoint(devicePath)
	default:
		f, err := os.Open("/proc/mounts")
		if err != nil {
			return "", fmt.Errorf("could not read /proc/mounts: %w", err)
		}
		defer f.Close()
		return mountPointFromProcMounts(bufio.NewScanner(f), devicePath)
	}
}

var windowsDriveLetter = regexp.MustCompile(`^([A-Za-z]):`)

func windowsDriveRoot(devicePath string) (string, error) {
	m := windowsDriveLetter.FindStringSubmatch(devicePath)
	if m == nil {
		return "", fmt.Errorf("could not determine the drive letter from %q", devicePath)
	}
	root := strings.ToUpper(m[1]) + `:\`
	if _, err := os.Stat(root); err != nil {
		return "", fmt.Errorf("drive %s is not accessible: %w", root, err)
	}
	return root, nil
}

// mountPointFromProcMounts searches /proc/mounts style content for the device.
func mountPointFromProcMounts(scanner *bufio.Scanner, devicePath string) (string, error) {
	wanted := map[string]struct{}{devicePath: {}}
	if resolved, err := filepath.EvalSymlinks(devicePath); err == nil {
		wanted[resolved] = struct{}{}
	}

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		dev := unescapeMountField(fields[0])
		if _, ok := wanted[dev]; !ok {
			if resolved, err := filepath.EvalSymlinks(dev); err != nil {
				continue
			} else if _, ok := wanted[resolved]; !ok {
				continue
			}
		}
		return unescapeMountField(fields[1]), nil
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("disc in %s is not mounted", devicePath)
}

// unescapeMountField decodes the octal escapes (e.g. "\040" for a space) used in /proc/mounts.
func unescapeMountField(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+4 <= len(s) {
			if n, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
				sb.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		sb.WriteByte(s[i])
	}
	return sb.String()
}

var darwinMountLine = regexp.MustCompile(`^(\S+) on (.+) \([^)]*\)$`)

func darwinMountPoint(devicePath string) (string, error) {
	// makemkvcon reports the raw device (/dev/rdiskN), mount lists the block device (/dev/diskN).
	dev := strings.Replace(devicePath, "/dev/rdisk", "/dev/disk", 1)

	out, err := exec.Command("mount").Output()
	if err != nil {
		return "", fmt.Errorf("could not run mount: %w", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		m := darwinMountLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		if m[1] == dev || strings.HasPrefix(m[1], dev+"s") {
			return m[2], nil
		}
	}

	return "", fmt.Errorf("disc in %s is not mounted", devicePath)
}
