# Handy

## Description

Handy is a tool that is designed to automate the process of ripping discs using MakeMKV and then encoding the resulting files using Handbrake.

## Features

- Rip titles from discs using MakeMKV
- Encode video files using HandBrake
- Flexible configuration options
- Clear and concise progress display
- Concurrency to reduce overall time
- Summary of space saved and time elapsed
- Automated cleanup (after prompt) of raw unencoded files

## Objectives

### Time Saving

Its primary aim is to save time and effort by removing the disconnected nature of performing these tasks manually and/or sequentially. Concurrency is used to reduce the overall time taken to complete the process.

### Ease of Use

Handy is designed to be easy to setup and use. Once a configuration has been created the process of ripping and encoding a disc is simplified to a single command and a few prompts.

Output from the process is displayed in a clear and concise manner to keep the user informed of the progress of the tasks.

### Flexibility

Handy is designed to be flexible. The user can select which titles to rip from the disc and can configure the encoding options to suit their needs.

## Prerequisites

### MakeMKV

MakeMKV is a tool that is used to rip the contents of a disc to a file on the local system. Note: this tool is not free and a license must be purchased for use.

Specifically the `makemkvcon` command is used to interact with MakeMKV from the command line. The `makemkvcon` command must be in the system path for Handy to work.

MakeMKV can be downloaded from [here](https://www.makemkv.com/).

Documentation for MakeMKV can be found [here](https://www.makemkv.com/developers/usage.txt).

### Handbrake

Handbrake is a tool that is used to encode video files. The `HandBrakeCLI` command is used to interact with Handbrake from the command line. The `HandBrakeCLI` command must be in the system path for Handy to work.

Handbrake can be downloaded from [here](https://handbrake.fr/). Note: Handbrake is free to use. See the Handbrake website for more information.

Documentation for Handbrake can be found [here](https://handbrake.fr/docs/en/latest/cli/cli-guide.html).

## Command Line Options

Handy has a number of command line options that can be used to control its behaviour. These options are described below.

```shell
  -c    Config. Runs the configuration wizard.
  -d int
        Disc. The disc index to rip. If not provided then disc 0 will be ripped.
  -e string
        Encoder.
                If not provided then the value will be read from the 'config.json' file.
                If the config file cannot be found then then a default value of 'h264' will be used.
  -q int
        Quality.
                Sets the quality value to be used for each encoding task. If not provided then the value will be read from the 'config.json' file.
                If the config file cannot be found then then a default value of '20' will be used. (default -1)
```

## Installation

Handy is a Go application and can be installed using the following command:

```shell
go install github.com/dmars8047/handy/cmd/handy@latest
```

Pre-built binaries can be downloaded from the [here](https://marshall-labs.com/handy/releases/latest) page.

## A Note on Concurrency

Handy will attempt to execute tasks concurrently to reduce the overall time taken to complete the process. Opportunities for concurrency are limited by the dependencies between tasks, for example, encoding cannot begin until ripping is complete. Encoding tasks also tend to be resource intensive so the number of concurrent encoding tasks is limited to avoid overloading the system. When a ripping task is complete, it will be handed off the next available encoding task. This process will continue until all indicated titles have been ripped and encoded.