package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/pentora-ai/pentora/pkg/scanexec"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type progressEventMsg scanexec.ProgressEvent

type progressCompleteMsg struct {
	status string
	errMsg string
}

type progressLogMsg struct {
	line  string
	level string
}

type progressUISink struct {
	program    *tea.Program
	done       chan struct{}
	origLogger zerolog.Logger
	origCtx    *zerolog.Logger
	final      chan string
	logPipe    *progressLogWriter
}

func newProgressUISink() (*progressUISink, error) {
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return nil, fmt.Errorf("progress UI requires an interactive terminal")
	}

	ready := make(chan struct{})
	model := newProgressModel(ready)

	program := tea.NewProgram(model, tea.WithOutput(os.Stdout), tea.WithoutSignalHandler(), tea.WithAltScreen())

	sink := &progressUISink{
		program: program,
		done:    make(chan struct{}),
		final:   make(chan string, 1),
	}

	origLogger := log.Logger
	origCtx := zerolog.DefaultContextLogger
	pipe := &progressLogWriter{sink: sink}
	consoleWriter := zerolog.ConsoleWriter{
		Out:        pipe,
		TimeFormat: "15:04:05",
		NoColor:    true,
	}
	log.Logger = origLogger.Output(consoleWriter)
	zerolog.DefaultContextLogger = &log.Logger
	sink.origLogger = origLogger
	sink.origCtx = origCtx
	sink.logPipe = pipe

	go func() {
		defer close(sink.done)
		res, _ := program.Run()
		if m, ok := res.(*progressModel); ok {
			select {
			case sink.final <- m.View():
			default:
			}
		}
		close(sink.final)
	}()

	<-ready
	return sink, nil
}

func (p *progressUISink) OnEvent(ev scanexec.ProgressEvent) {
	if p == nil || p.program == nil {
		return
	}
	p.program.Send(progressEventMsg(ev))
}

func (p *progressUISink) Stop(status string, runErr error) {
	if p == nil || p.program == nil {
		return
	}
	msg := progressCompleteMsg{status: status}
	if runErr != nil {
		msg.errMsg = runErr.Error()
	}
	p.program.Send(msg)
	<-p.done
	if p.logPipe != nil {
		p.logPipe.flush()
	}
	log.Logger = p.origLogger
	zerolog.DefaultContextLogger = p.origCtx
	if view, ok := <-p.final; ok {
		if strings.TrimSpace(view) != "" {
			fmt.Println(view)
		}
	}
}

type moduleStatus struct {
	Phase   string
	Module  string
	Status  string
	Message string
	Updated time.Time
}

type logEntry struct {
	text   string
	status string
}

type progressModel struct {
	ready       chan struct{}
	statuses    map[string]moduleStatus
	finished    bool
	finalStatus string
	errMessage  string
	started     time.Time
	spinnerIdx  int
	logs        []logEntry
	maxLogs     int
	lastEvent   moduleStatus
	events      int
}

func newProgressModel(ready chan struct{}) *progressModel {
	return &progressModel{
		ready:    ready,
		statuses: make(map[string]moduleStatus),
		started:  time.Now(),
		logs:     make([]logEntry, 0, 32),
		maxLogs:  500,
	}
}

func (m *progressModel) Init() tea.Cmd {
	return tea.Batch(func() tea.Msg {
		close(m.ready)
		return nil
	}, tickCmd())
}

