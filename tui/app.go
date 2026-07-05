package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"vibeark/internal/apps"
	"vibeark/internal/config"
	"vibeark/internal/downloader"
	"vibeark/internal/i18n"
	"vibeark/internal/scanner"
	"vibeark/internal/updater"
	"vibeark/internal/version"
)

type view string

const (
	viewMain     view = "main"
	viewSettings view = "settings"
)

// Message types
type checkDoneMsg struct {
	appID         string
	remoteVersion string
	err           string
}

type dlProgressMsg struct {
	transferred int64
	total       int64
	percent     float64
}

type dlStatusMsg struct {
	text string
	url  string
}

type dlDoneMsg struct {
	appID      string
	success    bool
	exePath    string
	version    string
	err        string
	cancelled  bool
}

type pollMsg struct{}

type checkAllProgressMsg struct {
	appName       string
	appID         string
	remoteVersion string
	done          int
	total         int
}

type checkAllDoneMsg struct {
	toast toastMsg
}

type toastMsg struct {
	text  string
	color string
}

type listItem struct {
	isHeader bool
	label    string
	appIndex int // index into Model.apps, valid only when !isHeader
}

// Model
type Model struct {
	apps           []apps.AppWithStatus
	displayList    []listItem
	settings       config.Settings
	lock           *config.LockFile
	selectedIdx    int
	currentView    view
	proxyEnabled   bool
	ghProxyEnabled bool

	action     string // "idle" | "checking" | "downloading" | "extracting"
	dlAppName  string
	dlURL      string
	progress   dlProgressMsg
	errorMsg   string
	message    string
	msgColor   string

	// Download channels (references survive struct copies)
	dlProgressCh chan dlProgressMsg
	dlStatusCh   chan dlStatusMsg
	dlDoneCh     chan dlDoneMsg

	// CheckAll channels
	checkAllProgressCh chan checkAllProgressMsg
	checkAllDoneCh     chan toastMsg

	debugLog *os.File

	settingsIdx  int
	editingField string
	editValue    string

	keys keyMap
}

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Check    key.Binding
	CheckAll key.Binding
	GhProxy  key.Binding
	Launch   key.Binding
	Delete   key.Binding
	Proxy    key.Binding
	Refresh  key.Binding
	Cancel   key.Binding
	Settings key.Binding
	Lang     key.Binding
	Quit     key.Binding
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case checkDoneMsg:
		m.action = "idle"
		if msg.err != "" {
			m.errorMsg = msg.err
		} else {
			for i := range m.apps {
				if m.apps[i].ID == msg.appID {
					m.apps[i].RemoteVersion = msg.remoteVersion
					break
				}
			}
			m.showToast(i18n.T("check_done"), "green")
		}
		return m, nil

	case checkAllProgressMsg:
		m.action = "checking"
		m.dlAppName = fmt.Sprintf("%s (%d/%d)", msg.appName, msg.done+1, msg.total)
		if msg.remoteVersion != "" {
			for i := range m.apps {
				if m.apps[i].ID == msg.appID {
					m.apps[i].RemoteVersion = msg.remoteVersion
					break
				}
			}
		}
		return m, checkAllPoll(m.checkAllProgressCh, m.checkAllDoneCh)

	case checkAllDoneMsg:
		m.action = "idle"
		m.dlAppName = ""
		m.checkAllProgressCh = nil
		m.checkAllDoneCh = nil
		if msg.toast.text != "" {
			m.showToast(msg.toast.text, msg.toast.color)
		}
		return m, nil

	case dlProgressMsg:
		m.progress = msg
		m.dbg("Update: dlProgressMsg pct=%.0f%% trans=%d total=%d", msg.percent, msg.transferred, msg.total)
		return m, readDownloadChannels(m.dlProgressCh, m.dlStatusCh, m.dlDoneCh)

	case dlStatusMsg:
		if msg.url != "" {
			m.dlURL = msg.url
		}
		if msg.text == "extracting" {
			m.action = "extracting"
		}
		m.dbg("Update: dlStatusMsg text=%s url=%s", msg.text, msg.url)
		return m, readDownloadChannels(m.dlProgressCh, m.dlStatusCh, m.dlDoneCh)

	case dlDoneMsg:
		m.dbg("Update: dlDoneMsg success=%v cancelled=%v err=%s", msg.success, msg.cancelled, msg.err)
		m.action = "idle"
		m.dlAppName = ""
		m.dlURL = ""
		m.progress = dlProgressMsg{}
		m.dlProgressCh = nil
		m.dlStatusCh = nil
		m.dlDoneCh = nil
		if msg.success {
			m.refreshApps()
			m.showToast(i18n.T("install_success"), "green")
		} else if msg.cancelled {
			m.showToast(i18n.T("download_cancelled"), "yellow")
		} else {
			m.errorMsg = msg.err
		}
		return m, nil

	case toastMsg:
		m.message = msg.text
		m.msgColor = msg.color
		return m, nil

	case pollMsg:
		if m.checkAllProgressCh != nil {
			return m, checkAllPoll(m.checkAllProgressCh, m.checkAllDoneCh)
		}
		if m.dlProgressCh != nil {
			return m, readDownloadChannels(m.dlProgressCh, m.dlStatusCh, m.dlDoneCh)
		}
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.currentView == viewSettings {
		return m.viewSettings()
	}
	return m.viewMain()
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.currentView == viewSettings {
		return m.handleSettingsKey(msg)
	}

	if m.errorMsg != "" {
		m.errorMsg = ""
		return m, nil
	}

	if m.message != "" {
		m.message = ""
		return m, nil
	}

	if key.Matches(msg, m.keys.Cancel) {
		if m.action != "idle" {
			m.action = "idle"
			m.dlAppName = ""
			m.dlURL = ""
			m.progress = dlProgressMsg{}
			return m, nil
		}
	}

	if key.Matches(msg, m.keys.Up) {
		if m.selectedIdx > 0 {
			m.selectedIdx--
			for m.selectedIdx > 0 && m.displayList[m.selectedIdx].isHeader {
				m.selectedIdx--
			}
		}
		return m, nil
	}
	if key.Matches(msg, m.keys.Down) {
		if m.selectedIdx < len(m.displayList)-1 {
			m.selectedIdx++
			for m.selectedIdx < len(m.displayList)-1 && m.displayList[m.selectedIdx].isHeader {
				m.selectedIdx++
			}
		}
		return m, nil
	}

	if m.action != "idle" {
		return m, nil
	}

	if key.Matches(msg, m.keys.Quit) {
		return m, tea.Quit
	}
	if key.Matches(msg, m.keys.Check) {
		m.action = "checking"
		return m, m.checkOne()
	}
	if key.Matches(msg, m.keys.CheckAll) {
		m.action = "checking"
		return m, m.checkAll()
	}
	if key.Matches(msg, m.keys.Enter) {
		app, ok := m.selectedApp()
		if !ok {
			return m, nil
		}
		if app.Status == "installed" {
			m.showToast(i18n.T("already_latest"), "cyan")
			return m, nil
		}
		m.action = "downloading"
		m.dlAppName = app.Name
		m.dbg("handleKey: Enter pressed for %s", app.ID)
		cmd := m.installOrUpdate()
		return m, cmd
	}
	if key.Matches(msg, m.keys.Launch) {
		m.launchApp()
		return m, nil
	}
	if key.Matches(msg, m.keys.Delete) {
		m.deleteApp()
		return m, nil
	}
	if key.Matches(msg, m.keys.Proxy) {
		m.proxyEnabled = !m.proxyEnabled
		if m.proxyEnabled {
			m.showToast(i18n.T("proxy_on"), "cyan")
		} else {
			m.showToast(i18n.T("proxy_off"), "cyan")
		}
		return m, nil
	}
	if key.Matches(msg, m.keys.GhProxy) {
		m.ghProxyEnabled = !m.ghProxyEnabled
		if m.ghProxyEnabled {
			m.showToast(i18n.T("ghproxy_on"), "cyan")
		} else {
			m.showToast(i18n.T("ghproxy_off"), "cyan")
		}
		return m, nil
	}
	if key.Matches(msg, m.keys.Refresh) {
		m.refreshApps()
		m.showToast(i18n.T("refreshed"), "cyan")
		return m, nil
	}
	if key.Matches(msg, m.keys.Settings) {
		m.currentView = viewSettings
		m.settingsIdx = 0
		return m, nil
	}
	if key.Matches(msg, m.keys.Lang) {
		newLang := i18n.ToggleLang()
		m.settings.Language = string(newLang)
		config.SaveSettings(m.settings)
		m.showToast(string(newLang), "cyan")
		return m, nil
	}

	return m, nil
}

