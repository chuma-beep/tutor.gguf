package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Options configures the TUI program.
type Options struct {
	TutorURL string
	ASCII    bool
	// Render turns raw assistant content into display text. When nil the raw
	// content is shown verbatim.
	Render func(string) string
}

// Run starts the TUI on an alternate screen and blocks until it exits.
func Run(opts Options) error {
	if opts.TutorURL == "" {
		opts.TutorURL = "http://localhost:8082"
	}
	if opts.Render == nil {
		opts.Render = identity
	}
	p := tea.NewProgram(newModel(opts), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func identity(s string) string { return s }

// Turn is one completed question/answer pair in the transcript.
type Turn struct {
	Question string
	Answer   string
	Err      error
}

// answerMsg carries generation output back into the model. A single blocking
// request arrives as one msg with done=true; streaming sends a sequence of
// appended deltas then a final msg with done=true.
type answerMsg struct {
	id    int
	delta string
	done  bool
	err   error
}

type Model struct {
	opts      Options
	render    func(string) string
	input     textinput.Model
	spinner   spinner.Model
	turns     []Turn
	loading   bool
	streaming *Turn
	nextID    int
	scroll    int

	width, height int
}

func newModel(opts Options) Model {
	ti := textinput.New()
	ti.Placeholder = "Ask a math question (Enter to send, Ctrl+C to quit)"
	ti.Prompt = "> "
	ti.CharLimit = 512
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	return Model{
		opts:    opts,
		render:  opts.Render,
		input:   ti,
		spinner: sp,
		scroll:  -1, // -1 == pinned to bottom
		nextID:  1,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "ctrl+l":
			m.turns = nil
			m.streaming = nil
			m.loading = false
			m.scroll = -1
			return m, nil
		case "pgup":
			if m.scroll == -1 {
				m.scroll = 0
			} else {
				m.scroll++
			}
			return m, nil
		case "pgdown":
			if m.scroll > 0 {
				m.scroll--
			} else {
				m.scroll = -1
			}
			return m, nil
		case "enter":
			q := strings.TrimSpace(m.input.Value())
			if q == "" || m.loading {
				return m, nil
			}
			m.input.SetValue("")
			m.loading = true
			m.streaming = &Turn{Question: q}
			id := m.nextID
			m.nextID++
			return m, tea.Batch(m.spinner.Tick, askCmd(m.opts.TutorURL, q, id))
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case answerMsg:
		if msg.id != m.nextID-1 || m.streaming == nil {
			return m, nil
		}
		if msg.err != nil {
			m.streaming.Err = msg.err
			m.turns = append(m.turns, *m.streaming)
			m.streaming = nil
			m.loading = false
			m.scroll = -1
			return m, nil
		}
		m.streaming.Answer += msg.delta
		if msg.done {
			m.turns = append(m.turns, *m.streaming)
			m.streaming = nil
			m.loading = false
			m.scroll = -1
		}
		return m, nil
	}
	return m, nil
}

// askCmd runs a single blocking request for a given request id. Streaming
// replaces this in a later step; the blocking path is the correctness baseline.
func askCmd(tutorURL, q string, id int) tea.Cmd {
	return func() tea.Msg {
		content, err := Ask(tutorURL, q)
		if err != nil {
			return answerMsg{id: id, err: err}
		}
		return answerMsg{id: id, delta: content, done: true}
	}
}

func (m Model) View() string {
	if m.width == 0 {
		m.width = 80
	}
	if m.height == 0 {
		m.height = 24
	}

	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Render("tutor.gguf — on-device math tutor"))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("server: %s    Ctrl+L clear · PgUp/PgDn scroll", m.opts.TutorURL)))
	b.WriteString("\n\n")

	b.WriteString(m.transcriptView())

	b.WriteString("\n\ninput: ")
	if m.loading {
		b.WriteString(m.spinner.View())
		b.WriteString(" thinking")
	}
	b.WriteByte('\n')
	b.WriteString(m.input.View())

	return b.String()
}

// transcriptView renders the scrollable question/answer history, pinning to the
// bottom by request. It always renders the in-flight streaming answer (with a
// trailing cursor when not done) so partial output is visible.
func (m Model) transcriptView() string {
	lines := m.transcriptLines()
	if m.scroll == -1 {
		m.scroll = transcriptOverscan(m.height, lines)
	}

	top := -1 * m.scroll
	if top < 0 {
		top = 0
	}
	bottom := top + m.height - 6
	if bottom > len(lines) {
		bottom = len(lines)
	}
	if top > bottom {
		top = bottom
	}

	body := strings.Join(lines[top:bottom], "\n")
	if body == "" {
		return ""
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		MaxHeight(m.height - 6).
		Width(m.width - 2).
		Render(body)
}

// transcriptLines flattens the history into wrapped, pre-rendered viewport
// lines (one question line, answer lines, separator between turns).
func (m Model) transcriptLines() []string {
	wrapAt := m.width - 8
	if wrapAt < 10 {
		wrapAt = 10
	}

	var lines []string
	for _, t := range m.turns {
		lines = append(lines, questionLine(t.Question))
		if t.Err != nil {
			lines = append(lines, lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Render("error: "+t.Err.Error()))
		} else {
			lines = append(lines, wrapString(m.render(t.Answer), wrapAt)...)
		}
		lines = append(lines, "")
	}

	if m.streaming != nil {
		t := m.streaming
		lines = append(lines, questionLine(t.Question))
		if t.Err != nil {
			lines = append(lines, lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Render("error: "+t.Err.Error()))
		} else {
			shown := m.render(t.Answer)
			if m.loading {
				shown += " ▍"
			}
			lines = append(lines, wrapString(shown, wrapAt)...)
		}
		lines = append(lines, "")
	}

	return lines
}

func questionLine(q string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214")).
		Render("Q: " + q)
}

// transcriptOverscan is how many lines the transcript exceeds its pane; used
// to compute the bottom-pinned scroll offset.
func transcriptOverscan(paneH int, lines []string) int {
	body := paneH - 6
	if body < 1 {
		body = 1
	}
	overscan := len(lines) - body
	if overscan < 0 {
		return 0
	}
	return overscan
}

func wrapString(s string, width int) []string {
	if s == "" {
		return []string{""}
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(s)
	return strings.Split(strings.TrimSuffix(wrapped, "\n"), "\n")
}
