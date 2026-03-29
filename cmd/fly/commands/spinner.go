package commands

import (
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// spinnerStyle defines the visual style for the spinner
var spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

// messageStyle defines the style for the spinner message
var messageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("white"))

// SpinnerModel manages the spinner state
type SpinnerModel struct {
	spinner  spinner.Model
	message  string
	quitting bool
	done     bool
	result   error
	mu       sync.Mutex
}

// Init initializes the spinner
func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles spinner updates
func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the spinner
func (m SpinnerModel) View() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.quitting {
		return ""
	}
	return fmt.Sprintf("%s %s", spinnerStyle.Render(m.spinner.View()), messageStyle.Render(m.message))
}

// newSpinnerModel creates a new spinner model
func newSpinnerModel(message string) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle
	return SpinnerModel{
		spinner: s,
		message: message,
	}
}

// WithSpinner runs a function with a spinner displayed
// If VerboseMode or DebugMode is enabled, it runs the function without spinner
func WithSpinner(message string, fn func() error) error {
	// Skip spinner in verbose or debug mode
	if VerboseMode || DebugMode {
		fmt.Printf("%s...\n", message)
		return fn()
	}

	model := newSpinnerModel(message)
	program := tea.NewProgram(model)

	done := make(chan error, 1)
	quit := make(chan struct{})

	go func() {
		if err := fn(); err != nil {
			done <- err
		} else {
			done <- nil
		}
		close(quit)
	}()

	go func() {
		<-quit
		model.mu.Lock()
		model.done = true
		model.mu.Unlock()
		program.Quit()
	}()

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("spinner error: %w", err)
	}

	return <-done
}

// ProgressModel manages a progress bar state
type ProgressModel struct {
	spinner  spinner.Model
	message  string
	current  int
	total    int
	quitting bool
	done     bool
	result   error
	mu       sync.Mutex
}

// Init initializes the progress model
func (m ProgressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles progress updates
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the progress bar
func (m ProgressModel) View() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.quitting {
		return ""
	}

	percentage := 0
	if m.total > 0 {
		percentage = (m.current * 100) / m.total
	}

	// Create a simple progress indicator
	barWidth := 30
	filled := 0
	if m.total > 0 {
		filled = int((int64(barWidth) * int64(m.current)) / int64(m.total))
	}

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Render(bar)

	return fmt.Sprintf("%s %s [%s] %d%%",
		spinnerStyle.Render(m.spinner.View()),
		messageStyle.Render(m.message),
		progressStyle,
		percentage)
}

// newProgressModel creates a new progress model
func newProgressModel(message string, current, total int) ProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle
	return ProgressModel{
		spinner: s,
		message: message,
		current: current,
		total:   total,
	}
}

// WithProgress runs a function with a progress bar displayed
// The fn receives a ProgressUpdater callback to update progress
func WithProgress(message string, current, total int, fn func(updater func(int, int) error) error) error {
	// Skip spinner in verbose or debug mode
	if VerboseMode || DebugMode {
		fmt.Printf("%s...\n", message)
		updater := func(c, t int) error {
			fmt.Printf("  Progress: %d/%d (%d%%)\n", c, t, (c*100)/t)
			return nil
		}
		return fn(updater)
	}

	model := newProgressModel(message, current, total)
	program := tea.NewProgram(model)

	done := make(chan error, 1)
	quit := make(chan struct{})

	updater := func(c, t int) error {
		model.mu.Lock()
		model.current = c
		model.total = t
		model.mu.Unlock()
		return nil
	}

	go func() {
		if err := fn(updater); err != nil {
			done <- err
		} else {
			done <- nil
		}
		close(quit)
	}()

	go func() {
		<-quit
		model.mu.Lock()
		model.done = true
		model.mu.Unlock()
		program.Quit()
	}()

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("progress error: %w", err)
	}

	return <-done
}

// FileProgress tracks file upload/download progress
type FileProgress struct {
	Total   int64
	Current int64
}

// FileProgressUpdater is a callback for file progress updates
type FileProgressUpdater func(current, total int64)