func (m Model) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editingField != "" {
		if msg.String() == "enter" {
			if m.editingField == "installDir" {
				m.settings.InstallDir = m.editValue
			} else if m.editingField == "proxy" {
				m.settings.Proxy = config.NormalizeProxy(m.editValue)
			} else if m.editingField == "ghProxy" {
				m.settings.GhProxy = m.editValue
			}
			config.SaveSettings(m.settings)
			m.editingField = ""
			m.refreshApps()
			m.showToast(i18n.T("settings_saved"), "green")
			return m, nil
		}
		if msg.String() == "esc" {
			m.editingField = ""
			return m, nil
		}
		if msg.String() == "backspace" {
			if len(m.editValue) > 0 {
				m.editValue = m.editValue[:len(m.editValue)-1]
			}
			return m, nil
		}
		if len(msg.String()) == 1 {
			m.editValue += msg.String()
			return m, nil
		}
		return m, nil
	}

	if key.Matches(msg, m.keys.Up) && m.settingsIdx > 0 {
		m.settingsIdx--
		return m, nil
	}
	if key.Matches(msg, m.keys.Down) && m.settingsIdx < 2 {
		m.settingsIdx++
		return m, nil
	}
	if msg.String() == "enter" {
		if m.settingsIdx == 0 {
			m.editingField = "installDir"
			m.editValue = m.settings.InstallDir
		} else if m.settingsIdx == 1 {
			m.editingField = "proxy"
			m.editValue = m.settings.Proxy
		} else {
			m.editingField = "ghProxy"
			m.editValue = m.settings.GhProxy
		}
		return m, nil
	}
	if msg.String() == "q" || msg.String() == "esc" {
		m.currentView = viewMain
		return m, nil
	}

	return m, nil
}

