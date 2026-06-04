package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	focusMethod = iota
	focusURL
	focusTabs
	focusPanel
	focusBearer
	focusSend
)

type responseMsg struct {
	result responseResult
	err    error
}

type responseResult struct {
	status     string
	statusCode int
	headers    http.Header
	body       []byte
	duration   time.Duration
}

type savedConfig struct {
	Method    string `json:"method"`
	URL       string `json:"url"`
	Body      string `json:"body"`
	Headers   string `json:"headers"`
	Bearer    string `json:"bearer"`
	ActiveTab int    `json:"active_tab"`
}

type model struct {
	methods  table.Model
	url      textinput.Model
	body     textarea.Model
	headers  textarea.Model
	bearer   textinput.Model
	response viewport.Model
	spinner  spinner.Model

	focus        int
	activeTab    int
	tabsUnlocked bool
	loading      bool
	responseSet  bool
	status       string
	width        int
	height       int
	lastBody     []byte
	httpClient   *http.Client
	panelStyles  panelStyles
}

type panelStyles struct {
	tab        lipgloss.Style
	activeTab  lipgloss.Style
	input      lipgloss.Style
	focusInput lipgloss.Style
	button     lipgloss.Style
	buttonHot  lipgloss.Style
	error      lipgloss.Style
	ok         lipgloss.Style
	muted      lipgloss.Style
}

func main() {
	if _, err := tea.NewProgram(newModel()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func newModel() model {
	styles := newStyles()
	cfg := loadConfig()

	methods := table.New(
		table.WithColumns([]table.Column{{Title: "", Width: 8}}),
		table.WithRows([]table.Row{{"GET"}, {"POST"}, {"PUT"}, {"PATCH"}, {"DELETE"}}),
		table.WithFocused(true),
		table.WithHeight(1),
		table.WithWidth(10),
	)
	methodStyles := table.DefaultStyles()
	methodStyles.Header = methodStyles.Header.
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	methodStyles.Selected = methodStyles.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	methods.SetStyles(methodStyles)

	url := textinput.New()
	url.Placeholder = "localhost:5000"
	url.SetValue("localhost:5000")
	if cfg.URL != "" {
		url.SetValue(cfg.URL)
	}
	url.SetVirtualCursor(false)
	url.SetWidth(38)
	url.CharLimit = 2048
	url.KeyMap.DeleteWordBackward = key.NewBinding(key.WithKeys("alt+backspace", "ctrl+backspace", "ctrl+w"))
	url.KeyMap.DeleteWordForward = key.NewBinding(key.WithKeys("alt+delete", "ctrl+delete", "alt+d"))

	body := textarea.New()
	body.Placeholder = ""
	body.SetValue(cfg.Body)
	body.SetVirtualCursor(false)
	body.CharLimit = 50_000
	body.SetWidth(42)
	body.SetHeight(9)
	body.KeyMap.InsertNewline.SetEnabled(true)
	body.KeyMap.WordForward = key.NewBinding(key.WithKeys("alt+right", "ctrl+right", "alt+f"))
	body.KeyMap.WordBackward = key.NewBinding(key.WithKeys("alt+left", "ctrl+left", "alt+b"))
	body.KeyMap.DeleteWordBackward = key.NewBinding(key.WithKeys("alt+backspace", "ctrl+backspace", "ctrl+w"))
	body.KeyMap.DeleteWordForward = key.NewBinding(key.WithKeys("alt+delete", "ctrl+delete", "alt+d"))
	body.Blur()
	body.SetStyles(noCursorLineStyles(body))

	headers := textarea.New()
	headers.Placeholder = "Header Input Box\nContent-Type: application/json"
	headers.SetValue("Content-Type: application/json")
	if cfg.Headers != "" {
		headers.SetValue(cfg.Headers)
	}
	headers.SetVirtualCursor(false)
	headers.CharLimit = 20_000
	headers.SetWidth(42)
	headers.SetHeight(9)
	headers.KeyMap.InsertNewline.SetEnabled(true)
	headers.KeyMap.WordForward = key.NewBinding(key.WithKeys("alt+right", "ctrl+right", "alt+f"))
	headers.KeyMap.WordBackward = key.NewBinding(key.WithKeys("alt+left", "ctrl+left", "alt+b"))
	headers.KeyMap.DeleteWordBackward = key.NewBinding(key.WithKeys("alt+backspace", "ctrl+backspace", "ctrl+w"))
	headers.KeyMap.DeleteWordForward = key.NewBinding(key.WithKeys("alt+delete", "ctrl+delete", "alt+d"))
	headers.Blur()
	headers.SetStyles(noCursorLineStyles(headers))

	bearer := textinput.New()
	bearer.Placeholder = "Bearer Token Input Box"
	bearer.SetValue(cfg.Bearer)
	bearer.SetVirtualCursor(false)
	bearer.SetWidth(42)
	bearer.CharLimit = 4096
	bearer.KeyMap.DeleteWordBackward = key.NewBinding(key.WithKeys("alt+backspace", "ctrl+backspace", "ctrl+w"))
	bearer.KeyMap.DeleteWordForward = key.NewBinding(key.WithKeys("alt+delete", "ctrl+delete", "alt+d"))
	bearer.Blur()

	response := viewport.New(viewport.WithWidth(80), viewport.WithHeight(10))
	response.SetContent("No response yet.")

	spin := spinner.New()
	spin.Spinner = spinner.Dot

	m := model{
		methods:     methods,
		url:         url,
		body:        body,
		headers:     headers,
		bearer:      bearer,
		response:    response,
		spinner:     spin,
		focus:       focusMethod,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		panelStyles: styles,
	}
	if cfg.ActiveTab >= 0 && cfg.ActiveTab <= 2 {
		m.activeTab = cfg.ActiveTab
	}
	if cfg.Method != "" {
		for i, row := range m.methods.Rows() {
			if len(row) > 0 && row[0] == cfg.Method {
				m.methods.SetCursor(i)
				break
			}
		}
	}
	return m
}

func newStyles() panelStyles {
	return panelStyles{
		tab: lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			PaddingRight(3),
		activeTab: lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Underline(true).
			PaddingRight(3),
		input: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("250")).
			Padding(0, 1),
		focusInput: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("229")).
			Padding(0, 1),
		button: lipgloss.NewStyle().
			Foreground(lipgloss.Color("236")).
			Background(lipgloss.Color("252")).
			Padding(0, 3).
			MarginRight(1),
		buttonHot: lipgloss.NewStyle().
			Foreground(lipgloss.Color("236")).
			Background(lipgloss.Color("229")).
			Padding(0, 3).
			MarginRight(1),
		error: lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
		ok:    lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		muted: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	}
}

