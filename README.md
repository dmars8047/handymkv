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

### MakeMKV

MakeMKV is a tool that is used to rip the contents of a disc to a file on the local system. Note: this tool is not free and a license must be purchased for use.

Specifically the `makemkvcon` command is used to interact with MakeMKV from the command line. The `makemkvcon` command must be in the system path for HandyMKV to work.

MakeMKV can be downloaded from [here](https://www.makemkv.com/).

Documentation for makemkvcon can be found [here](https://www.makemkv.com/developers/usage.txt).

### Handbrake

Handbrake is a tool that is used to encode video files. The `HandBrakeCLI` command is used to interact with Handbrake from the command line. The `HandBrakeCLI` command must be in the system path for HandyMKV to work.

Handbrake can be downloaded from [here](https://handbrake.fr/). Note: Handbrake is free to use. See the Handbrake website for more information.

Documentation for Handbrake can be found [here](https://handbrake.fr/docs/en/latest/cli/cli-guide.html).

## Command Line Options

HandyMKV has a number of command line options that can be used to control its behavior. These options are described below.

```shell
Usage of handymkv:
  -c    Configure. Runs the configuration wizard.
  -d int
        Disc. The disc index to rip. If not provided then disc 0 will be ripped.
  -r    Read. Reads and outputs the first encountered configuration file. The current working directory is searched first, then the user-level configuration.
  -v    Version. Prints the version of the application.
```

## Installation

HandyMKV is a Go application and can be installed using the following command:

```shell
go install github.com/dmars8047/handymkv/cmd/handymkv@latest
```

Prebuilt binaries can be downloaded from the [releases](https://marshall-labs.com/handy/releases/latest) page.

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

## A Note on Concurrency

HandyMKV will attempt to execute tasks concurrently to reduce the overall time taken to complete the process. However, encoding tasks are resource intensive and running multiple encoding tasks is likely to slow down the overall process. Likewise ripping tasks are bottle-necked by the speed of the disc drive. For this reason HandyMKV will execute ripping and encoding pipelines concurrently but each task in those pipelines will be executed sequentially.