func (m *Model) showToast(text, color string) {
	m.message = text
	m.msgColor = color
}

func (m *Model) refreshApps() {
	appDefs := apps.AppDefinitions
	scanned := scanner.ScanAll(appDefs, m.settings.InstallDir)
	lock := config.LoadLock(m.settings.InstallDir)

	m.apps = make([]apps.AppWithStatus, len(appDefs))
	for i, app := range appDefs {
		m.apps[i] = apps.AppWithStatus{
			AppEntry: app,
			Status:   scanned[i].Status,
		}
		if entry, ok := lock.Entries[app.ID]; ok && entry.InstalledVersion != "" {
			m.apps[i].LocalVersion = entry.InstalledVersion
		}
	}
	m.lock = lock

	m.buildDisplayList()
}

func (m *Model) buildDisplayList() {
	m.displayList = nil

	type category struct {
		label string
		apps  []int
	}

	categories := []category{
		{label: i18n.T("category_official")},
		{label: i18n.T("category_mirror")},
		{label: i18n.T("category_github")},
	}

	for i, app := range m.apps {
		cat := sourceCategory(app)
		switch cat {
		case "mirror":
			categories[1].apps = append(categories[1].apps, i)
		case "github":
			categories[2].apps = append(categories[2].apps, i)
		default:
			categories[0].apps = append(categories[0].apps, i)
		}
	}

	for _, cat := range categories {
		if len(cat.apps) == 0 {
			continue
		}
		m.displayList = append(m.displayList, listItem{isHeader: true, label: cat.label})
		for _, idx := range cat.apps {
			m.displayList = append(m.displayList, listItem{appIndex: idx})
		}
	}

	if m.selectedIdx >= len(m.displayList) {
		m.selectedIdx = 0
	}
	for m.selectedIdx < len(m.displayList) && m.displayList[m.selectedIdx].isHeader {
		m.selectedIdx++
	}
}

