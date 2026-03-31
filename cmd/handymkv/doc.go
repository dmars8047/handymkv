package main

/*
Package main is the entry point for the HandyMKV application.

# Overview

The main package is responsible for parsing command line arguments and prerequisite checking (makemkvcon and HandBrakeCLI binaries), then delegating to Setup() or Exec() in the hmkv package.

## Command Line Flags

  -v         Print the application version and exit.
  -d string  Comma-delimited list of disc indexes to rip. (default "0")
  -a string  Comma-delimited list of automation names to run after encoding.

## Subcommands

  config              Display the current configuration file.
  config setup        Run the configuration wizard to create/update config.json.
  config edit         Open the config file in the default editor (EDITOR or VISUAL env var).
  discs               List available discs.
  history             Show a summary list of past runs.
  history <number>    Show details for a specific past run by manifest index.
  history clear       Delete all manifest files from the run history directory.
  automations         List all saved automations.
  automations create  Create a new automation.
  automations show    Show details of a specific automation.
  automations delete  Delete an automation.

## Prerequisite Checking

The checkForPrerequisites function verifies that makemkvcon (MakeMKV) and HandBrakeCLI (HandBrake) binaries are available. If either is missing, an error is returned and the application exits without starting the ripping or encoding pipeline.
*/
