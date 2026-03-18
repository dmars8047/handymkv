# HandyMKV

A MakeMKV + HandBrake productivity tool.

## Description

HandyMKV is a tool that is designed to automate the process of ripping discs using MakeMKV and then encoding the resulting files using Handbrake.

#### Why I Created HandyMKV

I found the process of manually ripping using MakeMKV and then encoding using HandBrake to be time consuming, disjointed, and error prone. I wanted a tool that would automate the process and provide a more user-friendly experience. Additionally, I wanted to offload the process from my main desktop computer to my home server which is headless and does not have a GUI. HandyMKV was created to address these needs.

As I developed HandyMKV, I found that I was able to add features that I found useful and that made the process faster and easier. I hope that others will find HandyMKV useful and that it will save them time and effort.

## Features

- Rip titles from discs using MakeMKV
- Encode video files using HandBrake
- Flexible configuration options
- Clear and concise progress display
- Concurrency to reduce overall processing time
- Summary of space saved and time elapsed
- Automated cleanup of raw unencoded files
- Run history — browse and inspect past ripping/encoding sessions
- Parsing of `HandBrakeCLI` and `makemkvcon` output to provide a more user-friendly experience

## Objectives

### Time Saving

Its primary aim is to save time and effort by removing the disconnected nature of performing these tasks manually and/or sequentially. Concurrency is used to reduce the overall time taken to complete the process.

### Ease of Use

HandyMKV is designed to be easy to setup and use. Once a configuration has been created the process of ripping and encoding a disc is simplified to a single command and a few prompts.

Output from the process is displayed in a clear and concise manner to keep the user informed of the progress of the tasks.

### Flexibility

HandyMKV is designed to be flexible. The user can select which titles to rip from the disc and can configure the encoding options to suit their needs. Encoding options can be setup in three ways: 

- Using HandyMKV Simplified Encoding Options - Essentially a collection of settings which are meant to address the most common use cases. These settings are designed to be easy to use and understand. These are gathered via a series of prompts during the configuration process with sensible defaults (where possible).
- Using Built-in HandBrake Presets - HandyMKV can be setup to use a specific built-in HandBrake preset. This option is for users who are familiar with HandBrake and have a specific preset they want to use.
- Using a Custom HandBrake Preset File - HandyMKV can be setup to use a custom HandBrake preset file. This option offers the most granular control over the encoding process but requires the user to create a HandBrake preset file.

## Prerequisites

### Supported Operating Systems

HandyMKV is designed to work on Windows, MacOS, and Linux.

### MakeMKV

MakeMKV is a tool that is used to rip the contents of a disc to a file on the local system. Note: this tool is not free and a license must be purchased for use.

Specifically the `makemkvcon` command is used to interact with MakeMKV from the command line. The `makemkvcon` command must be in the system path for HandyMKV to work.