// WithFileProgress runs a function with file progress tracking
func WithFileProgress(message string, totalSize int64, fn func(FileProgressUpdater) error) error {
	// Skip spinner in verbose or debug mode
	if VerboseMode || DebugMode {
		fmt.Printf("%s...\n", message)
		return fn(func(current, total int64) {
			if total > 0 {
				percentage := (current * 100) / total
				fmt.Printf("  Uploaded: %d / %d bytes (%d%%)\n", current, total, percentage)
			}
		})
	}

	model := newFileProgressModel(message, 0, totalSize)
	program := tea.NewProgram(model)

	done := make(chan error, 1)
	quit := make(chan struct{})

	updater := func(current, total int64) {
		model.mu.Lock()
		model.current = current
		model.total = total
		model.mu.Unlock()
	}

	go func() {
		if err := fn(updater); err != nil {
			done <- err
		} else {
			done <- nil
		}
		close(quit)
	}()

	go func() {
		<-quit
		model.mu.Lock()
		model.done = true
		model.mu.Unlock()
		program.Quit()
	}()

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("file progress error: %w", err)
	}

	return <-done
}

// FileProgressModel manages file upload/download progress
type FileProgressModel struct {
	spinner   spinner.Model
	message   string
	current   int64
	total     int64
	quitting  bool
	done      bool
	result    error
	mu        sync.Mutex
	startTime time.Time
}

// Init initializes the file progress model
func (m FileProgressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles file progress updates
func (m FileProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the file progress
func (m FileProgressModel) View() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.quitting {
		return ""
	}

	percentage := 0
	eta := "calculating..."

	if m.total > 0 {
		percentage = int((m.current * 100) / m.total)

		// Calculate ETA
		elapsed := time.Since(m.startTime)
		if m.current > 0 {
			bytesPerSecond := float64(m.current) / elapsed.Seconds()
			remainingBytes := m.total - m.current
			if bytesPerSecond > 0 {
				etaSeconds := int(remainingBytes / int64(bytesPerSecond))
				if etaSeconds < 60 {
					eta = fmt.Sprintf("%ds", etaSeconds)
				} else if etaSeconds < 3600 {
					eta = fmt.Sprintf("%dm %ds", etaSeconds/60, etaSeconds%60)
				} else {
					eta = fmt.Sprintf("%dh %dm", etaSeconds/3600, (etaSeconds%3600)/60)
				}
			}
		}
	}

	// Create progress bar
	barWidth := 30
	filled := 0
	if m.total > 0 {
		filled = int((int64(barWidth) * m.current) / m.total)
	}

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Render(bar)

	currentMB := float64(m.current) / (1024 * 1024)
	_ = currentMB // silence unused variable warning

	return fmt.Sprintf("%s %s [%s] %d%% | %s | ETA: %s\n    %s / %s",
		spinnerStyle.Render(m.spinner.View()),
		messageStyle.Render(m.message),
		progressStyle,
		percentage,
		formatSpeed(m.current, m.startTime),
		eta,
		formatBytes(m.current),
		formatBytes(m.total))
}

func formatSpeed(current int64, start time.Time) string {
	elapsed := time.Since(start)
	if elapsed.Seconds() == 0 || current == 0 {
		return "0 B/s"
	}
	speed := float64(current) / elapsed.Seconds()
	return fmt.Sprintf("%s/s", formatBytes(int64(speed)))
}

func formatBytes(bytes int64) string {
	const unit = int64(1024)
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// newFileProgressModel creates a new file progress model
func newFileProgressModel(message string, current, total int64) FileProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle
	return FileProgressModel{
		spinner:   s,
		message:   message,
		current:   current,
		total:     total,
		startTime: time.Now(),
	}
}

// SimpleSpinner is a simple non-interactive spinner for non-TTY environments
type SimpleSpinner struct {
	message string
}

// NewSimpleSpinner creates a new simple spinner
func NewSimpleSpinner(message string) *SimpleSpinner {
	return &SimpleSpinner{message: message}
}

// Start starts the simple spinner (no-op, returns immediately)
func (s *SimpleSpinner) Start() {
	if VerboseMode || DebugMode {
		fmt.Printf("%s...\n", s.message)
	}
}

// Stop stops the simple spinner (no-op)
func (s *SimpleSpinner) Stop() {}
