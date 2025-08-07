package ui

import (
	"github.com/HamStudy/kubewatch/internal/ui/views"
	tea "github.com/charmbracelet/bubbletea"
)

// applyContextSelectionForTest simulates context selection without requiring real k8s clients
func (a *App) applyContextSelectionForTest() tea.Cmd {
	if a.contextView == nil {
		return nil
	}
	newContexts := a.contextView.GetSelectedContexts()
	if len(newContexts) > 0 {
		a.activeContexts = newContexts
		a.state.SetCurrentContexts(newContexts)

		// For testing, we directly set the multi-context mode based on context count
		// without trying to create real k8s clients
		if len(newContexts) == 1 {
			a.isMultiContext = false
			// In a real scenario, we'd create a single client here
			// For testing, we just update the resource view
			a.resourceView = views.NewResourceView(a.state, nil)
			a.resourceView.SetSize(a.width, a.height)
		} else {
			a.isMultiContext = true
			// In a real scenario, we'd create a multi-client here
			// For testing, we just update the resource view
			a.resourceView = views.NewResourceViewWithMultiContext(a.state, nil)
			a.resourceView.SetSize(a.width, a.height)
		}

		// Return to list mode
		a.setMode(ModeList)
		// In tests, we don't actually refresh resources since there's no client
		return func() tea.Msg { return nil }
	}
	a.setMode(ModeList)
	return nil
}
