# mosquitto commandline client

## Purpose

A commandline go executable that lets a user monitor and interact with a mosquitto server intuitively.

## Architecture

Go program using Bubble Tea for the TUI, displaying topics with their latest message payload and timestamp, configured via command-line flags for the initial server connection.