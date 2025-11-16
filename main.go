package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Version info - set by goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

// View states
type viewState int

const (
	mainMenuView viewState = iota
	customURLView
	quitConfirmView
	loadingView
	managePresetsView
	addPresetView
	editPresetView
	deleteConfirmView
	restoreDefaultsConfirmView
)

// Messages
type streamURLMsg struct {
	url string
	err error
}
type streamEndedMsg struct{}

// Implement list.Item interface for Preset
func (p Preset) FilterValue() string { return p.Name }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	preset, ok := listItem.(Preset)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, preset.Name)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("• " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list          list.Model
	textInput     textinput.Model
	nameInput     textinput.Model // For add/edit preset name
	urlInput      textinput.Model // For add/edit preset URL
	spinner       spinner.Model
	config        *Config
	state         viewState
	quitting      bool
	width         int
	height        int
	ready         bool   // Track if we've received initial WindowSizeMsg
	loadingTitle  string // What we're loading
	selectedIndex int    // For edit/delete operations
	focusedInput  int    // Which input is focused (0=name, 1=url)
}

func initialModel() model {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		// If config fails to load, use defaults and save them
		config = getDefaultConfig()
		_ = saveConfig(config) // Ignore error on initial save
	}

	// Create list items from config
	items := make([]list.Item, len(config.Presets))
	for i, preset := range config.Presets {
		items[i] = preset
	}

	const defaultWidth = 20

	// Setup list
	l := list.New(items, itemDelegate{}, defaultWidth, len(items))
	l.Title = "LofiTUI - Select a Stream"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false) // Disable default help
	l.DisableQuitKeybindings() // Disable default quit keys
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	// Setup custom URL text input
	ti := textinput.New()
	ti.Placeholder = "Paste YouTube URL here"
	ti.Width = 50

	// Setup name input for add/edit
	ni := textinput.New()
	ni.Placeholder = "Preset Name"
	ni.Width = 50

	// Setup URL input for add/edit
	ui := textinput.New()
	ui.Placeholder = "YouTube URL"
	ui.Width = 50

	// Setup spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		list:      l,
		textInput: ti,
		nameInput: ni,
		urlInput:  ui,
		spinner:   s,
		config:    config,
		state:     mainMenuView,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case streamURLMsg:
		// URL extracted, now play it
		if msg.err != nil {
			// Error loading stream, go back to menu
			m.state = mainMenuView
			return m, nil
		}
		// Launch mpv with the extracted URL
		return m, playMPV(msg.url)

	case streamEndedMsg:
		// Stream finished, return to main menu
		m.state = mainMenuView
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update list dimensions - full width, account for title and help
		listHeight := msg.Height - 4
		if listHeight < 5 {
			listHeight = 5
		}
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(listHeight)

		// Update text input width to be responsive
		inputWidth := msg.Width - 20
		if inputWidth < 30 {
			inputWidth = 30
		}
		if inputWidth > 80 {
			inputWidth = 80
		}
		m.textInput.Width = inputWidth

		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case mainMenuView:
			switch msg.String() {
			case "ctrl+c", "q":
				m.state = quitConfirmView
				return m, nil
			case "c":
				m.state = customURLView
				m.textInput.Focus()
				return m, textinput.Blink
			case "m":
				// Open preset management
				m.state = managePresetsView
				return m, nil
			case "enter":
				// Play selected preset
				if preset, ok := m.list.SelectedItem().(Preset); ok {
					m.state = loadingView
					m.loadingTitle = preset.Name
					return m, tea.Batch(
						spinner.Tick,
						extractStreamURL(preset.URL),
					)
				}
			}

		case customURLView:
			switch msg.String() {
			case "ctrl+c", "esc":
				m.state = mainMenuView
				m.textInput.SetValue("")
				return m, nil
			case "enter":
				url := m.textInput.Value()
				if url != "" {
					m.state = loadingView
					m.loadingTitle = "Custom Stream"
					m.textInput.SetValue("")
					return m, tea.Batch(
						spinner.Tick,
						extractStreamURL(url),
					)
				}
			}

		case quitConfirmView:
			switch msg.String() {
			case "y", "Y":
				m.quitting = true
				return m, tea.Quit
			case "n", "N", "esc":
				m.state = mainMenuView
				return m, nil
			}

		case managePresetsView:
			switch msg.String() {
			case "esc":
				m.state = mainMenuView
				return m, nil
			case "a":
				// Add new preset
				m.state = addPresetView
				m.nameInput.SetValue("")
				m.urlInput.SetValue("")
				m.focusedInput = 0
				m.nameInput.Focus()
				m.urlInput.Blur()
				return m, textinput.Blink
			case "e":
				// Edit selected preset
				if preset, ok := m.list.SelectedItem().(Preset); ok {
					m.state = editPresetView
					m.selectedIndex = m.list.Index()
					m.nameInput.SetValue(preset.Name)
					m.urlInput.SetValue(preset.URL)
					m.focusedInput = 0
					m.nameInput.Focus()
					m.urlInput.Blur()
					return m, textinput.Blink
				}
			case "d", "x":
				// Delete selected preset
				m.state = deleteConfirmView
				m.selectedIndex = m.list.Index()
				return m, nil
			case "r":
				// Restore defaults
				m.state = restoreDefaultsConfirmView
				return m, nil
			case "enter":
				// Play selected preset from manage view
				if preset, ok := m.list.SelectedItem().(Preset); ok {
					m.state = loadingView
					m.loadingTitle = preset.Name
					return m, tea.Batch(
						spinner.Tick,
						extractStreamURL(preset.URL),
					)
				}
			}

		case addPresetView:
			switch msg.String() {
			case "esc":
				m.state = managePresetsView
				return m, nil
			case "tab", "shift+tab":
				// Toggle between name and URL inputs
				if m.focusedInput == 0 {
					m.focusedInput = 1
					m.nameInput.Blur()
					m.urlInput.Focus()
				} else {
					m.focusedInput = 0
					m.nameInput.Focus()
					m.urlInput.Blur()
				}
				return m, textinput.Blink
			case "enter":
				// Save new preset
				name := strings.TrimSpace(m.nameInput.Value())
				url := strings.TrimSpace(m.urlInput.Value())
				if name != "" && url != "" {
					newPreset := Preset{Name: name, URL: url}
					m.config.Presets = append(m.config.Presets, newPreset)
					saveConfig(m.config)
					m = refreshList(m)
					m.state = managePresetsView
				}
				return m, nil
			}

		case editPresetView:
			switch msg.String() {
			case "esc":
				m.state = managePresetsView
				return m, nil
			case "tab", "shift+tab":
				// Toggle between name and URL inputs
				if m.focusedInput == 0 {
					m.focusedInput = 1
					m.nameInput.Blur()
					m.urlInput.Focus()
				} else {
					m.focusedInput = 0
					m.nameInput.Focus()
					m.urlInput.Blur()
				}
				return m, textinput.Blink
			case "enter":
				// Save edited preset
				name := strings.TrimSpace(m.nameInput.Value())
				url := strings.TrimSpace(m.urlInput.Value())
				if name != "" && url != "" && m.selectedIndex < len(m.config.Presets) {
					m.config.Presets[m.selectedIndex] = Preset{Name: name, URL: url}
					saveConfig(m.config)
					m = refreshList(m)
					m.state = managePresetsView
				}
				return m, nil
			}

		case deleteConfirmView:
			switch msg.String() {
			case "y", "Y":
				// Confirm delete
				if m.selectedIndex < len(m.config.Presets) {
					m.config.Presets = append(m.config.Presets[:m.selectedIndex], m.config.Presets[m.selectedIndex+1:]...)
					saveConfig(m.config)
					m = refreshList(m)
				}
				m.state = managePresetsView
				return m, nil
			case "n", "N", "esc":
				m.state = managePresetsView
				return m, nil
			}

		case restoreDefaultsConfirmView:
			switch msg.String() {
			case "y", "Y":
				// Restore defaults
				m.config = getDefaultConfig()
				saveConfig(m.config)
				m = refreshList(m)
				m.state = managePresetsView
				return m, nil
			case "n", "N", "esc":
				m.state = managePresetsView
				return m, nil
			}
		}
	}

	// Update the appropriate widget based on state
	var cmd tea.Cmd
	switch m.state {
	case mainMenuView, managePresetsView:
		m.list, cmd = m.list.Update(msg)
	case customURLView:
		m.textInput, cmd = m.textInput.Update(msg)
	case loadingView:
		m.spinner, cmd = m.spinner.Update(msg)
	case addPresetView, editPresetView:
		if m.focusedInput == 0 {
			m.nameInput, cmd = m.nameInput.Update(msg)
		} else {
			m.urlInput, cmd = m.urlInput.Update(msg)
		}
	}

	return m, cmd
}

