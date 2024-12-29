package hnd

/*
Package hnd provides provides the bulk of the functionality for the Handy application.

# Overview

Its two main functions provide are to setup the configuration files and to execute the ripping and encoding of the files from a specified disc.

Setup() - This function is called by the main package to setup the handy configuration files. Configuration files store settings for the ripping and encoding process and various other runtime settings.

Exec() - This function is called by the main package to execute the Ripping and Encoding of the files from a specified disc. Ripping and Encoding are two separate processes that are executed with concurrency (where possible).
*/
