# mosquitto commandline client

## Purpose

A commandline go executable that lets a user monitor and interact with a mosquitto server with an intuitive interface.

## Architecture

- Go program using Bubble Tea for the TUI
- Displaying topics with their latest message payload and timestamp
- Configured connection to broker via command-line flags
- Break down your code in small methods with maximum 200 lines
- Use as little state as possible