func sourceCategory(app apps.AppWithStatus) string {
	if strings.Contains(app.Download.URL, "mirror") || strings.Contains(app.Source.URL, "mirror") {
		return "mirror"
	}
	if strings.Contains(app.Download.URL, "github") {
		return "github"
	}
	return "official"
}

func (m Model) selectedApp() (apps.AppWithStatus, bool) {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.displayList) {
		return apps.AppWithStatus{}, false
	}
	item := m.displayList[m.selectedIdx]
	if item.isHeader || item.appIndex < 0 || item.appIndex >= len(m.apps) {
		return apps.AppWithStatus{}, false
	}
	return m.apps[item.appIndex], true
}

func (m Model) checkOne() tea.Cmd {
	return func() tea.Msg {
		app, ok := m.selectedApp()
		if !ok {
			return checkDoneMsg{err: i18n.T("no_app_selected")}
		}

		result := updater.CheckVersion(app.AppEntry, m.getEffectiveProxy())

		if result.Error != nil {
			return checkDoneMsg{appID: app.ID, err: result.Error.Error()}
		}

		lock := config.LoadLock(m.settings.InstallDir)
		config.UpdateLockEntry(lock, app.ID, config.LockEntry{LastChecked: config.NowUTC()})
		if entry, ok := lock.Entries[app.ID]; ok && entry.Status == "installed" && entry.InstalledVersion == "" {
			config.UpdateLockEntry(lock, app.ID, config.LockEntry{InstalledVersion: result.RemoteVersion})
		}
		config.SaveLock(m.settings.InstallDir, lock)

		return checkDoneMsg{appID: app.ID, remoteVersion: result.RemoteVersion}
	}
}

func (m *Model) checkAll() tea.Cmd {
	total := 0
	for _, item := range m.displayList {
		if !item.isHeader {
			total++
		}
	}
	m.checkAllProgressCh = make(chan checkAllProgressMsg, total+1)
	m.checkAllDoneCh = make(chan toastMsg, 1)

	go func() {
		lock := config.LoadLock(m.settings.InstallDir)
		done := 0
		for _, item := range m.displayList {
			if item.isHeader {
				continue
			}
			app := m.apps[item.appIndex]
			result := updater.CheckVersion(app.AppEntry, m.getEffectiveProxy())
			m.checkAllProgressCh <- checkAllProgressMsg{
				appName:       app.Name,
				appID:         app.ID,
				remoteVersion: result.RemoteVersion,
				done:          done,
				total:         total,
			}
			done++
			if result.RemoteVersion != "" {
				config.UpdateLockEntry(lock, app.ID, config.LockEntry{LastChecked: config.NowUTC()})
				if entry, ok := lock.Entries[app.ID]; ok && entry.Status == "installed" && entry.InstalledVersion == "" {
					config.UpdateLockEntry(lock, app.ID, config.LockEntry{InstalledVersion: result.RemoteVersion})
				}
			}
		}
		config.SaveLock(m.settings.InstallDir, lock)
		m.checkAllDoneCh <- toastMsg{text: i18n.T("check_all_done"), color: "green"}
		close(m.checkAllProgressCh)
		close(m.checkAllDoneCh)
	}()

	return checkAllPoll(m.checkAllProgressCh, m.checkAllDoneCh)
}

