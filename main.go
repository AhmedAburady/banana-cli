package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"nano_banana_pro/api"
	"nano_banana_pro/ui"
	"nano_banana_pro/views"
)

// ViewState represents the current view
type ViewState int

const (
	MenuView ViewState = iota
	GenerateView
	EditView
	ProcessingView
	ResultsView
)

// Model is the main application model
type Model struct {
	currentView   ViewState
	menuModel     ui.MenuModel
	generateModel views.GenerateModel
	editModel     views.EditModel

	apiKey  string
	width   int
	height  int
	spinner spinner.Model

	// Processing state
	processingMsg string
	results       []api.GenerationResult
	outputFolder  string
	successCount  int
	errorCount    int
	elapsed       time.Duration
}

// ProcessingStartMsg signals the start of image generation
type ProcessingStartMsg struct {
	Config *api.Config
}

// ProcessingDoneMsg signals completion of image generation
type ProcessingDoneMsg struct {
	Results      []api.GenerationResult
	OutputFolder string
	Elapsed      time.Duration
}

// NewModel creates a new application model
func NewModel(apiKey string) Model {
	menuStyles := ui.MenuStyles{
		Window:       ui.WindowStyle,
		Title:        ui.TitleStyle,
		Item:         ui.MenuItemStyle,
		SelectedItem: ui.MenuSelectedStyle,
		Help:         ui.HelpStyle,
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.DraculaPurple)

	return Model{
		currentView: MenuView,
		menuModel:   ui.NewMenuModel(menuStyles),
		apiKey:      apiKey,
		spinner:     s,
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.menuModel.Init(),
		tea.EnterAltScreen,
	)
}

// Update handles all application messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Global: Ctrl+C quits
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+c" {
		return m, tea.Quit
	}

	// Handle window resize
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = size.Width
		m.height = size.Height
		m.menuModel.SetSize(size.Width, size.Height)
	}

	// Route to current view
	switch m.currentView {
	case MenuView:
		return m.updateMenuView(msg)

	case GenerateView:
		return m.updateGenerateView(msg)

	case EditView:
		return m.updateEditView(msg)

	case ProcessingView:
		return m.updateProcessingView(msg)

	case ResultsView:
		return m.updateResultsView(msg)
	}

	return m, cmd
}

func (m Model) updateMenuView(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle menu selection
	if sel, ok := msg.(ui.MenuSelectionMsg); ok {
		switch sel.Choice {
		case ui.GenerateImage:
			m.currentView = GenerateView
			m.generateModel = views.NewGenerateModel()
			return m, m.generateModel.Init()
		case ui.EditImage:
			m.currentView = EditView
			m.editModel = views.NewEditModel()
			return m, m.editModel.Init()
		}
	}

	var cmd tea.Cmd
	m.menuModel, cmd = m.menuModel.Update(msg)
	return m, cmd
}

func (m Model) updateGenerateView(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle back to menu
	if _, ok := msg.(ui.BackToMenuMsg); ok {
		m.currentView = MenuView
		return m, nil
	}

	// Handle form submission
	if submit, ok := msg.(views.GenerateSubmitMsg); ok {
		config := &api.Config{
			OutputFolder: submit.OutputFolder,
			NumImages:    submit.NumImages,
			Prompt:       submit.Prompt,
			APIKey:       m.apiKey,
			AspectRatio:  submit.AspectRatio,
			ImageSize:    submit.ImageSize,
			Grounding:    submit.Grounding,
			RefImages:    nil, // No reference images for generate
		}

		m.currentView = ProcessingView
		m.processingMsg = fmt.Sprintf("Generating %d image(s)...", submit.NumImages)
		m.outputFolder = submit.OutputFolder

		return m, func() tea.Msg {
			return ProcessingStartMsg{Config: config}
		}
	}

	var cmd tea.Cmd
	m.generateModel, cmd = m.generateModel.Update(msg)
	return m, cmd
}

func (m Model) updateEditView(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle back to menu
	if _, ok := msg.(ui.BackToMenuMsg); ok {
		m.currentView = MenuView
		return m, nil
	}

	// Handle form submission
	if submit, ok := msg.(views.EditSubmitMsg); ok {
		// Load reference images
		refImages, err := api.LoadReferences(submit.ReferencePath)
		if err != nil {
			// Show error and stay in edit view
			m.currentView = ResultsView
			m.errorCount = 1
			m.successCount = 0
			m.results = []api.GenerationResult{{
				Index: 0,
				Error: fmt.Errorf("failed to load references: %v", err),
			}}
			return m, nil
		}

		config := &api.Config{
			OutputFolder: submit.OutputFolder,
			NumImages:    submit.NumImages,
			Prompt:       submit.Prompt,
			APIKey:       m.apiKey,
			AspectRatio:  submit.AspectRatio,
			ImageSize:    submit.ImageSize,
			Grounding:    submit.Grounding,
			RefImages:    refImages,
		}

		m.currentView = ProcessingView
		m.processingMsg = fmt.Sprintf("Generating %d image(s) with %d reference(s)...", submit.NumImages, len(refImages))
		m.outputFolder = submit.OutputFolder

		return m, func() tea.Msg {
			return ProcessingStartMsg{Config: config}
		}
	}

	var cmd tea.Cmd
	m.editModel, cmd = m.editModel.Update(msg)
	return m, cmd
}