func noCursorLineStyles(ta textarea.Model) textarea.Styles {
	s := ta.Styles()
	s.Focused.CursorLine = lipgloss.NewStyle()
	s.Blurred.CursorLine = lipgloss.NewStyle()
	prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	s.Focused.Prompt = prompt
	s.Blurred.Prompt = prompt
	return s
}

func configPath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "api", "config.json"), nil
}

func loadConfig() savedConfig {
	path, err := configPath()
	if err != nil {
		return savedConfig{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return savedConfig{}
	}
	var cfg savedConfig
	if json.Unmarshal(data, &cfg) != nil {
		return savedConfig{}
	}
	return cfg
}

func (m model) saveConfig() {
	path, err := configPath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	cfg := savedConfig{
		Method:    m.method(),
		URL:       m.url.Value(),
		Body:      m.body.Value(),
		Headers:   m.headers.Value(),
		Bearer:    m.bearer.Value(),
		ActiveTab: m.activeTab,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, textarea.Blink)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		panelWidth := clamp(msg.Width-20, 42, 72)
		m.body.SetWidth(panelWidth)
		m.headers.SetWidth(panelWidth)
		m.bearer.SetWidth(panelWidth)
		m.response.SetWidth(clamp(msg.Width-20, 42, 88))
		m.response.SetHeight(clamp(msg.Height-25, 8, 18))
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.saveConfig()
			return m, tea.Quit
		case "tab":
			m = m.nextFocus()
		case "shift+tab":
			m = m.prevFocus()
		case "left", "h":
			if m.focus == focusTabs {
				m.activeTab = wrap(m.activeTab-1, 3)
				m = m.focusTabPanel()
			}
		case "right", "l":
			if m.focus == focusTabs {
				m.activeTab = wrap(m.activeTab+1, 3)
				m = m.focusTabPanel()
			}
		case "enter":
			if m.focus == focusMethod || m.focus == focusURL || m.focus == focusTabs || m.focus == focusBearer || m.focus == focusSend {
				return m.startRequest()
			}
		case "ctrl+s":
			return m.startRequest()
		}
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			var mouseCmd tea.Cmd
			m, mouseCmd = m.focusFromClick(msg.Mouse())
			if mouseCmd != nil {
				return m, mouseCmd
			}
		}
	case tea.MouseWheelMsg:
		mouse := msg.Mouse()
		if m.focus == focusMethod || (mouse.Y >= 0 && mouse.Y <= 2 && mouse.X <= 12) {
			switch mouse.Button {
			case tea.MouseWheelUp:
				m.methods.MoveUp(1)
			case tea.MouseWheelDown:
				m.methods.MoveDown(1)
			}
		}
	case responseMsg:
		m.loading = false
		if msg.err != nil {
			m.status = m.panelStyles.error.Render(msg.err.Error())
			m.response.SetContent("Request failed:\n" + msg.err.Error())
			m.responseSet = true
			return m, nil
		}
		m.lastBody = msg.result.body
		m.response.SetContent(renderResponse(msg.result))
		m.response.GotoTop()
		m.responseSet = true
		m.status = ""
	case cursor.BlinkMsg:
		switch m.focus {
		case focusURL:
			m.url, cmd = m.url.Update(msg)
		case focusPanel:
			m = m.updateActivePanelCursor(msg)
		case focusBearer:
			m.bearer, cmd = m.bearer.Update(msg)
		}
		return m, cmd
	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	switch m.focus {
	case focusMethod:
		m.methods, cmd = m.methods.Update(msg)
	case focusURL:
		m.url, cmd = m.url.Update(msg)
	case focusPanel:
		m, cmd = m.updateActivePanel(msg)
	case focusBearer:
		m.bearer, cmd = m.bearer.Update(msg)
	case focusSend:
		if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "enter" {
			return m.startRequest()
		}
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if _, ok := msg.(tea.KeyPressMsg); ok {
		m.saveConfig()
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	methodStyle := m.panelStyles.input
	if m.focus == focusMethod {
		methodStyle = m.panelStyles.focusInput
	}
	methodView := methodStyle.Render(m.method())
	urlStyle := m.panelStyles.input
	if m.focus == focusURL {
		urlStyle = m.panelStyles.focusInput
	}
	top := lipgloss.JoinHorizontal(lipgloss.Top, methodView, "  ", urlStyle.Render(m.url.View()))

	sections := []string{
		top,
		"",
		m.tabsView(),
		m.panelView(),
		"",
		"",
		m.sendView(),
	}
	if response := m.responseView(); response != "" {
		sections = append(sections, "", response)
	}
	if m.status != "" {
		sections = append(sections, "", m.status)
	}

	str := lipgloss.JoinVertical(lipgloss.Left, sections...)

	view := tea.NewView(str)
	if c := m.cursor(); c != nil {
		view.Cursor = c
	}
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	return view
}

func (m model) panelView() string {
	switch m.activeTab {
	case 1:
		return m.headers.View()
	case 2:
		style := m.panelStyles.input
		if m.focus == focusBearer {
			style = m.panelStyles.focusInput
		}
		return style.Render(m.bearer.View())
	default:
		return m.body.View()
	}
}

func (m model) tabsView() string {
	tabs := []string{"Body", "Header", "Bearer"}
	out := make([]string, len(tabs))
	for i, tab := range tabs {
		style := m.panelStyles.tab
		if i == m.activeTab {
			style = m.panelStyles.activeTab
		}
		out[i] = style.Render(tab)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, out...)
}

func (m model) sendView() string {
	label := "Send"
	if m.loading {
		label = m.spinner.View() + " Sending"
	}
	style := m.panelStyles.button
	if m.focus == focusSend {
		style = m.panelStyles.buttonHot
	}
	return style.Render(label)
}

func (m model) responseView() string {
	if !m.responseSet {
		return ""
	}
	return m.panelStyles.input.Width(clamp(m.width-20, 42, 88)).Render(m.response.View())
}

func (m model) cursor() *tea.Cursor {
	addOffset := func(c *tea.Cursor, x, y int) *tea.Cursor {
		if c == nil {
			return nil
		}
		next := *c
		next.X += x
		next.Y += y
		return &next
	}

	switch m.focus {
	case focusMethod:
		return nil
	case focusURL:
		methodStyle := m.panelStyles.input
		if m.focus == focusMethod {
			methodStyle = m.panelStyles.focusInput
		}
		methodWidth := lipgloss.Width(methodStyle.Render(m.method()))
		return addOffset(m.url.Cursor(), methodWidth+4, 1)
	case focusPanel:
		if m.activeTab == 0 {
			return addOffset(m.body.Cursor(), 0, 5)
		}
		if m.activeTab == 1 {
			return addOffset(m.headers.Cursor(), 0, 5)
		}
	case focusBearer:
		return addOffset(m.bearer.Cursor(), 2, 6)
	}
	return nil
}

func (m model) updateActivePanel(msg tea.Msg) (model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.activeTab {
	case 1:
		m.headers, cmd = m.headers.Update(msg)
	default:
		m.body, cmd = m.body.Update(msg)
	}
	return m, cmd
}

func (m model) updateActivePanelCursor(msg tea.Msg) model {
	switch m.activeTab {
	case 1:
		m.headers, _ = m.headers.Update(msg)
	default:
		m.body, _ = m.body.Update(msg)
	}
	return m
}

func (m model) nextFocus() model {
	m.blurAll()
	switch m.focus {
	case focusMethod:
		m.focus = focusURL
	case focusURL:
		if m.tabsUnlocked {
			m.focus = focusTabs
		} else {
			m.tabsUnlocked = true
			if m.activeTab == 2 {
				m.focus = focusBearer
			} else {
				m.focus = focusPanel
			}
		}
	case focusTabs:
		if m.activeTab == 2 {
			m.focus = focusBearer
		} else {
			m.focus = focusPanel
		}
	case focusPanel:
		m.focus = focusSend
	case focusBearer:
		m.focus = focusSend
	case focusSend:
		m.focus = focusMethod
	default:
		m.focus = focusMethod
	}
	return m.focusCurrent()
}

func (m model) prevFocus() model {
	m.blurAll()
	switch m.focus {
	case focusMethod:
		m.focus = focusSend
	case focusURL:
		m.focus = focusMethod
	case focusTabs:
		m.focus = focusURL
	case focusPanel:
		if m.tabsUnlocked {
			m.focus = focusTabs
		} else {
			m.tabsUnlocked = true
			m.focus = focusURL
		}
	case focusBearer:
		if m.tabsUnlocked {
			m.focus = focusTabs
		} else {
			m.tabsUnlocked = true
			m.focus = focusURL
		}
	case focusSend:
		if m.activeTab == 2 {
			m.focus = focusBearer
		} else {
			m.focus = focusPanel
		}
	default:
		m.focus = focusSend
	}
	return m.focusCurrent()
}

func (m model) focusFromClick(mouse tea.Mouse) (model, tea.Cmd) {
	m = m.blurAll()
	bodyStart := 5
	bodyEnd := 14
	sendStart := 15
	sendEnd := 17
	switch {
	case mouse.Y >= 0 && mouse.Y <= 2:
		if mouse.X <= 12 {
			m.focus = focusMethod
		} else {
			m.focus = focusURL
		}
	case mouse.Y == 4:
		m.focus = focusTabs
		switch {
		case mouse.X < 8:
			m.activeTab = 0
		case mouse.X < 18:
			m.activeTab = 1
		default:
			m.activeTab = 2
		}
	case mouse.Y >= sendStart && mouse.Y <= sendEnd:
		m.focus = focusSend
		m = m.focusCurrent()
		return m.startRequestModel()
	case mouse.Y >= bodyStart && mouse.Y <= bodyEnd:
		if m.activeTab == 2 {
			m.focus = focusBearer
		} else {
			m.focus = focusPanel
		}
	default:
		m.focus = focusMethod
	}
	return m.focusCurrent(), nil
}

func (m model) focusTabPanel() model {
	m.blurAll()
	if m.activeTab == 2 {
		m.focus = focusBearer
	} else {
		m.focus = focusPanel
	}
	return m.focusCurrent()
}

func (m model) startRequestModel() (model, tea.Cmd) {
	m.saveConfig()
	cfg, err := m.requestConfig()
	if err != nil {
		m.status = m.panelStyles.error.Render(err.Error())
		return m, nil
	}
	m.loading = true
	m.status = "Sending..."
	return m, tea.Batch(m.spinner.Tick, sendRequestCmd(m.httpClient, cfg))
}

func (m model) blurAll() model {
	m.methods.Blur()
	m.url.Blur()
	m.body.Blur()
	m.headers.Blur()
	m.bearer.Blur()
	return m
}

func (m model) focusCurrent() model {
	switch m.focus {
	case focusMethod:
		m.methods.Focus()
	case focusURL:
		m.url.Focus()
	case focusPanel:
		if m.activeTab == 1 {
			m.headers.Focus()
		} else {
			m.body.Focus()
		}
	case focusBearer:
		m.bearer.Focus()
	}
	return m
}

func (m model) startRequest() (tea.Model, tea.Cmd) {
	next, cmd := m.startRequestModel()
	return next, cmd
}

func (m model) requestConfig() (*http.Request, error) {
	rawURL := strings.TrimSpace(m.url.Value())
	if rawURL == "" {
		return nil, fmt.Errorf("URL required")
	}

	body := strings.TrimSpace(m.body.Value())
	req, err := http.NewRequest(m.method(), normalizeURL(rawURL), strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	headers, err := parseHeaders(m.headers.Value())
	if err != nil {
		return nil, err
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if token := strings.TrimSpace(m.bearer.Value()); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (m model) method() string {
	row := m.methods.SelectedRow()
	if len(row) == 0 {
		return "GET"
	}
	return row[0]
}

func sendRequestCmd(client *http.Client, req *http.Request) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			return responseMsg{err: err}
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return responseMsg{err: err}
		}
		return responseMsg{
			result: responseResult{
				status:     resp.Status,
				statusCode: resp.StatusCode,
				headers:    resp.Header,
				body:       body,
				duration:   time.Since(start),
			},
		}
	}
}

func parseHeaders(raw string) (http.Header, error) {
	headers := http.Header{}
	for i, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("header line %d must use Name: value", i+1)
		}
		headers.Add(strings.TrimSpace(name), strings.TrimSpace(value))
	}
	return headers, nil
}