func (m *Model) dbg(format string, args ...interface{}) {
	if m.debugLog == nil {
		return
	}
	msg := fmt.Sprintf(time.Now().Format("15:04:05.000")+" "+format+"\n", args...)
	m.debugLog.WriteString(msg)
}

// readDownloadChannels returns a Cmd that reads from download channels.
// Uses non-blocking reads: returns data immediately if available,
// returns pollMsg if nothing is ready (so Update re-queues immediately).
func readDownloadChannels(progressCh chan dlProgressMsg, statusCh chan dlStatusMsg, doneCh chan dlDoneMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case p, ok := <-progressCh:
			if !ok {
				select {
				case d, dok := <-doneCh:
					if !dok {
						return dlDoneMsg{}
					}
					return d
				default:
					return dlDoneMsg{}
				}
			}
			return p
		case s, ok := <-statusCh:
			if !ok {
				select {
				case d, dok := <-doneCh:
					if !dok {
						return dlDoneMsg{}
					}
					return d
				default:
					return dlDoneMsg{}
				}
			}
			return s
		case d, ok := <-doneCh:
			if !ok {
				return dlDoneMsg{}
			}
			return d
		default:
			time.Sleep(50 * time.Millisecond)
			return pollMsg{}
		}
	}
}

func checkAllPoll(progressCh chan checkAllProgressMsg, doneCh chan toastMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case p, ok := <-progressCh:
			if !ok {
				select {
				case t, tok := <-doneCh:
					if !tok {
						return checkAllDoneMsg{}
					}
					return checkAllDoneMsg{toast: t}
				default:
					return checkAllDoneMsg{}
				}
			}
			return p
		case t, ok := <-doneCh:
			if !ok {
				return checkAllDoneMsg{}
			}
			return checkAllDoneMsg{toast: t}
		default:
			time.Sleep(50 * time.Millisecond)
			return pollMsg{}
		}
	}
}

// installOrUpdate starts download and returns a Cmd that chains progress updates
func (m *Model) installOrUpdate() tea.Cmd {
	app, ok := m.selectedApp()
	if !ok {
		return nil
	}

	// Open debug log
	if m.debugLog == nil {
		f, err := os.Create(filepath.Join(os.TempDir(), "vibeark_debug.log"))
		if err == nil {
			m.debugLog = f
		}
	}
	m.dbg("installOrUpdate: app=%s version=%s", app.ID, app.RemoteVersion)

	effectiveSettings := config.Settings{
		InstallDir: m.settings.InstallDir,
		Proxy:      m.getEffectiveProxy(),
	}
	if m.ghProxyEnabled {
		effectiveSettings.GhProxy = m.settings.GhProxy
	}

	// Create channels for progress reporting
	m.dlProgressCh = make(chan dlProgressMsg, 64)
	m.dlStatusCh = make(chan dlStatusMsg, 16)
	m.dlDoneCh = make(chan dlDoneMsg, 1)

	m.dbg("installOrUpdate: channels created, starting goroutine")

	// Run download in background goroutine
	go func() {
		m.dbg("goroutine: starting DownloadAndInstall")
		result := downloader.DownloadAndInstall(
			app.AppEntry,
			m.settings.InstallDir,
			app.RemoteVersion,
			&effectiveSettings,
			func(transferred, total int64, percent float64) {
				m.dbg("goroutine: progress %.0f%% (%d/%d)", percent, transferred, total)
				m.dlProgressCh <- dlProgressMsg{transferred, total, percent}
			},
			func(status string, url string) {
				m.dbg("goroutine: status=%s url=%s", status, url)
				m.dlStatusCh <- dlStatusMsg{text: status, url: url}
			},
		)

		m.dbg("goroutine: DownloadAndInstall done. success=%v err=%s", result.Success, result.Error)

		lock := config.LoadLock(m.settings.InstallDir)
		if result.Success {
			config.UpdateLockEntry(lock, app.ID, config.LockEntry{
				Status:           "installed",
				InstalledVersion: result.Version,
				LastDownloaded:   config.NowUTC(),
				InstalledAt:      config.NowUTC(),
				ExePath:          result.ExePath,
			})
			config.SaveLock(m.settings.InstallDir, lock)
			m.dlDoneCh <- dlDoneMsg{appID: app.ID, success: true, exePath: result.ExePath, version: result.Version}
		} else if result.Cancelled {
			m.dlDoneCh <- dlDoneMsg{appID: app.ID, cancelled: true}
		} else {
			config.UpdateLockEntry(lock, app.ID, config.LockEntry{Status: "error", Error: result.Error})
			config.SaveLock(m.settings.InstallDir, lock)
			m.dlDoneCh <- dlDoneMsg{appID: app.ID, err: result.Error}
		}

		m.dbg("goroutine: closing channels")
		close(m.dlProgressCh)
		close(m.dlStatusCh)
		close(m.dlDoneCh)
		m.dbg("goroutine: done")
	}()

	m.dbg("installOrUpdate: returning readDownloadChannels cmd")
	return readDownloadChannels(m.dlProgressCh, m.dlStatusCh, m.dlDoneCh)
}

