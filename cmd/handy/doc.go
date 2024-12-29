package main

/*
Package main is the entry point for the Handy application.

# Overview

The main package is responsible for parsing command line arguments and executing the application.

The main function is the entry point for the application. It parses command line arguments and then executes the application.

## Prequisite Checking

The checkForPrerequisites function is called to ensure that the application has all the prerequisites applications installed it needs to run.

The following applications are checked for:

- makemkvcon
- HandBrakeCLI

If any of these applications are not found in the $PATH then an error is returned and the application exits. Execution of ripping and encoding will not start without these applications.

## Command Line Arguments

If the -c flag is provided then the application will run the setup process. This process will create the configuration files needed for the application to run.

If the -d flag is provided then the application will rip the disc with the specified index. If no index is provided then the application will rip disc 0.

If the -q flag is provided then the application will rip the disc with the specified quality. If no quality is provided then the application will rip with the quality specificed in the config file.

If the -e flag is provided then the application will rip the disc with the specified encoder. If no encoder is provided then the application will rip with the encoder specificed in the config file.

*/
