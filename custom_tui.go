package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
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
	viewport   viewport.Model
}

type tickMsg time.Time
type pollResultMsg *RCAPollResponse
type pollErrorMsg error

var (
	// Spinner
	rcaSpinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Header
	rcaTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		MarginBottom(1)

	// Content: section labels and values
	rcaSectionHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		MarginTop(1)

	rcaLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	rcaValueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	rcaItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		PaddingLeft(2)

	rcaErrorStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Padding(1)

	rcaSuccessStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("46"))

	rcaMetaBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	// Content: evidence blocks
	rcaEvidenceBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginLeft(2).
		MarginBottom(1)

	rcaEvidenceQueryStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("117"))

	rcaEvidenceSnippetStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Italic(true)

	// Footer
	rcaFooterHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	rcaFooterStatusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	// Status badges
	rcaCompleteStatusStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("46"))

	rcaInProgressStatusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("226"))
)

func initialModel(config *Config, sessionID string) rcaModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = rcaSpinnerStyle
	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true

	m := rcaModel{
		config:     config,
		sessionID:  sessionID,
		spinner:    s,
		viewport:   vp,
		lastUpdate: time.Now(),
		maxRetries: 72,
		results:    &RCAPollResponse{SessionID: sessionID},
	}
	m.refreshViewportContent()
	return m
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
		viewportHeight := msg.Height - m.layoutChromeHeight(msg.Width)
		if viewportHeight < 1 {
			viewportHeight = 1
		}

		m.viewport.Width = msg.Width
		m.viewport.Height = viewportHeight
		m.refreshViewportContent()
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
			return m, nil
		case "j":
			m.viewport.LineDown(1)
			return m, nil
		case "k":
			m.viewport.LineUp(1)
			return m, nil
		case "up", "down", "pgup", "pgdown", "home", "end":
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}

		switch msg.Button {
		case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
			// Forward only wheel events so high-frequency mouse motion/drag
			// messages do not trigger unnecessary viewport updates.
		default:
			return m, nil
		}

		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case spinner.TickMsg:
		if m.isComplete || m.err != nil {
			return m, nil
		}

		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tickMsg:
		m.lastUpdate = time.Time(msg)
		m.refreshViewportContent()
		if !m.isComplete && m.err == nil {
			return m, tea.Batch(tickCmd(), pollRCACmd(m.config, m.sessionID))
		}
		return m, nil

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

		m.refreshViewportContent()
		return m, nil

	case pollErrorMsg:
		m.retryCount++
		if m.retryCount >= m.maxRetries {
			m.err = msg
			m.isComplete = true
		}
		m.refreshViewportContent()
		return m, nil
	}

	return m, nil
}

func (m *rcaModel) refreshViewportContent() {
	m.viewport.SetContent(m.buildContent())
}

func (m rcaModel) View() string {
	if m.quitting {
		return ""
	}
	return m.renderLayout(m.viewport.View())
}

func (m rcaModel) renderedHeight(content string, width int) int {
	if width <= 0 {
		return lipgloss.Height(content)
	}

	height := 0
	for _, line := range strings.Split(content, "\n") {
		lineWidth := lipgloss.Width(line)
		if lineWidth == 0 {
			height++
			continue
		}
		height += (lineWidth-1)/width + 1
	}

	return height
}

func (m rcaModel) layoutChromeHeight(width int) int {
	// Measure the rendered layout with empty content so size calculations stay
	// aligned with header/footer styles and any future layout tweaks. Count
	// visual line-wrapping at the current viewport width so narrow terminals
	// don't under-measure the header/footer chrome.
	return m.renderedHeight(m.renderLayout(""), width)
}

func (m rcaModel) renderLayout(content string) string {
	return lipgloss.JoinVertical(lipgloss.Left, m.buildHeader(), content, m.buildFooter())
}

func (m rcaModel) buildHeader() string {
	if m.err != nil {
		return rcaTitleStyle.Render("❌ RCA ANALYSIS ERROR")
	}

	if m.isComplete {
		return rcaTitleStyle.Render("✅ RCA ANALYSIS COMPLETED")
	}

	return rcaTitleStyle.Render(fmt.Sprintf("%s RCA ANALYSIS IN PROGRESS", m.spinner.View()))
}