func (m *Model) launchApp() {
	app, ok := m.selectedApp()
	if !ok {
		return
	}
	if app.Status == "not-found" {
		return
	}

	lock := config.LoadLock(m.settings.InstallDir)
	exePath := ""
	if entry, ok := lock.Entries[app.ID]; ok && entry.ExePath != "" {
		if _, err := os.Stat(entry.ExePath); err == nil {
			exePath = entry.ExePath
		}
	}

	if exePath == "" {
		appDir := filepath.Join(m.settings.InstallDir, app.ID)
		found := findExeSimple(appDir, app.ExePath)
		if found != "" {
			exePath = found
		}
	}

	if exePath == "" {
		m.errorMsg = i18n.T("exe_not_found")
		return
	}

	c := exec.Command(exePath)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}
	c.Start()
	m.showToast(i18n.T("launched"), "green")
}

func (m *Model) deleteApp() {
	app, ok := m.selectedApp()
	if !ok {
		return
	}
	if app.Status == "not-found" {
		return
	}

	appDir := filepath.Join(m.settings.InstallDir, app.ID)
	os.RemoveAll(appDir)

	lock := config.LoadLock(m.settings.InstallDir)
	config.UpdateLockEntry(lock, app.ID, config.LockEntry{
		Status: "not-found", InstalledVersion: "", ExePath: "",
	})
	config.SaveLock(m.settings.InstallDir, lock)

	m.refreshApps()
	m.showToast(i18n.T("deleted"), "green")
}

func (m *Model) getEffectiveProxy() string {
	if !m.proxyEnabled {
		return ""
	}
	return m.settings.Proxy
}

func findExeSimple(dir, pattern string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			matched, _ := filepath.Match(pattern, entry.Name())
			if matched {
				return filepath.Join(dir, entry.Name())
			}
		}
	}
	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(dir, entry.Name())
			subEntries, err := os.ReadDir(subDir)
			if err != nil {
				continue
			}
			for _, subEntry := range subEntries {
				if !subEntry.IsDir() {
					matched, _ := filepath.Match(pattern, subEntry.Name())
					if matched {
						return filepath.Join(subDir, subEntry.Name())
					}
				}
			}
		}
	}
	return ""
}

// Styles
var (
	titleStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	selectedStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	statusInstalled    = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	statusNotFound     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	versionStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	remoteVerStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	helpStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	warnStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	devStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Italic(true)
	sectionHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	descStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	infoStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
)

func formatBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%dB", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(b)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(b)/1024/1024)
}