MakeMKV can be downloaded from [here](https://www.makemkv.com/).

Documentation for makemkvcon can be found [here](https://www.makemkv.com/developers/usage.txt).

#### A Note for Mac Users 

`makemkvcon` is not included in the $PATH by default when MakeMKV is installed. However, the binary is included in the MakeMKV.app bundle. The binary can be found at `/Applications/MakeMKV.app/Contents/MacOS/makemkvcon`. You can add this to your $PATH or create a symlink to it in a directory that is in your $PATH.

### Handbrake

Handbrake is a tool that is used to encode video files. The `HandBrakeCLI` command is used to interact with Handbrake from the command line. The `HandBrakeCLI` command must be in the system path for HandyMKV to work.

The HandBrakeCLI can be installed from [here](https://handbrake.fr/downloads2.php). Note: HandBrake is free to use. See the HandBrake website for more information.

Documentation for the HandBrakeCLI can be found [here](https://handbrake.fr/docs/en/latest/cli/cli-guide.html). This is not needed for usage with HandyMKV but is useful for context.

## Command Line Options

HandyMKV has a number of command line options and subcommands that can be used to control its behavior.

```shell
Usage of handymkv:
  -c    Configure. Runs the configuration wizard.
  -d string
        Discs. A comma delimited list of disc indexes to rip. Example: -d 0,1,2 (default "0")
  -l    List. Lists the available discs. The disc index is required to rip a disc. Drives without a valid disc inserted will not be listed.
  -r    Read. Reads and outputs the first encountered configuration file. The current working directory is searched first, then the user-level configuration.
  -v    Version. Prints the version of the application.

Subcommands:
  history             Show a summary list of past runs.
  history <number>    Show details for a specific past run.
  history clear       Delete all manifest files from the run history directory.
```

## Installation

### Install Script (Linux and macOS)

The quickest way to install HandyMKV on Linux or macOS is with the install script:

```shell
curl -fsSL https://raw.githubusercontent.com/dmars8047/handymkv/release/install.sh | bash
```

The script automatically detects your OS and architecture, downloads the appropriate binary from the latest release, and installs it to `/usr/local/bin` (or `~/.local/bin` if `/usr/local/bin` is not writable).

### Pre-built Binaries

Pre-built binaries are available on the [Releases](https://github.com/dmars8047/handymkv/releases) page for the following platforms:

- Linux (AMD64, ARM64)
- macOS (Intel, Apple Silicon)
- Windows (AMD64, ARM64)

Download the appropriate binary for your system, place it in a directory on your `$PATH`, and you're ready to go.

### Installing with Go

HandyMKV can be installed using the `go install` command:

```shell
go install github.com/dmars8047/handymkv/cmd/handymkv@latest
```

This requires Go to be installed on the system. Go can be installed from [here](https://go.dev/doc/install).

### Building from Source

If you have cloned the repository, you can build HandyMKV using the provided Makefile or Go commands directly.

#### Using Make (Recommended)

The Makefile provides convenient targets for building HandyMKV for various platforms:

**Build for your current system:**
```shell
make current
```
This will create a binary in `bin/handymkv` (or `bin/handymkv.exe` on Windows).

**Install to your GOPATH/bin:**
```shell
make install
```
This installs the binary to your Go bin directory, making it available system-wide.

**Cross-compile for all supported platforms:**
```shell
make all
```
This builds binaries for Linux, macOS, and Windows (both AMD64 and ARM64 architectures) in separate subdirectories under `bin/`.

**Cross-compile for a specific platform:**
```shell
make linux-amd64      # Linux AMD64
make linux-arm64      # Linux ARM64
make darwin-amd64     # macOS Intel
make darwin-arm64     # macOS Apple Silicon
make windows-amd64    # Windows AMD64
make windows-arm64    # Windows ARM64
```

**Clean build artifacts:**
```shell
make clean
```

**View all available targets:**
```shell
make help
```

#### Using Go Commands Directly

Alternatively, you can build using Go commands:

```shell
# Build for current system
go build -o bin/handymkv ./cmd/handymkv

# Install to GOPATH/bin
go install ./cmd/handymkv

# Cross-compile (example for Linux AMD64)
GOOS=linux GOARCH=amd64 go build -o bin/linux-amd64/handymkv ./cmd/handymkv
```

## Basic Usage

The first step is to create a configuration file. This can be done by running the following command:

```shell
handymkv -c
```

This will start the configuration wizard. It will prompt you for encode settings and various operational settings. Once saved, the configuration will be stored in a file called `config.json`. The location of that file depends on whether user-wide or directory-wide configuration is used.

- On Unix systems, the user-wide configuration file is stored at '~/.config/handymkv/config.json'.
- On Windows systems, the user-wide configuration file is stored at '%APPDATA%\handymkv\config.json'.

Then to rip and encode a disc, run the following command:

```shell
handymkv
```

This will first read the titles on the disc and prompt you to select which titles to rip. Titles are selected by providing the index of each title. Multiple titles can be selected by providing a comma delimited list. Example: `0, 1, 3,4`. Once you have selected the titles to rip, the process will begin. The progress of the process will be displayed in the terminal.

![alt text](https://github.com/dmars8047/handymkv/blob/develop/doc/handymkv_process_in_progress.png?raw=true)

Once the process is complete, a summary will be displayed showing the space saved and the time taken to complete the process.

All output files will be stored in the directory specified in the configuration file.

Note: If there is a `config.json` file in the working directory at execution time, that file will be used instead of the user-wide configuration file.

## Run History

After each run, HandyMKV writes a manifest file recording what was ripped and encoded, file sizes, durations, and whether raw MKV files were deleted. These manifests can be browsed at any time with the `history` subcommand.

**List all past runs:**
```shell
handymkv history
```

**Inspect a specific run:**
```shell
handymkv history 3
```

**Clear all saved history:**
```shell
handymkv history clear
```
This will show the number of files to be deleted and prompt for confirmation before proceeding.

### Manifest File Location

By default, manifest files are stored alongside the main configuration:

- Unix: `~/.config/handymkv/manifests/`
- Windows: `%APPDATA%\handymkv\manifests\`

A custom directory can be set during the configuration wizard (`handymkv -c`), or by setting `manifest_directory` in `config.json`.

### Disabling Run History

Run history can be disabled entirely via the configuration wizard or by setting `"disable_manifests": true` in `config.json`. When disabled, `handymkv history` and `handymkv history clear` will display an informational message rather than attempting to read or modify manifest files.

## Multi-Disc Support

HandyMKV supports ripping and encoding multiple discs in a single run. This option is intended for when mutliple disc drives are available and connected to the host.

During such multi-disc runs the output files will be sorted into subdirectories that indicate the disc they originated from.

To rip and encode multiple discs, simply provide a comma delimited list of disc indexes to the `-d` flag. Example: `handymkv -d 0,1,2`.

To see a list of available discs, use the `-l` flag. Example: `handymkv -l`.

## A Note on Concurrency

HandyMKV will attempt to execute tasks concurrently to reduce the overall time taken to complete the process. However, encoding tasks are resource intensive and running multiple encoding tasks is likely to slow down the overall process. Likewise ripping tasks are bottle-necked by the speed of the disc drive. For this reason HandyMKV will execute ripping and encoding pipelines concurrently but each task in those pipelines will be executed sequentially. In multi-disc runs, each disc drive's ripping process will be processed concurrently.

## Support

If you find HandyMKV useful, please consider supporting the project:

[Buy me a coffee on Ko-fi](https://ko-fi.com/dmars8047)