func (m *progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch t := msg.(type) {
	case progressEventMsg:
		ev := scanexec.ProgressEvent(t)
		key := m.keyForEvent(ev)
		status := moduleStatus{
			Phase:   ev.Phase,
			Module:  formatModule(ev),
			Status:  strings.ToLower(ev.Status),
			Message: ev.Message,
			Updated: time.Now(),
		}
		m.statuses[key] = status
		m.events++
		m.lastEvent = status

		timestamp := ev.Timestamp
		if timestamp.IsZero() {
			timestamp = time.Now()
		}
		message := ev.Message
		if message == "" {
			message = "(no message)"
		}
		logLine := fmt.Sprintf("%s  %-7s %-26s %-9s %s",
			timestamp.Format("15:04:05"),
			strings.ToUpper(ev.Phase),
			truncate(formatModule(ev), 26),
			strings.ToUpper(ev.Status),
			message,
		)
		m.appendLog(logLine, status.Status)
		return m, nil
	case progressCompleteMsg:
		m.finished = true
		m.finalStatus = strings.ToLower(t.status)
		m.errMessage = t.errMsg
		return m, tea.Quit
	case progressLogMsg:
		m.appendLog(t.line, t.level)
		return m, nil
	case tickMsg:
		if m.finished {
			return m, nil
		}
		m.spinnerIdx = (m.spinnerIdx + 1) % len(spinnerFrames)
		return m, tickCmd()
	case tea.KeyMsg:
		if t.Type == tea.KeyCtrlC {
			m.finished = true
			m.finalStatus = "cancelled"
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *progressModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Pentora Scan Progress"))
	b.WriteString("\n")

	if len(m.logs) == 0 {
		elapsed := time.Since(m.started).Round(time.Second)
		placeholder := fmt.Sprintf("%s waiting for first module event… (%s elapsed)", spinnerFrame(m.spinnerIdx), elapsed)
		b.WriteString(subtleStyle.Render(placeholder))
		b.WriteString("\n")
	} else {
		for _, entry := range m.logs {
			b.WriteString(styleForStatus(entry.status).Render(entry.text))
			b.WriteString("\n")
		}
	}

	b.WriteString(dividerStyle.Render(strings.Repeat("─", 72)))
	b.WriteString("\n")

	elapsed := time.Since(m.started).Round(time.Second)
	if m.finished {
		summary := fmt.Sprintf("Scan %s", finishPhrase(m.finalStatus))
		if m.errMessage != "" {
			summary += fmt.Sprintf(" — %s", m.errMessage)
		}
		summary += fmt.Sprintf(" | elapsed %s | events %d", elapsed, m.events)
		b.WriteString(statusBarStyle.Render(summary))
	} else {
		lastModule := "waiting"
		lastPhase := ""
		lastStatus := ""
		lastMessage := ""
		if m.lastEvent.Module != "" {
			lastModule = m.lastEvent.Module
			lastPhase = m.lastEvent.Phase
			lastStatus = strings.ToUpper(m.lastEvent.Status)
			lastMessage = m.lastEvent.Message
		}
		info := fmt.Sprintf("%s phase=%s module=%s status=%s events=%d elapsed=%s",
			spinnerFrame(m.spinnerIdx),
			lastPhase,
			truncate(lastModule, 24),
			lastStatus,
			m.events,
			elapsed,
		)
		if lastMessage != "" {
			info += " | " + lastMessage
		}
		b.WriteString(statusBarStyle.Render(info))
	}

	return b.String()
}

func (m *progressModel) appendLog(text, status string) {
	entry := logEntry{text: text, status: status}
	if len(m.logs) >= m.maxLogs {
		m.logs = append(m.logs[1:], entry)
	} else {
		m.logs = append(m.logs, entry)
	}
}

type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

type progressLogWriter struct {
	sink *progressUISink
	buf  strings.Builder
}

func (w *progressLogWriter) Write(p []byte) (int, error) {
	for _, b := range p {
		if b == '\n' {
			w.flush()
			continue
		}
		w.buf.WriteByte(b)
	}
	return len(p), nil
}

func (w *progressLogWriter) flush() {
	if w == nil {
		return
	}
	line := strings.TrimSpace(w.buf.String())
	w.buf.Reset()
	if line == "" || w.sink == nil {
		return
	}
	level := parseLevelFromLine(line)
	w.sink.program.Send(progressLogMsg{line: line, level: level})
}

func parseLevelFromLine(line string) string {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "info"
	}
	level := strings.ToLower(parts[1])
	switch level {
	case "dbg", "debug":
		return "debug"
	case "inf", "info":
		return "info"
	case "wrn", "warn":
		return "warn"
	case "err", "error":
		return "error"
	case "ftl", "fatal":
		return "error"
	default:
		return "info"
	}
}

func spinnerFrame(idx int) string {
	return spinnerFrames[idx%len(spinnerFrames)]
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (m *progressModel) keyForEvent(ev scanexec.ProgressEvent) string {
	if ev.ModuleID != "" {
		return ev.Phase + ":" + ev.ModuleID
	}
	return ev.Phase + ":" + ev.Module
}

func formatModule(ev scanexec.ProgressEvent) string {
	if ev.ModuleID != "" && ev.ModuleID != ev.Module {
		return fmt.Sprintf("%s (%s)", ev.Module, ev.ModuleID)
	}
	return ev.Module
}

func truncate(s string, width int) string {
	if len([]rune(s)) <= width {
		return s
	}
	runes := []rune(s)
	return string(runes[:width-1]) + "…"
}

func finishPhrase(status string) string {
	switch status {
	case "completed":
		return "completed successfully"
	case "failed":
		return "failed"
	case "cancelled":
		return "was cancelled"
	default:
		return status
	}
}

var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	subtleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	infoStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	dividerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	statusBarStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("57")).Padding(0, 1)
)

func styleForStatus(status string) lipgloss.Style {
	switch status {
	case "completed", "success", "done":
		return successStyle
	case "failed", "error", "fatal", "err":
		return errorStyle
	case "warn", "warning", "wrn":
		return warnStyle
	case "debug", "dbg":
		return subtleStyle
	case "info", "inf", "start", "running", "in-progress":
		return infoStyle
	default:
		return warnStyle
	}
}
