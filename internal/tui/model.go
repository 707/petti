package tui

import (
	"context"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nad/pkgview/internal/app"
	"github.com/nad/pkgview/internal/collectors"
	"github.com/nad/pkgview/internal/model"
)

type ExportFormat string

const (
	ExportTXT  ExportFormat = "txt"
	ExportJSON ExportFormat = "json"
)

type LayoutMode string

const (
	LayoutFull    LayoutMode = "full"
	LayoutCompact LayoutMode = "compact"
)

type Mode string

const (
	ModeNormal Mode = "NORMAL"
	ModeFilter Mode = "FILTER"
	ModeColumn Mode = "SELECT"
	ModeExport Mode = "EXPORT"
)

type PopupKind string

const (
	PopupNone    PopupKind = ""
	PopupSource  PopupKind = "SOURCE"
	PopupAction  PopupKind = "ACTION"
	PopupUpdated PopupKind = "UPDATED"
)

type RefreshFunc func(context.Context) collectors.CollectResult
type ExportFunc func(ExportFormat, []model.Package) (string, error)
type CopyToClipboardFunc func(string) error
type InspectFunc func(context.Context, model.Package) (model.PackageDetails, error)

type Config struct {
	Packages        []model.Package
	Statuses        []model.CollectorStatus
	Filter          string
	Layout          LayoutMode
	Theme           ThemeName
	Refresh         RefreshFunc
	Export          ExportFunc
	CopyToClipboard CopyToClipboardFunc
	Inspect         InspectFunc
}

type Model struct {
	state           app.State
	filter          textinput.Model
	table           table.Model
	grid            viewport.Model
	help            help.Model
	keys            KeyMap
	layout          LayoutMode
	theme           ThemeName
	mode            Mode
	refresh         RefreshFunc
	export          ExportFunc
	copyToClipboard CopyToClipboardFunc
	inspect         InspectFunc
	exportMenu      bool
	exportChoice    int
	columnFocus     int
	popupKind       PopupKind
	popupChoice     int
	detailOpen      bool
	infoOpen        bool
	infoLoading     bool
	infoKey         string
	infoDetails     model.PackageDetails
	infoError       string
	loadingFrame    int
	statusMessage   string
}

type refreshDoneMsg collectors.CollectResult

type exportDoneMsg struct {
	path string
}

type exportFailedMsg struct {
	err error
}

type infoLoadedMsg struct {
	key     string
	details model.PackageDetails
	err     error
}

type loadingTickMsg struct{}

type KeyMap struct {
	Filter  key.Binding
	Sort    key.Binding
	Theme   key.Binding
	Export  key.Binding
	Delete  key.Binding
	Info    key.Binding
	Refresh key.Binding
	Help    key.Binding
	Quit    key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Filter, k.Sort, k.Theme, k.Export, k.Delete, k.Info, k.Refresh, k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Filter, k.Sort, k.Theme, k.Export, k.Info}, {k.Delete, k.Refresh, k.Help, k.Quit}}
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Sort:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		Theme:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "theme")),
		Export:  key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export")),
		Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "copy uninstall cmd")),
		Info:    key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "info")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func New(config Config) Model {
	filter := textinput.New()
	filter.Prompt = "> "
	filter.SetValue(config.Filter)

	tbl := table.New(
		table.WithColumns([]table.Column{
			{Title: "Package", Width: 28},
			{Title: "Version", Width: 16},
			{Title: "Source", Width: 18},
		}),
		table.WithFocused(true),
		table.WithHeight(12),
	)

	keys := DefaultKeyMap()
	helpModel := help.New()
	grid := viewport.New(0, 0)

	m := Model{
		state: app.State{
			Packages: config.Packages,
			Statuses: config.Statuses,
			Filter:   config.Filter,
		},
		filter:          filter,
		table:           tbl,
		grid:            grid,
		help:            helpModel,
		keys:            keys,
		layout:          config.Layout,
		theme:           config.Theme,
		mode:            ModeNormal,
		refresh:         config.Refresh,
		export:          config.Export,
		copyToClipboard: config.CopyToClipboard,
		inspect:         config.Inspect,
	}
	if m.layout == "" {
		m.layout = LayoutFull
	}
	if m.theme == "" || !isValidTheme(m.theme) {
		m.theme = ThemeDefault
	}
	m.detailOpen = false
	if m.refresh == nil {
		m.refresh = func(context.Context) collectors.CollectResult {
			return collectors.CollectResult{
				Packages: m.state.Packages,
				Statuses: m.state.Statuses,
			}
		}
	}
	if m.export == nil {
		m.export = defaultExportFunc
	}
	if m.copyToClipboard == nil {
		m.copyToClipboard = clipboard.WriteAll
	}
	m.syncRows()
	return m
}

func (m Model) ThemeName() ThemeName {
	return m.theme
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if handled, updated, cmd := m.updateNonKey(msg); handled {
		return updated, cmd
	}
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	return m.updateKey(keyMsg)
}

func (m Model) View() string {
	if m.layout == LayoutCompact {
		view := m.renderCompactView()
		if m.infoOpen {
			return m.renderInfoOverlay(view)
		}
		return view
	}
	view := m.renderFullView()
	if m.infoOpen {
		return m.renderInfoOverlay(view)
	}
	return view
}
