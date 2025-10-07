package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BubbleTeaTUI struct {
	config    *Config
	sessionID string
}

func NewBubbleTeaTUI() *BubbleTeaTUI {
	return &BubbleTeaTUI{}
}

type rcaModel struct {
	config     *Config
	sessionID  string
	spinner    spinner.Model
	results    *RCAPollResponse
	pollCount  int
	err        error
	isComplete bool
	quitting   bool
	lastUpdate time.Time
	retryCount int
	maxRetries int
	width      int
	height     int
}

type tickMsg time.Time
type pollResultMsg *RCAPollResponse
type pollErrorMsg error

func initialModel(config *Config, sessionID string) rcaModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return rcaModel{
		config:     config,
		sessionID:  sessionID,
		spinner:    s,
		lastUpdate: time.Now(),
		maxRetries: 72,
		results:    &RCAPollResponse{SessionID: sessionID},
	}
}

func (m rcaModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tickCmd(),
		pollRCACmd(m.config, m.sessionID),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func pollRCACmd(config *Config, sessionID string) tea.Cmd {
	return func() tea.Msg {
		result, err := fetchRCAStatus(config, sessionID)
		if err != nil {
			return pollErrorMsg(err)
		}
		return pollResultMsg(result)
	}
}

func (m rcaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if m.isComplete || m.err != nil {
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tickMsg:
		m.lastUpdate = time.Time(msg)
		if !m.isComplete && m.err == nil {
			return m, tea.Batch(tickCmd(), pollRCACmd(m.config, m.sessionID))
		}
		return m, tickCmd()

	case pollResultMsg:
		m.results = msg
		m.pollCount++
		m.retryCount = 0
		m.isComplete = msg.IsComplete

		if msg.IsComplete && msg.RawData != nil {
			logRawRCAData("Final RCA Response", msg.RawData)
		}

		if m.pollCount > 300 {
			m.err = fmt.Errorf("timeout reached (15 minutes)")
			m.isComplete = true
		}

		return m, nil

	case pollErrorMsg:
		m.retryCount++
		if m.retryCount >= m.maxRetries {
			m.err = msg
			m.isComplete = true
		}
		return m, nil
	}

	return m, nil
}

func (m rcaModel) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		MarginBottom(1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		MarginTop(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		PaddingLeft(2)

	errorStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Padding(1)

	successStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("46"))

	if m.err != nil {
		s.WriteString(errorStyle.Render("❌ Error: " + m.err.Error()))
		s.WriteString("\n\n")
		s.WriteString(labelStyle.Render("Press Enter or Ctrl+C to exit"))
		return s.String()
	}

	if m.isComplete {
		s.WriteString(titleStyle.Render("✅ RCA ANALYSIS COMPLETED"))
	} else {
		s.WriteString(titleStyle.Render(fmt.Sprintf("%s RCA ANALYSIS IN PROGRESS", m.spinner.View())))
	}
	s.WriteString("\n\n")

	metaBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	metaContent := fmt.Sprintf("%s %s\n%s %s\n%s %d | %s %s",
		labelStyle.Render("Session ID:"),
		valueStyle.Render(m.results.SessionID),
		labelStyle.Render("Status:"),
		m.getStatusView(),
		labelStyle.Render("Poll Count:"),
		m.pollCount,
		labelStyle.Render("Last Update:"),
		valueStyle.Render(m.lastUpdate.Format("15:04:05")),
	)
	s.WriteString(metaBox.Render(metaContent))
	s.WriteString("\n")

	if m.results.ProblemShort != "" {
		s.WriteString(headerStyle.Render("📋 Problem"))
		s.WriteString("\n")
		s.WriteString(itemStyle.Render(m.results.ProblemShort))
		s.WriteString("\n")
	}

	if m.results.Recommendation != "" {
		s.WriteString(headerStyle.Render("💡 Recommendation"))
		s.WriteString("\n")
		s.WriteString(itemStyle.Render(m.results.Recommendation))
		s.WriteString("\n")
	}

	s.WriteString(headerStyle.Render("📝 What Happened"))
	s.WriteString("\n")
	if len(m.results.WhatHappened) > 0 {
		for i, event := range m.results.WhatHappened {
			s.WriteString(itemStyle.Render(fmt.Sprintf("%d. %s", i+1, event)))
			s.WriteString("\n")
		}
	} else {
		s.WriteString(itemStyle.Render(labelStyle.Render("⏳ Waiting for data...")))
		s.WriteString("\n")
	}

	s.WriteString(headerStyle.Render("🔍 Evidence"))
	s.WriteString("\n")
	if len(m.results.EvidenceCollection) > 0 {
		evidenceBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			MarginLeft(2).
			MarginBottom(1)

		evidenceQueryStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("117"))

		evidenceSnippetStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Italic(true)

		for i, evidence := range m.results.EvidenceCollection {
			evidenceContent := fmt.Sprintf("%s\n%s",
				evidenceQueryStyle.Render(fmt.Sprintf("%d. %s", i+1, evidence.Query)),
				evidenceSnippetStyle.Render("   → "+evidence.Snippet),
			)
			s.WriteString(evidenceBoxStyle.Render(evidenceContent))
			s.WriteString("\n")
		}
	} else {
		s.WriteString(itemStyle.Render(labelStyle.Render("⏳ Waiting for data...")))
		s.WriteString("\n")
	}

	if !m.isComplete {
		s.WriteString(headerStyle.Render("📊 Operations"))
		s.WriteString("\n")
		if len(m.results.Operations) > 0 {
			for i, operation := range m.results.Operations {
				s.WriteString(itemStyle.Render(fmt.Sprintf("%d. %s", i+1, operation)))
				s.WriteString("\n")
			}
		} else {
			s.WriteString(itemStyle.Render(labelStyle.Render("⏳ Waiting for data...")))
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")
	if m.isComplete {
		s.WriteString(successStyle.Render("✓ Analysis Complete"))
		s.WriteString("\n")
		s.WriteString(labelStyle.Render("Press Enter or Ctrl+C to exit"))
	} else {
		s.WriteString(labelStyle.Render("Press Ctrl+C to stop monitoring"))
	}

	return s.String()
}

func (m rcaModel) getStatusView() string {
	if m.isComplete {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46")).
			Render("✅ Complete")
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Render("⏳ In Progress")
}

func (b *BubbleTeaTUI) MonitorRCA(config *Config, sessionID string) error {
	b.config = config
	b.sessionID = sessionID

	p := tea.NewProgram(
		initialModel(config, sessionID),
		tea.WithAltScreen(),
	)

	model, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	finalModel := model.(rcaModel)
	if finalModel.err != nil {
		return finalModel.err
	}

	return nil
}

func (b *BubbleTeaTUI) ClearScreen() {
}

func (b *BubbleTeaTUI) DisplayLiveRCAResults(results *RCAPollResponse, pollCount int) {
}

func (b *BubbleTeaTUI) DisplayFinalRCAResults(results *RCAPollResponse) {
}

func (b *BubbleTeaTUI) DisplayError(message string, err error) {
	errorStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Padding(1)

	fmt.Println(errorStyle.Render(fmt.Sprintf("❌ %s: %v", message, err)))
}

func (b *BubbleTeaTUI) DisplayMessage(message string) {
	fmt.Println(message)
}

func (b *BubbleTeaTUI) DisplayProgressIndicator(message string) {
}

func (b *BubbleTeaTUI) WaitForExit() {
}
