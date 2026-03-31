package hmkv

/*
Package hmkv provides the core orchestration logic for HandyMKV.

# Overview

This package coordinates MakeMKV (disc ripping) and HandBrake (video encoding) as subprocesses, with concurrent MKV ripping and encoding pipelines connected by a channel.

# Main Entry Points

Setup() - Interactive configuration wizard that writes to ~/.config/handymkv/config.json (or platform-specific equivalents).

Exec() - Main execution pipeline: reads disc titles, launches concurrent goroutines for MKV ripping and HB encoding, tracks progress, and optionally cleans up source files.

# Execution Pipeline

Exec() follows this flow:

 1. Reads disc titles from MakeMKV via mkv.ListTitles()
 2. User interactively selects which titles to rip
 3. Creates timestamped output directories
 4. Launches two concurrent goroutines connected by a channel:
    - MKV worker: calls mkv.RipTitle() serially for each selected title, sends completed paths to the encoding channel
    - HB worker: reads MKV paths from channel, encodes with hb.EncodeFile()
 5. Central progress tracker (progress.go) polls file sizes and refreshes terminal output every 200ms
 6. Waits for both workers to complete, then summarizes space saved and time elapsed
 7. Optionally deletes raw MKV files per user config

# Key Files

  conf.go     Config file read/write, OS-aware paths
  mkv.go      Shell out to makemkvcon, parse disc/title output
  hb.go       Shell out to HandBrakeCLI, parse encoding progress
  progress.go Concurrent progress state, file-size polling
  term.go     ANSI colors, logos, terminal utilities
  constants.go Custom error types, encoder constants
*/
