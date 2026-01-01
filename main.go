package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	broker   = flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	username = flag.String("username", "", "MQTT username")
	password = flag.String("password", "", "MQTT password")
	clientID = flag.String("client-id", "mqttmon", "MQTT client ID")
)

type topicMessage struct {
	payload   string
	timestamp time.Time
	qos       byte
}

type model struct {
	topics         map[string]*topicMessage
	topicsMutex    sync.RWMutex
	client         mqtt.Client
	err            error
	width          int
	height         int
	publishMode    bool
	editingTopic   bool
	publishTopic   string
	publishMessage string
	focusedField   int // 0 = topic, 1 = message
	selectedIndex  int
	statusMessage  string
	statusTime     time.Time
}

type mqttMsg struct {
	topic   string
	payload string
	qos     byte
}

func initialModel(client mqtt.Client) model {
	return model{
		topics: make(map[string]*topicMessage),
		client: client,
	}
}

func (m model) Init() tea.Cmd {
	return waitForMQTTMessage()
}

func waitForMQTTMessage() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		return nil
	}
}

func (m model) executePublish() model {
	if m.publishTopic == "" {
		m.statusMessage = "Error: Topic cannot be empty"
		m.statusTime = time.Now()
		return m
	}

	token := m.client.Publish(m.publishTopic, 0, false, m.publishMessage)
	go func() {
		token.Wait()
	}()

	m.statusMessage = fmt.Sprintf("Published to: %s", m.publishTopic)
	m.statusTime = time.Now()
	m.publishMode = false
	m.editingTopic = false
	m.publishTopic = ""
	m.publishMessage = ""
	m.focusedField = 0

	return m
}

func (m model) handlePublishModeKeys(key string) (model, tea.Cmd) {
	switch key {
	case "esc":
		m.publishMode = false
		m.editingTopic = false
		m.publishTopic = ""
		m.publishMessage = ""
		m.focusedField = 0
		return m, nil

	case "tab":
		if m.publishMode {
			m.focusedField = (m.focusedField + 1) % 2
		}
		return m, nil

	case "enter":
		if m.publishMode && m.focusedField == 0 && m.publishTopic != "" {
			m.focusedField = 1
			return m, nil
		} else if (m.publishMode && m.focusedField == 1) || m.editingTopic {
			m = m.executePublish()
			return m, nil
		}

	case "backspace":
		if m.publishMode && m.focusedField == 0 && len(m.publishTopic) > 0 {
			m.publishTopic = m.publishTopic[:len(m.publishTopic)-1]
		} else if (m.publishMode && m.focusedField == 1) || m.editingTopic {
			if len(m.publishMessage) > 0 {
				m.publishMessage = m.publishMessage[:len(m.publishMessage)-1]
			}
		}
		return m, nil

	default:
		if len(key) == 1 {
			if m.publishMode && m.focusedField == 0 {
				m.publishTopic += key
			} else if (m.publishMode && m.focusedField == 1) || m.editingTopic {
				m.publishMessage += key
			}
		}
		return m, nil
	}

	return m, nil
}

func (m model) handleMainViewKeys(key string) (model, tea.Cmd) {
	switch key {
	case "ctrl+c", "q":
		if m.client != nil && m.client.IsConnected() {
			m.client.Disconnect(250)
		}
		return m, tea.Quit

	case "p":
		m.publishMode = true
		m.focusedField = 0
		m.publishTopic = ""
		m.publishMessage = ""
		return m, nil

	case "up":
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
		return m, nil

	case "down":
		m.topicsMutex.RLock()
		maxIndex := len(m.topics) - 1
		m.topicsMutex.RUnlock()
		if m.selectedIndex < maxIndex {
			m.selectedIndex++
		}
		return m, nil

	case "enter":
		m.topicsMutex.RLock()
		if len(m.topics) > 0 {
			sortedTopics := make([]string, 0, len(m.topics))
			for topic := range m.topics {
				sortedTopics = append(sortedTopics, topic)
			}
			sort.Strings(sortedTopics)

			if m.selectedIndex < len(sortedTopics) {
				selectedTopic := sortedTopics[m.selectedIndex]
				m.topicsMutex.RUnlock()
				m.editingTopic = true
				m.publishTopic = selectedTopic
				m.publishMessage = ""
				return m, nil
			}
		}
		m.topicsMutex.RUnlock()
		return m, nil
	}

	return m, nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.publishMode || m.editingTopic {
			return m.handlePublishModeKeys(msg.String())
		}
		return m.handleMainViewKeys(msg.String())

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case mqttMsg:
		m.topicsMutex.Lock()
		m.topics[msg.topic] = &topicMessage{
			payload:   msg.payload,
			timestamp: time.Now(),
			qos:       msg.qos,
		}
		m.topicsMutex.Unlock()
		return m, waitForMQTTMessage()

	case error:
		m.err = msg
		return m, tea.Quit
	}

	return m, nil
}