func renderResponse(result responseResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", statusStyle(result.statusCode).Render(result.status))
	fmt.Fprintf(&b, "Time: %s\n", result.duration.Round(time.Millisecond))
	fmt.Fprintf(&b, "Size: %d bytes\n\n", len(result.body))
	fmt.Fprintf(&b, "Headers:\n%s\n", formatHeaders(result.headers))
	fmt.Fprintf(&b, "Body:\n%s", previewBody(result.body))
	return b.String()
}

func statusStyle(code int) lipgloss.Style {
	switch {
	case code >= 200 && code < 300:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	case code >= 300 && code < 400:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true)
	case code >= 500:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("178")).Bold(true)
	case code >= 400:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Bold(true)
	}
}

func formatHeaders(headers http.Header) string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&b, "  %s: %s\n", key, strings.Join(headers.Values(key), ", "))
	}
	return b.String()
}

func previewBody(body []byte) string {
	if len(body) == 0 {
		return "  <empty>"
	}

	preview := body
	if len(preview) > 8000 {
		preview = preview[:8000]
	}

	var pretty bytes.Buffer
	if json.Indent(&pretty, preview, "  ", "  ") == nil {
		preview = pretty.Bytes()
	}

	text := string(preview)
	if len(body) > len(preview) {
		text += "\n  ... truncated ..."
	}
	return text
}

func normalizeURL(rawURL string) string {
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}
	if strings.HasPrefix(rawURL, "localhost:") {
		return "http://" + rawURL
	}
	return "https://" + rawURL
}

func wrap(value, size int) int {
	if value < 0 {
		return size - 1
	}
	if value >= size {
		return 0
	}
	return value
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
