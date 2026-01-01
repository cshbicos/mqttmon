# MQTT Monitor

A vibe coded, fullscreen terminal UI for monitoring all topics on a Mosquitto MQTT server in real-time.

Mainly a playing ground for some simple AI experiments with a semi-functional outcome.

## Features

- Fullscreen terminal interface using Bubble Tea
- Real-time monitoring of all MQTT topics (subscribes to `#` wildcard)
- Navigate topics with arrow keys and visual selection highlighting
- Quick publish to existing topics: select with arrow keys and press Enter
- Publish to new topics with an interactive modal dialog (press `p`)
- Displays latest message payload and timestamp for each topic
- Sorted topic list with automatic scrolling
- Clean, colorful interface with syntax highlighting
- Status messages for successful publishes

## Installation

```bash
go build -o mqttmon
```

## Usage

Basic usage with local broker:
```bash
./mqttmon
```

Connect to a specific broker:
```bash
./mqttmon --broker tcp://mqtt.example.com:1883
```

With authentication:
```bash
./mqttmon --broker tcp://mqtt.example.com:1883 --username user --password pass
```

Custom client ID:
```bash
./mqttmon --client-id my-monitor
```

### Command-line Flags

- `--broker` - MQTT broker URL (default: `tcp://localhost:1883`)
- `--username` - MQTT username (optional)
- `--password` - MQTT password (optional)
- `--client-id` - MQTT client ID (default: `mqttmon`)

### Controls

**Main View:**
- `↑` / `↓` (Arrow keys) - Navigate and select topics
- `Enter` - Publish a new message to the selected topic
- `p` - Open publish dialog for a new topic
- `q` or `Ctrl+C` - Quit the application

**Publish to Existing Topic (after pressing Enter on selection):**
- Type to enter the message payload
- `Enter` - Publish the message
- `Esc` - Cancel and return to main view
- `Backspace` - Delete characters

**Publish to New Topic (after pressing 'p'):**
- `Tab` - Switch between topic and message fields
- `Enter` - Submit and publish message (or move from topic to message field)
- `Esc` - Cancel and close dialog
- `Backspace` - Delete characters
- Type to enter text in the focused field

## How It Works

The monitor:
1. Connects to your MQTT broker
2. Subscribes to all topics using the `#` wildcard
3. Displays topics in a sorted list with their latest message and timestamp
4. Updates in real-time as new messages arrive
5. Navigate topics with arrow keys - the selected topic is highlighted with a `>` indicator
6. Press Enter on a selected topic to quickly publish a new message to it
7. Press `p` to publish to any topic (new or existing) via an interactive dialog
8. Shows status notifications for successful publishes
9. Automatically handles screen resizing and scrolling for many topics

## Requirements

- Go 1.21 or later
- Access to a Mosquitto (or any MQTT 3.1.1 compatible) broker

## Dependencies

- [paho.mqtt.golang](https://github.com/eclipse/paho.mqtt.golang) - MQTT client library
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions for terminal rendering