// refreshList rebuilds the list from config
func refreshList(m model) model {
	items := make([]list.Item, len(m.config.Presets))
	for i, preset := range m.config.Presets {
		items[i] = preset
	}
	m.list.SetItems(items)
	return m
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	switch m.state {
	case mainMenuView:
		// Reset list title for main menu
		m.list.Title = "LofiTUI - Select a Stream"

		// Show main menu with help text
		helpText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(1, 0, 0, 2).
			Render("m=manage presets • c=custom URL • q=quit")
		return m.list.View() + "\n" + helpText

	case customURLView:
		// Responsive dialog width
		dialogWidth := m.width - 10
		if dialogWidth < 40 {
			dialogWidth = 40
		}
		if dialogWidth > 80 {
			dialogWidth = 80
		}

		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("170")).
			Padding(1, 2).
			Width(dialogWidth)

		content := fmt.Sprintf(
			"Enter Custom YouTube URL\n\n%s\n\n%s",
			m.textInput.View(),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Press Enter to play • ESC to cancel"),
		)

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			style.Render(content),
		)

	case quitConfirmView:
		// Responsive dialog width
		dialogWidth := m.width - 20
		if dialogWidth < 30 {
			dialogWidth = 30
		}
		if dialogWidth > 50 {
			dialogWidth = 50
		}

		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(1, 2).
			Width(dialogWidth)

		content := fmt.Sprintf(
			"Are you sure you want to quit?\n\n%s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Press Y to quit • N to cancel"),
		)

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			style.Render(content),
		)

	case loadingView:
		// Show loading spinner
		content := fmt.Sprintf(
			"%s Loading %s...\n\nPlease wait while we fetch the stream",
			m.spinner.View(),
			m.loadingTitle,
		)

		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2).
			Width(50)

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			style.Render(content),
		)

	case managePresetsView:
		// Update list title for manage view
		m.list.Title = "Manage Presets"

		// Show list with management instructions
		helpText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(1, 0, 0, 2).
			Render("a=add • e=edit • d=delete • r=restore defaults • Enter=play • ESC=back")
		return m.list.View() + "\n" + helpText

	case addPresetView, editPresetView:
		dialogWidth := m.width - 10
		if dialogWidth < 60 {
			dialogWidth = 60
		}
		if dialogWidth > 80 {
			dialogWidth = 80
		}

		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("170")).
			Padding(1, 2).
			Width(dialogWidth)

		title := "Add New Preset"
		if m.state == editPresetView {
			title = "Edit Preset"
		}

		content := fmt.Sprintf(
			"%s\n\nName:\n%s\n\nURL:\n%s\n\n%s",
			title,
			m.nameInput.View(),
			m.urlInput.View(),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Press Enter to save • TAB to switch fields • ESC to cancel"),
		)

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			style.Render(content),
		)

	case deleteConfirmView:
		dialogWidth := m.width - 20
		if dialogWidth < 40 {
			dialogWidth = 40
		}
		if dialogWidth > 60 {
			dialogWidth = 60
		}

		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(1, 2).
			Width(dialogWidth)

		presetName := ""
		if m.selectedIndex < len(m.config.Presets) {
			presetName = m.config.Presets[m.selectedIndex].Name
		}

		content := fmt.Sprintf(
			"Delete preset '%s'?\n\n%s",
			presetName,
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Press Y to confirm • N to cancel"),
		)

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			style.Render(content),
		)

	case restoreDefaultsConfirmView:
		dialogWidth := m.width - 20
		if dialogWidth < 50 {
			dialogWidth = 50
		}
		if dialogWidth > 70 {
			dialogWidth = 70
		}

		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("208")).
			Padding(1, 2).
			Width(dialogWidth)

		content := fmt.Sprintf(
			"Restore Default Presets?\n\n%s\n\n%s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("This will replace all current presets with the original 10 defaults."),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Press Y to confirm • N to cancel"),
		)

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			style.Render(content),
		)
	}

	return ""
}

// extractStreamURL extracts the actual stream URL using yt-dlp
func extractStreamURL(youtubeURL string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("yt-dlp", "-f", "best", "-g", youtubeURL)
		output, err := cmd.Output()
		if err != nil {
			return streamURLMsg{err: err}
		}
		streamURL := strings.TrimSpace(string(output))
		return streamURLMsg{url: streamURL}
	}
}

// playMPV launches mpv with the extracted stream URL
func playMPV(streamURL string) tea.Cmd {
	return tea.ExecProcess(
		exec.Command("mpv", "--vo=tct", "--quiet", "--script=/usr/share/mpv/scripts/mpris.so", streamURL),
		func(err error) tea.Msg {
			// Stream ended (user quit mpv or it errored)
			return streamEndedMsg{}
		},
	)
}

func main() {
	// Parse flags
	versionFlag := flag.Bool("version", false, "Print version information")
	flag.BoolVar(versionFlag, "v", false, "Print version information (shorthand)")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("lofitui %s\ncommit: %s\nbuilt: %s\n", version, commit, date)
		os.Exit(0)
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
