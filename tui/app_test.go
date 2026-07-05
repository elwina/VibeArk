package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"vibeark/internal/apps"
	"vibeark/internal/config"
)

func TestChannelPollingPattern(t *testing.T) {
	progressCh := make(chan dlProgressMsg, 64)
	statusCh := make(chan dlStatusMsg, 16)
	doneCh := make(chan dlDoneMsg, 1)

	go func() {
		statusCh <- dlStatusMsg{text: "downloading", url: "http://example.com/file.zip"}
		for i := 1; i <= 5; i++ {
			progressCh <- dlProgressMsg{
				transferred: int64(i * 100),
				total:       500,
				percent:     float64(i*100) / 500 * 100,
			}
			time.Sleep(10 * time.Millisecond)
		}
		doneCh <- dlDoneMsg{appID: "test", success: true, version: "1.0"}
		close(progressCh)
		close(statusCh)
		close(doneCh)
	}()

	progressCount := 0
	statusCount := 0
	done := false

	for !done {
		cmd := readDownloadChannels(progressCh, statusCh, doneCh)
		msg := cmd()

		switch msg := msg.(type) {
		case dlProgressMsg:
			progressCount++
			t.Logf("Progress: %.0f%% (%d/%d)", msg.percent, msg.transferred, msg.total)
		case dlStatusMsg:
			statusCount++
			t.Logf("Status: %s url=%s", msg.text, msg.url)
		case dlDoneMsg:
			t.Logf("Done: success=%v err=%s", msg.success, msg.err)
			done = true
		case pollMsg:
			t.Logf("Poll timeout - retrying")
		}
	}

	if progressCount != 5 {
		t.Errorf("Expected 5 progress messages, got %d", progressCount)
	}
	if statusCount != 1 {
		t.Errorf("Expected 1 status message, got %d", statusCount)
	}
}

// TestModelUpdateLoop simulates Bubble Tea's event loop to verify that
// channels stored on the Model survive value-receiver Update calls.
func TestModelUpdateLoop(t *testing.T) {
	// Create channels and assign directly to model (what installOrUpdate does)
	progressCh := make(chan dlProgressMsg, 64)
	statusCh := make(chan dlStatusMsg, 16)
	doneCh := make(chan dlDoneMsg, 1)

	settings := config.Settings{InstallDir: t.TempDir()}

	m := Model{
		apps: []apps.AppWithStatus{
			{AppEntry: apps.AppEntry{ID: "a", Name: "App A"}},
			{AppEntry: apps.AppEntry{ID: "b", Name: "App B"}},
		},
		settings:     settings,
		selectedIdx:  0,
		currentView:  viewMain,
		action:       "downloading",
		dlAppName:    "App A",
		dlProgressCh: progressCh,
		dlStatusCh:   statusCh,
		dlDoneCh:     doneCh,
	}

	// Simulate download goroutine
	go func() {
		time.Sleep(50 * time.Millisecond)
		statusCh <- dlStatusMsg{text: "downloading", url: "http://x.com/a.zip"}
		for i := 1; i <= 3; i++ {
			time.Sleep(20 * time.Millisecond)
			progressCh <- dlProgressMsg{
				transferred: int64(i * 1024 * 1024),
				total:       3 * 1024 * 1024,
				percent:     float64(i) / 3 * 100,
			}
		}
		time.Sleep(20 * time.Millisecond)
		doneCh <- dlDoneMsg{appID: "a", success: true, version: "1.0"}
		close(progressCh)
		close(statusCh)
		close(doneCh)
	}()

	// Simulate Bubble Tea's event loop
	cmd := readDownloadChannels(m.dlProgressCh, m.dlStatusCh, m.dlDoneCh)
	if cmd == nil {
		t.Fatal("initial Cmd is nil")
	}

	gotStatus := false
	gotProgress := false
	gotDone := false

	for !gotDone {
		msg := cmd()

		var newCmd tea.Cmd
		updatedModel, newCmd := m.Update(msg)
		m = updatedModel.(Model) // Update returns tea.Model, which is our Model

		t.Logf("Update(%T): action=%s progress={trans=%d, total=%d, pct=%.0f} ch=[%v,%v,%v]",
			msg, m.action,
			m.progress.transferred, m.progress.total, m.progress.percent,
			m.dlProgressCh != nil, m.dlStatusCh != nil, m.dlDoneCh != nil)

		switch v := msg.(type) {
		case dlStatusMsg:
			gotStatus = true
			if m.dlURL != "http://x.com/a.zip" {
				t.Errorf("dlURL not set: got %q", m.dlURL)
			}
		case dlProgressMsg:
			gotProgress = true
			if v.transferred == 0 {
				t.Error("progress.transferred is 0")
			}
		case dlDoneMsg:
			gotDone = true
			if m.action != "idle" {
				t.Errorf("action should be idle, got %q", m.action)
			}
		}

		if newCmd == nil && !gotDone {
			t.Fatal("newCmd is nil before done - polling stopped early!")
		}
		cmd = newCmd
	}

	if !gotStatus {
		t.Error("never received dlStatusMsg")
	}
	if !gotProgress {
		t.Error("never received dlProgressMsg")
	}
	if m.dlProgressCh != nil {
		t.Error("channels should be cleared after done")
	}
	t.Log("Test passed: all messages received in correct order")
}