func (m rcaModel) buildContent() string {
	var s strings.Builder

	if m.err != nil {
		s.WriteString(rcaErrorStyle.Render("❌ Error: " + m.err.Error()))
		return s.String()
	}

	metaContent := fmt.Sprintf("%s %s\n%s %s\n%s %d | %s %s",
		rcaLabelStyle.Render("Session ID:"),
		rcaValueStyle.Render(m.results.SessionID),
		rcaLabelStyle.Render("Status:"),
		m.getStatusView(),
		rcaLabelStyle.Render("Poll Count:"),
		m.pollCount,
		rcaLabelStyle.Render("Last Update:"),
		rcaValueStyle.Render(m.lastUpdate.Format("15:04:05")),
	)
	s.WriteString(rcaMetaBoxStyle.Render(metaContent))
	s.WriteString("\n")

	if m.results.ProblemShort != "" {
		s.WriteString(rcaSectionHeaderStyle.Render("📋 Problem"))
		s.WriteString("\n")
		s.WriteString(rcaItemStyle.Render(m.results.ProblemShort))
		s.WriteString("\n")
	}

	if m.results.Recommendation != "" {
		s.WriteString(rcaSectionHeaderStyle.Render("💡 Recommendation"))
		s.WriteString("\n")
		s.WriteString(rcaItemStyle.Render(m.results.Recommendation))
		s.WriteString("\n")
	}

	s.WriteString(rcaSectionHeaderStyle.Render("📝 What Happened"))
	s.WriteString("\n")
	if len(m.results.WhatHappened) > 0 {
		for i, event := range m.results.WhatHappened {
			s.WriteString(rcaItemStyle.Render(fmt.Sprintf("%d. %s", i+1, event)))
			s.WriteString("\n")
		}
	} else {
		s.WriteString(rcaItemStyle.Render(rcaLabelStyle.Render("⏳ Waiting for data...")))
		s.WriteString("\n")
	}

	s.WriteString(rcaSectionHeaderStyle.Render("🔍 Evidence"))
	s.WriteString("\n")
	if len(m.results.EvidenceCollection) > 0 {
		for i, evidence := range m.results.EvidenceCollection {
			evidenceContent := fmt.Sprintf("%s\n%s",
				rcaEvidenceQueryStyle.Render(fmt.Sprintf("%d. %s", i+1, evidence.Query)),
				rcaEvidenceSnippetStyle.Render("   → "+evidence.Snippet),
			)
			s.WriteString(rcaEvidenceBoxStyle.Render(evidenceContent))
			s.WriteString("\n")
		}
	} else {
		s.WriteString(rcaItemStyle.Render(rcaLabelStyle.Render("⏳ Waiting for data...")))
		s.WriteString("\n")
	}

	if !m.isComplete {
		s.WriteString(rcaSectionHeaderStyle.Render("📊 Operations"))
		s.WriteString("\n")
		if len(m.results.Operations) > 0 {
			for i, operation := range m.results.Operations {
				s.WriteString(rcaItemStyle.Render(fmt.Sprintf("%d. %s", i+1, operation)))
				s.WriteString("\n")
			}
		} else {
			s.WriteString(rcaItemStyle.Render(rcaLabelStyle.Render("⏳ Waiting for data...")))
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")
	if m.isComplete {
		s.WriteString(rcaSuccessStyle.Render("✓ Analysis Complete"))
	}

	return s.String()
}

func (m rcaModel) buildFooter() string {
	scrollPercent := int(m.viewport.ScrollPercent()*100 + 0.5)
	if scrollPercent < 0 {
		scrollPercent = 0
	}
	if scrollPercent > 100 {
		scrollPercent = 100
	}

	footerLeftText := fmt.Sprintf("↑/↓ to scroll • %d%%", scrollPercent)
	footerRightText := "Press Ctrl+C to stop monitoring"
	if m.err != nil || m.isComplete {
		footerRightText = "Press Enter or Ctrl+C to exit"
	}

	footerLeftText, footerRightText, separator := fitFooterToWidth(m.viewport.Width, footerLeftText, footerRightText)
	footerLeft := rcaFooterHintStyle.Render(footerLeftText)
	footerRight := rcaFooterStatusStyle.Render(footerRightText)

	return footerLeft + separator + footerRight
}

func fitFooterToWidth(width int, left string, right string) (string, string, string) {
	const defaultSeparator = "  "

	if width <= 0 {
		return left, right, defaultSeparator
	}

	left = strings.ReplaceAll(left, "\n", " ")
	right = strings.ReplaceAll(right, "\n", " ")

	separator := defaultSeparator
	totalWidth := lipgloss.Width(left) + lipgloss.Width(separator) + lipgloss.Width(right)
	if totalWidth <= width {
		return left, right, separator
	}

	availableLeft := width - lipgloss.Width(separator) - lipgloss.Width(right)
	if availableLeft >= 0 {
		return truncateToWidth(left, availableLeft), right, separator
	}

	separator = " "
	availableLeft = width - lipgloss.Width(separator) - lipgloss.Width(right)
	if availableLeft >= 0 {
		return truncateToWidth(left, availableLeft), right, separator
	}

	return "", truncateToWidth(right, width), ""
}

func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	if maxWidth == 1 {
		return "…"
	}

	const ellipsis = "…"
	limit := maxWidth - lipgloss.Width(ellipsis)
	if limit <= 0 {
		return ellipsis
	}

	var b strings.Builder
	currentWidth := 0
	for _, r := range s {
		rWidth := lipgloss.Width(string(r))
		if currentWidth+rWidth > limit {
			break
		}
		b.WriteRune(r)
		currentWidth += rWidth
	}

	return b.String() + ellipsis
}

func (m rcaModel) getStatusView() string {
	if m.isComplete {
		return rcaCompleteStatusStyle.Render("✅ Complete")
	}
	return rcaInProgressStatusStyle.Render("⏳ In Progress")
}

func (b *BubbleTeaTUI) MonitorRCA(config *Config, sessionID string) error {
	b.config = config
	b.sessionID = sessionID

	p := tea.NewProgram(
		initialModel(config, sessionID),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
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
	fmt.Println(rcaErrorStyle.Render(fmt.Sprintf("❌ %s: %v", message, err)))
}

func (b *BubbleTeaTUI) DisplayMessage(message string) {
	fmt.Println(message)
}

func (b *BubbleTeaTUI) DisplayProgressIndicator(message string) {
}

func (b *BubbleTeaTUI) WaitForExit() {
}