type styles struct {
	header          lipgloss.Style
	topic           lipgloss.Style
	selectedTopic   lipgloss.Style
	timestamp       lipgloss.Style
	payload         lipgloss.Style
	selectedPayload lipgloss.Style
	help            lipgloss.Style
	status          lipgloss.Style
	label           lipgloss.Style
	input           lipgloss.Style
	focusedInput    lipgloss.Style
	readOnly        lipgloss.Style
	publishBox      lipgloss.Style
}

func getStyles() styles {
	return styles{
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			Background(lipgloss.Color("236")).
			Padding(0, 1),
		topic: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10")),
		selectedTopic: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("11")).
			Background(lipgloss.Color("236")),
		timestamp: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")),
		payload: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")),
		selectedPayload: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("236")),
		help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true),
		status: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")),
		label: lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Bold(true),
		input: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")),
		focusedInput: lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Background(lipgloss.Color("236")),
		readOnly: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")),
		publishBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")).
			Padding(1, 2).
			Width(60),
	}
}

func (m model) renderTopicsList(s styles) string {
	var b strings.Builder

	m.topicsMutex.RLock()
	defer m.topicsMutex.RUnlock()

	if len(m.topics) == 0 {
		b.WriteString("Waiting for messages...\n")
		return b.String()
	}

	sortedTopics := make([]string, 0, len(m.topics))
	for topic := range m.topics {
		sortedTopics = append(sortedTopics, topic)
	}
	sort.Strings(sortedTopics)

	maxLines := m.height - 5
	if maxLines < 1 {
		maxLines = 10
	}

	displayCount := 0
	for i, topic := range sortedTopics {
		if displayCount >= maxLines {
			remaining := len(m.topics) - displayCount
			b.WriteString(fmt.Sprintf("\n... and %d more topics", remaining))
			break
		}

		msg := m.topics[topic]
		timeStr := msg.timestamp.Format("15:04:05")
		isSelected := i == m.selectedIndex

		if isSelected {
			topicLine := s.selectedTopic.Render("> " + topic)
			timestampLine := s.timestamp.Render(fmt.Sprintf("[%s]", timeStr))
			b.WriteString(fmt.Sprintf("%s %s\n", topicLine, timestampLine))
		} else {
			topicLine := s.topic.Render(topic)
			timestampLine := s.timestamp.Render(fmt.Sprintf("[%s]", timeStr))
			b.WriteString(fmt.Sprintf("  %s %s\n", topicLine, timestampLine))
		}

		payload := msg.payload
		maxPayloadLen := m.width - 4
		if maxPayloadLen < 20 {
			maxPayloadLen = 20
		}
		if len(payload) > maxPayloadLen {
			payload = payload[:maxPayloadLen-3] + "..."
		}

		if isSelected {
			b.WriteString(s.selectedPayload.Render(fmt.Sprintf("  %s\n", payload)))
		} else {
			b.WriteString(s.payload.Render(fmt.Sprintf("  %s\n", payload)))
		}

		b.WriteString("\n")
		displayCount++
	}

	return b.String()
}