func (m Model) viewMain() string {
	var s strings.Builder

	// Header
	s.WriteString(titleStyle.Render(fmt.Sprintf("VibeArk v%s", version.Value)) + " " + i18n.T("title") + "\n")
	s.WriteString(devStyle.Render("  ✦ by Elwina Vardal") + "\n\n")

	// App list
	s.WriteString("┌" + strings.Repeat("─", 52) + "┐\n")
	for i, item := range m.displayList {
		if item.isHeader {
			s.WriteString("│ " + sectionHeaderStyle.Render(item.label) + "\n")
			continue
		}
		app := m.apps[item.appIndex]

		cursor := "  "
		if i == m.selectedIdx {
			cursor = selectedStyle.Render("❯ ")
		}

		var icon string
		switch app.Status {
		case "installed":
			icon = statusInstalled.Render("✓")
		case "not-found":
			icon = statusNotFound.Render("○")
		default:
			icon = statusNotFound.Render("?")
		}

		name := app.Name
		if len(name) < 20 {
			name += strings.Repeat(" ", 20-len(name))
		}
		if i == m.selectedIdx {
			name = selectedStyle.Render(name)
		}

		ver := app.LocalVersion
		if ver == "" {
			ver = "---"
		}
		if len(ver) < 15 {
			ver += strings.Repeat(" ", 15-len(ver))
		}

		remote := ""
		if app.RemoteVersion != "" {
			remote = " → " + remoteVerStyle.Render(app.RemoteVersion)
		}

		notInst := ""
		if app.Status == "not-found" {
			notInst = versionStyle.Render(i18n.T("not_installed"))
		}

		s.WriteString(fmt.Sprintf("│ %s%s %s%s%s%s\n", cursor, icon, name, versionStyle.Render(ver), remote, notInst))
	}
	s.WriteString("└" + strings.Repeat("─", 52) + "┘\n\n")

	// Status area
	if m.action == "checking" {
		if m.dlAppName != "" {
			s.WriteString(warnStyle.Render(i18n.T("checking_prefix") + m.dlAppName) + "\n")
		} else {
			s.WriteString(warnStyle.Render(i18n.T("checking_dots")) + "\n")
		}
	} else if m.action == "downloading" || m.action == "extracting" {
		if m.dlURL != "" {
			s.WriteString(helpStyle.Render("URL: "+m.dlURL) + "\n")
		}
		if m.action == "downloading" {
			if m.progress.total > 0 {
				pct := fmt.Sprintf("%.0f%%", m.progress.percent)
				downloaded := formatBytes(m.progress.transferred)
				total := formatBytes(m.progress.total)
				s.WriteString(warnStyle.Render(fmt.Sprintf("↓ %s  %s (%s/%s)", m.dlAppName, pct, downloaded, total)) + "\n")
			} else if m.progress.transferred > 0 {
				s.WriteString(warnStyle.Render(fmt.Sprintf("↓ %s  %s", m.dlAppName, formatBytes(m.progress.transferred))) + "\n")
			} else {
				s.WriteString(warnStyle.Render(fmt.Sprintf("↓ %s  %s", m.dlAppName, i18n.T("waiting"))) + "\n")
			}
		} else {
			s.WriteString(warnStyle.Render(i18n.T("extracting")) + "\n")
		}
	} else if m.errorMsg != "" {
		s.WriteString(errorStyle.Render("✗ "+m.errorMsg) + i18n.T("any_key_close") + "\n")
	} else if m.message != "" {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(m.msgColor)).Render("✓ "+m.message) + "\n")
	} else {
		if app, ok := m.selectedApp(); ok {
			desc := app.Description
			if i18n.CurrentLang() == i18n.EN && app.DescriptionEn != "" {
				desc = app.DescriptionEn
			}
			if desc != "" {
				s.WriteString(descStyle.Render(app.Name+": "+desc) + "\n")
			} else {
				s.WriteString("\n")
			}
		} else {
			s.WriteString("\n")
		}
	}

	// Help
	s.WriteString(helpStyle.Render(i18n.T("help_line1")) + "\n")
	s.WriteString(helpStyle.Render(i18n.T("help_line2")) + "\n")
	s.WriteString(helpStyle.Render(i18n.T("help_line3")) + "\n")

	// Info
	proxy := i18n.T("proxy_not_set")
	if m.settings.Proxy != "" {
		if m.proxyEnabled {
			proxy = m.settings.Proxy
		} else {
			proxy = i18n.T("proxy_disabled")
		}
	}
	ghProxy := i18n.T("proxy_not_set")
	if m.settings.GhProxy != "" {
		if m.ghProxyEnabled {
			ghProxy = i18n.T("proxy_enabled")
		} else {
			ghProxy = i18n.T("proxy_disabled")
		}
	}
	s.WriteString(infoStyle.Render(fmt.Sprintf(i18n.T("info_status"), m.settings.InstallDir, proxy, ghProxy)) + "\n")

	return s.String()
}