func (m Model) updateProcessingView(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle processing start
	if start, ok := msg.(ProcessingStartMsg); ok {
		return m, tea.Batch(
			m.spinner.Tick,
			func() tea.Msg {
				return runGeneration(start.Config)
			},
		)
	}

	// Handle processing done
	if done, ok := msg.(ProcessingDoneMsg); ok {
		m.results = done.Results
		m.outputFolder = done.OutputFolder
		m.elapsed = done.Elapsed
		m.currentView = ResultsView

		// Count successes and errors
		m.successCount = 0
		m.errorCount = 0
		for _, r := range done.Results {
			if r.Error != nil {
				m.errorCount++
			} else {
				m.successCount++
			}
		}

		return m, nil
	}

	// Update spinner
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m Model) updateResultsView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter", " ", "esc", "q":
			m.currentView = MenuView
			m.results = nil
			return m, nil
		}
	}
	return m, nil
}

// runGeneration performs the actual image generation
func runGeneration(config *api.Config) tea.Msg {
	startTime := time.Now()

	// Ensure output folder exists
	if err := os.MkdirAll(config.OutputFolder, 0755); err != nil {
		return ProcessingDoneMsg{
			Results: []api.GenerationResult{{
				Index: 0,
				Error: fmt.Errorf("failed to create output folder: %v", err),
			}},
			OutputFolder: config.OutputFolder,
			Elapsed:      time.Since(startTime),
		}
	}

	// Generate images in parallel
	var wg sync.WaitGroup
	resultsChan := make(chan api.GenerationResult, config.NumImages)

	for i := 0; i < config.NumImages; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result := api.GenerateImage(config, index)

			// Save if successful
			if result.Error == nil && result.ImageData != nil {
				filename := fmt.Sprintf("generated_%d_%s.png", result.Index+1, time.Now().Format("20060102_150405"))
				outputFile := filepath.Join(config.OutputFolder, filename)
				if err := os.WriteFile(outputFile, result.ImageData, 0644); err != nil {
					result.Error = fmt.Errorf("failed to save: %v", err)
				} else {
					result.Filename = filename
				}
			}

			resultsChan <- result
		}(i)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	var results []api.GenerationResult
	for result := range resultsChan {
		results = append(results, result)
	}

	return ProcessingDoneMsg{
		Results:      results,
		OutputFolder: config.OutputFolder,
		Elapsed:      time.Since(startTime),
	}
}

// View renders the current view
func (m Model) View() string {
	var content string

	switch m.currentView {
	case MenuView:
		content = m.renderMenuView()
	case GenerateView:
		content = m.renderFormView(m.generateModel.View())
	case EditView:
		content = m.renderFormView(m.editModel.View())
	case ProcessingView:
		content = m.renderProcessingView()
	case ResultsView:
		content = m.renderResultsView()
	}

	return content
}

func (m Model) renderMenuView() string {
	banner := ui.RenderGradientBanner()
	subtitle := ui.RenderSubtitle()

	header := lipgloss.JoinVertical(lipgloss.Center,
		"",
		banner,
		subtitle,
		"",
	)

	menuContent := m.menuModel.View()

	content := lipgloss.JoinVertical(lipgloss.Center,
		header,
		menuContent,
	)

	window := ui.WindowStyle.Width(110).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, window)
}

func (m Model) renderFormView(formContent string) string {
	window := ui.WindowStyle.Width(80).Height(32).Render(formContent)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, window)
}

func (m Model) renderProcessingView() string {
	spinnerStyle := lipgloss.NewStyle().Foreground(ui.DraculaPurple)
	msgStyle := lipgloss.NewStyle().Foreground(ui.DraculaCyan).Bold(true)

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		spinnerStyle.Render(m.spinner.View())+" "+msgStyle.Render(m.processingMsg),
		"",
		ui.SubtleStyle.Render("Please wait..."),
		"",
	)

	window := ui.WindowStyle.Width(60).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, window)
}

func (m Model) renderResultsView() string {
	var lines []string

	lines = append(lines, "")
	lines = append(lines, ui.TitleStyle.Render("Results"))
	lines = append(lines, "")

	// Show individual results
	for _, r := range m.results {
		if r.Error != nil {
			lines = append(lines, ui.ErrorStyle.Render(fmt.Sprintf("[X] Image %d: %v", r.Index+1, r.Error)))
		} else {
			lines = append(lines, ui.SuccessStyle.Render(fmt.Sprintf("[+] %s", r.Filename)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, ui.SubtleStyle.Render("------------------------------"))
	lines = append(lines, ui.InfoStyle.Render(fmt.Sprintf("Summary: %d success, %d failed", m.successCount, m.errorCount)))
	lines = append(lines, ui.InfoStyle.Render(fmt.Sprintf("Time: %s", m.elapsed.Round(time.Millisecond))))
	lines = append(lines, ui.InfoStyle.Render(fmt.Sprintf("Output: %s", m.outputFolder)))
	lines = append(lines, "")
	lines = append(lines, ui.HelpStyle.Render("Press any key to continue..."))
	lines = append(lines, "")

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	window := ui.WindowStyle.Width(70).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, window)
}

func main() {
	// Get API key from environment
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if apiKey == "" {
		fmt.Println("Error: GEMINI_API_KEY environment variable not set")
		fmt.Println("Set it with: export GEMINI_API_KEY=your_key")
		os.Exit(1)
	}

	// Create and run the program
	p := tea.NewProgram(NewModel(apiKey), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