func (m model) renderStatusAndHelp(s styles, topicCount int) string {
	var b strings.Builder

	if m.statusMessage != "" && time.Since(m.statusTime) < 3*time.Second {
		b.WriteString(s.status.Render(m.statusMessage))
		b.WriteString(" | ")
	}

	help := s.help.Render(fmt.Sprintf("Topics: %d | ↑/↓: select | Enter: edit | p: publish new | q: quit", topicCount))
	b.WriteString(help)

	return b.String()
}

func (m model) renderPublishDialog(s styles, mainView string) string {
	var publishContent strings.Builder

	if m.editingTopic {
		publishContent.WriteString(s.label.Render("Publish to Topic") + "\n\n")
	} else {
		publishContent.WriteString(s.label.Render("Publish Message") + "\n\n")
	}

	topicLabel := "Topic: "
	topicInput := m.publishTopic

	if m.editingTopic {
		publishContent.WriteString(s.label.Render(topicLabel))
		publishContent.WriteString(s.readOnly.Render(topicInput + " (read-only)"))
	} else {
		if m.focusedField == 0 {
			publishContent.WriteString(s.label.Render(topicLabel))
			publishContent.WriteString(s.focusedInput.Render(topicInput + "▌"))
		} else {
			publishContent.WriteString(s.label.Render(topicLabel))
			publishContent.WriteString(s.input.Render(topicInput))
		}
	}

	publishContent.WriteString("\n\n")

	messageLabel := "Message: "
	messageInput := m.publishMessage

	if m.editingTopic || m.focusedField == 1 {
		publishContent.WriteString(s.label.Render(messageLabel))
		publishContent.WriteString(s.focusedInput.Render(messageInput + "▌"))
	} else {
		publishContent.WriteString(s.label.Render(messageLabel))
		publishContent.WriteString(s.input.Render(messageInput))
	}

	publishContent.WriteString("\n\n")
	if m.editingTopic {
		publishContent.WriteString(s.help.Render("Enter: publish | Esc: cancel"))
	} else {
		publishContent.WriteString(s.help.Render("Tab: switch field | Enter: publish | Esc: cancel"))
	}

	publishBox := s.publishBox.Render(publishContent.String())

	lines := strings.Split(mainView, "\n")
	dialogHeight := 12
	insertLine := (m.height - dialogHeight) / 2
	if insertLine < 0 {
		insertLine = 0
	}
	if insertLine >= len(lines) {
		insertLine = len(lines) - 1
	}

	var result strings.Builder
	for i := 0; i < len(lines); i++ {
		if i == insertLine {
			result.WriteString(publishBox + "\n")
		}
		if i < len(lines) {
			result.WriteString(lines[i] + "\n")
		}
	}

	return result.String()
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	s := getStyles()
	var b strings.Builder

	header := s.header.Render(fmt.Sprintf(" MQTT Monitor - %s ", *broker))
	b.WriteString(header)
	b.WriteString("\n\n")

	b.WriteString(m.renderTopicsList(s))

	b.WriteString("\n")

	m.topicsMutex.RLock()
	topicCount := len(m.topics)
	m.topicsMutex.RUnlock()

	b.WriteString(m.renderStatusAndHelp(s, topicCount))

	mainView := b.String()

	if m.publishMode || m.editingTopic {
		return m.renderPublishDialog(s, mainView)
	}

	return mainView
}

func main() {
	flag.Parse()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(*broker)
	opts.SetClientID(*clientID)

	if *username != "" {
		opts.SetUsername(*username)
	}
	if *password != "" {
		opts.SetPassword(*password)
	}

	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("Unexpected message on topic: %s\n", msg.Topic())
	})

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Println("Connected to MQTT broker")
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("Connection lost: %v\n", err)
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	p := tea.NewProgram(initialModel(client), tea.WithAltScreen())

	var program *tea.Program = p

	token := client.Subscribe("#", 0, func(client mqtt.Client, msg mqtt.Message) {
		if program != nil {
			program.Send(mqttMsg{
				topic:   msg.Topic(),
				payload: string(msg.Payload()),
				qos:     msg.Qos(),
			})
		}
	})

	if token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to subscribe: %v", token.Error())
	}

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