func (m Model) viewSettings() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render(i18n.T("settings_title")) + "\n\n")

	fields := []struct {
		label string
		value string
	}{
		{i18n.T("settings_install_dir"), m.settings.InstallDir},
		{i18n.T("settings_proxy_port"), m.settings.Proxy},
		{i18n.T("settings_gh_proxy"), m.settings.GhProxy},
	}

	for i, field := range fields {
		cursor := "  "
		if i == m.settingsIdx {
			cursor = selectedStyle.Render("❯ ")
		}
		value := field.value
		if m.editingField == "installDir" && i == 0 {
			value = warnStyle.Render(m.editValue + "▌")
		} else if m.editingField == "proxy" && i == 1 {
			value = warnStyle.Render(m.editValue + "▌")
		} else if m.editingField == "ghProxy" && i == 2 {
			value = warnStyle.Render(m.editValue + "▌")
		}
		if value == "" {
			value = i18n.T("settings_none")
		}
		s.WriteString(fmt.Sprintf("%s%s: %s\n", cursor, lipgloss.NewStyle().Bold(true).Render(field.label), value))
	}

	s.WriteString("\n")
	if m.editingField != "" {
		s.WriteString(helpStyle.Render(i18n.T("settings_enter_confirm")) + "\n")
	} else {
		s.WriteString(helpStyle.Render(i18n.T("settings_nav")) + "\n")
	}
	s.WriteString(helpStyle.Render(i18n.T("settings_proxy_hint")) + "\n")
	s.WriteString(helpStyle.Render(i18n.T("settings_gh_hint")) + "\n")
	return s.String()
}

func NewModel() Model {
	settings := config.LoadSettings()

	if settings.Language == "" {
		settings.Language = string(i18n.DetectLang())
		config.SaveSettings(settings)
	}
	i18n.SetLang(i18n.Lang(settings.Language))

	m := Model{
		settings:       settings,
		currentView:    viewMain,
		action:         "idle",
		ghProxyEnabled: settings.GhProxy != "",
		keys: keyMap{
			Up:       key.NewBinding(key.WithKeys("up", "w")),
			Down:     key.NewBinding(key.WithKeys("down", "s")),
			Enter:    key.NewBinding(key.WithKeys("enter", "d")),
			Check:    key.NewBinding(key.WithKeys("a", "u")),
			CheckAll: key.NewBinding(key.WithKeys("ctrl+u")),
			Launch:   key.NewBinding(key.WithKeys("o")),
			Delete:   key.NewBinding(key.WithKeys("x")),
			Proxy:    key.NewBinding(key.WithKeys("p")),
			GhProxy:  key.NewBinding(key.WithKeys("g")),
			Refresh:  key.NewBinding(key.WithKeys("r")),
			Cancel:   key.NewBinding(key.WithKeys("esc")),
			Settings: key.NewBinding(key.WithKeys("ctrl+p")),
			Lang:     key.NewBinding(key.WithKeys("ctrl+l")),
			Quit:     key.NewBinding(key.WithKeys("q")),
		},
	}
	m.refreshApps()
	return m
}

func Run() {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
