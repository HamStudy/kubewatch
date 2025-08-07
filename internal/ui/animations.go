package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AnimationManager handles smooth transitions and animations
type AnimationManager struct {
	enabled    bool
	duration   time.Duration
	easing     EasingFunction
	animations map[string]*Animation
	frameRate  time.Duration
}

// Animation represents an ongoing animation
type Animation struct {
	ID         string
	StartTime  time.Time
	Duration   time.Duration
	Easing     EasingFunction
	From       interface{}
	To         interface{}
	Current    interface{}
	OnUpdate   func(interface{})
	OnComplete func()
	Completed  bool
}

// EasingFunction defines animation easing
type EasingFunction func(t float64) float64

// AnimationMsg represents an animation frame update
type AnimationMsg struct {
	ID    string
	Value interface{}
	Done  bool
}

// NewAnimationManager creates a new animation manager
func NewAnimationManager() *AnimationManager {
	return &AnimationManager{
		enabled:    true,
		duration:   200 * time.Millisecond,
		easing:     EaseOutCubic,
		animations: make(map[string]*Animation),
		frameRate:  16 * time.Millisecond, // ~60fps
	}
}

// Easing functions
func Linear(t float64) float64 {
	return t
}

func EaseInQuad(t float64) float64 {
	return t * t
}

func EaseOutQuad(t float64) float64 {
	return t * (2 - t)
}

func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

func EaseInCubic(t float64) float64 {
	return t * t * t
}

func EaseOutCubic(t float64) float64 {
	t--
	return t*t*t + 1
}

func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	t = 2*t - 2
	return 1 + t*t*t
}

func EaseInBounce(t float64) float64 {
	return 1 - EaseOutBounce(1-t)
}

func EaseOutBounce(t float64) float64 {
	if t < 1/2.75 {
		return 7.5625 * t * t
	} else if t < 2/2.75 {
		t -= 1.5 / 2.75
		return 7.5625*t*t + 0.75
	} else if t < 2.5/2.75 {
		t -= 2.25 / 2.75
		return 7.5625*t*t + 0.9375
	} else {
		t -= 2.625 / 2.75
		return 7.5625*t*t + 0.984375
	}
}

// Enable enables or disables animations
func (am *AnimationManager) Enable(enabled bool) {
	am.enabled = enabled
	if !enabled {
		// Complete all animations immediately
		for _, anim := range am.animations {
			if anim.OnUpdate != nil {
				anim.OnUpdate(anim.To)
			}
			if anim.OnComplete != nil {
				anim.OnComplete()
			}
		}
		am.animations = make(map[string]*Animation)
	}
}

// SetDuration sets the default animation duration
func (am *AnimationManager) SetDuration(duration time.Duration) {
	am.duration = duration
}

// SetEasing sets the default easing function
func (am *AnimationManager) SetEasing(easing EasingFunction) {
	am.easing = easing
}

// Animate starts a new animation
func (am *AnimationManager) Animate(id string, from, to interface{}, onUpdate func(interface{}), onComplete func()) tea.Cmd {
	if !am.enabled {
		// Execute immediately if animations are disabled
		if onUpdate != nil {
			onUpdate(to)
		}
		if onComplete != nil {
			onComplete()
		}
		return nil
	}

	animation := &Animation{
		ID:         id,
		StartTime:  time.Now(),
		Duration:   am.duration,
		Easing:     am.easing,
		From:       from,
		To:         to,
		Current:    from,
		OnUpdate:   onUpdate,
		OnComplete: onComplete,
	}

	am.animations[id] = animation

	return am.animationTick(id)
}

// AnimateWithOptions starts an animation with custom options
func (am *AnimationManager) AnimateWithOptions(id string, from, to interface{}, duration time.Duration, easing EasingFunction, onUpdate func(interface{}), onComplete func()) tea.Cmd {
	if !am.enabled {
		if onUpdate != nil {
			onUpdate(to)
		}
		if onComplete != nil {
			onComplete()
		}
		return nil
	}

	animation := &Animation{
		ID:         id,
		StartTime:  time.Now(),
		Duration:   duration,
		Easing:     easing,
		From:       from,
		To:         to,
		Current:    from,
		OnUpdate:   onUpdate,
		OnComplete: onComplete,
	}

	am.animations[id] = animation

	return am.animationTick(id)
}

// animationTick creates a command for the next animation frame
func (am *AnimationManager) animationTick(id string) tea.Cmd {
	return tea.Tick(am.frameRate, func(t time.Time) tea.Msg {
		return AnimationMsg{ID: id}
	})
}

// Update processes animation messages
func (am *AnimationManager) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case AnimationMsg:
		return am.updateAnimation(msg.ID)
	}
	return nil
}

// updateAnimation updates a specific animation
func (am *AnimationManager) updateAnimation(id string) tea.Cmd {
	animation, exists := am.animations[id]
	if !exists || animation.Completed {
		return nil
	}

	now := time.Now()
	elapsed := now.Sub(animation.StartTime)

	if elapsed >= animation.Duration {
		// Animation complete
		animation.Current = animation.To
		animation.Completed = true

		if animation.OnUpdate != nil {
			animation.OnUpdate(animation.Current)
		}
		if animation.OnComplete != nil {
			animation.OnComplete()
		}

		delete(am.animations, id)
		return nil
	}

	// Calculate progress
	progress := float64(elapsed) / float64(animation.Duration)
	easedProgress := animation.Easing(progress)

	// Interpolate value
	animation.Current = am.interpolate(animation.From, animation.To, easedProgress)

	if animation.OnUpdate != nil {
		animation.OnUpdate(animation.Current)
	}

	// Schedule next frame
	return am.animationTick(id)
}

// interpolate interpolates between two values
func (am *AnimationManager) interpolate(from, to interface{}, progress float64) interface{} {
	switch f := from.(type) {
	case int:
		if t, ok := to.(int); ok {
			return int(float64(f) + float64(t-f)*progress)
		}
	case float64:
		if t, ok := to.(float64); ok {
			return f + (t-f)*progress
		}
	case lipgloss.Color:
		if t, ok := to.(lipgloss.Color); ok {
			return am.interpolateColor(f, t, progress)
		}
	case string:
		// For strings, we can do character-by-character transitions
		if t, ok := to.(string); ok {
			return am.interpolateString(f, t, progress)
		}
	}

	// Default: return target value when progress >= 0.5
	if progress >= 0.5 {
		return to
	}
	return from
}

// interpolateColor interpolates between two colors
func (am *AnimationManager) interpolateColor(from, to lipgloss.Color, progress float64) lipgloss.Color {
	// Simple color interpolation (could be enhanced with proper color space conversion)
	// For now, just switch at 50% progress
	if progress >= 0.5 {
		return to
	}
	return from
}

// interpolateString interpolates between two strings
func (am *AnimationManager) interpolateString(from, to string, progress float64) string {
	if progress >= 1.0 {
		return to
	}
	if progress <= 0.0 {
		return from
	}

	// Character-by-character transition
	maxLen := len(to)
	if len(from) > maxLen {
		maxLen = len(from)
	}

	targetLen := int(float64(maxLen) * progress)
	if targetLen > len(to) {
		targetLen = len(to)
	}

	return to[:targetLen]
}

// StopAnimation stops an animation
func (am *AnimationManager) StopAnimation(id string) {
	delete(am.animations, id)
}

// StopAllAnimations stops all animations
func (am *AnimationManager) StopAllAnimations() {
	am.animations = make(map[string]*Animation)
}

// IsAnimating returns whether an animation is running
func (am *AnimationManager) IsAnimating(id string) bool {
	_, exists := am.animations[id]
	return exists
}

// GetActiveAnimations returns the IDs of all active animations
func (am *AnimationManager) GetActiveAnimations() []string {
	ids := make([]string, 0, len(am.animations))
	for id := range am.animations {
		ids = append(ids, id)
	}
	return ids
}

// TransitionManager handles view transitions
type TransitionManager struct {
	animationManager *AnimationManager
	currentView      string
	targetView       string
	transitioning    bool
	transitionType   TransitionType
}

// TransitionType defines the type of transition
type TransitionType int

const (
	TransitionFade TransitionType = iota
	TransitionSlideLeft
	TransitionSlideRight
	TransitionSlideUp
	TransitionSlideDown
	TransitionZoom
)

// NewTransitionManager creates a new transition manager
func NewTransitionManager(animationManager *AnimationManager) *TransitionManager {
	return &TransitionManager{
		animationManager: animationManager,
		transitionType:   TransitionFade,
	}
}

// TransitionTo transitions to a new view
func (tm *TransitionManager) TransitionTo(viewID string, transitionType TransitionType, onComplete func()) tea.Cmd {
	if tm.transitioning {
		return nil // Already transitioning
	}

	tm.targetView = viewID
	tm.transitionType = transitionType
	tm.transitioning = true

	// Start transition animation
	return tm.animationManager.Animate(
		"view-transition",
		0.0,
		1.0,
		func(value interface{}) {
			// Update transition progress
		},
		func() {
			tm.currentView = tm.targetView
			tm.transitioning = false
			if onComplete != nil {
				onComplete()
			}
		},
	)
}

// IsTransitioning returns whether a transition is in progress
func (tm *TransitionManager) IsTransitioning() bool {
	return tm.transitioning
}

// GetCurrentView returns the current view ID
func (tm *TransitionManager) GetCurrentView() string {
	return tm.currentView
}

// LoadingIndicator provides animated loading states
type LoadingIndicator struct {
	frames    []string
	current   int
	message   string
	style     lipgloss.Style
	spinning  bool
	lastFrame time.Time
	interval  time.Duration
}

// NewLoadingIndicator creates a new loading indicator
func NewLoadingIndicator() *LoadingIndicator {
	return &LoadingIndicator{
		frames: []string{
			"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		},
		interval: 100 * time.Millisecond,
		style:    lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
	}
}

// SetFrames sets custom loading frames
func (li *LoadingIndicator) SetFrames(frames []string) {
	li.frames = frames
	li.current = 0
}

// SetMessage sets the loading message
func (li *LoadingIndicator) SetMessage(message string) {
	li.message = message
}

// SetStyle sets the loading indicator style
func (li *LoadingIndicator) SetStyle(style lipgloss.Style) {
	li.style = style
}

// Start starts the loading animation
func (li *LoadingIndicator) Start() {
	li.spinning = true
	li.lastFrame = time.Now()
}

// Stop stops the loading animation
func (li *LoadingIndicator) Stop() {
	li.spinning = false
}

// Update updates the loading indicator
func (li *LoadingIndicator) Update() tea.Cmd {
	if !li.spinning {
		return nil
	}

	now := time.Now()
	if now.Sub(li.lastFrame) >= li.interval {
		li.current = (li.current + 1) % len(li.frames)
		li.lastFrame = now
	}

	return tea.Tick(li.interval, func(t time.Time) tea.Msg {
		return LoadingTickMsg{}
	})
}

// LoadingTickMsg represents a loading animation tick
type LoadingTickMsg struct{}

// View renders the loading indicator
func (li *LoadingIndicator) View() string {
	if !li.spinning {
		return ""
	}

	frame := li.frames[li.current]
	if li.message != "" {
		return li.style.Render(fmt.Sprintf("%s %s", frame, li.message))
	}
	return li.style.Render(frame)
}

// ProgressBar provides animated progress indication
type ProgressBar struct {
	width     int
	progress  float64
	style     lipgloss.Style
	fillStyle lipgloss.Style
	showText  bool
	text      string
}

// NewProgressBar creates a new progress bar
func NewProgressBar(width int) *ProgressBar {
	return &ProgressBar{
		width:     width,
		style:     lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		fillStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
		showText:  true,
	}
}

// SetProgress sets the progress value (0.0 to 1.0)
func (pb *ProgressBar) SetProgress(progress float64) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	pb.progress = progress
}

// SetText sets custom progress text
func (pb *ProgressBar) SetText(text string) {
	pb.text = text
}

// SetShowText enables or disables progress text
func (pb *ProgressBar) SetShowText(show bool) {
	pb.showText = show
}

// View renders the progress bar
func (pb *ProgressBar) View() string {
	filled := int(float64(pb.width) * pb.progress)
	empty := pb.width - filled

	styledBar := pb.fillStyle.Render(strings.Repeat("█", filled)) +
		pb.style.Render(strings.Repeat("░", empty))

	if pb.showText {
		text := pb.text
		if text == "" {
			text = fmt.Sprintf("%.0f%%", pb.progress*100)
		}
		return fmt.Sprintf("%s %s", styledBar, text)
	}

	return styledBar
}

// NotificationManager handles user notifications
type NotificationManager struct {
	notifications []*Notification
	maxVisible    int
	position      NotificationPosition
	style         lipgloss.Style
}

// Notification represents a user notification
type Notification struct {
	ID       string
	Type     NotificationType
	Title    string
	Message  string
	Duration time.Duration
	ShowTime time.Time
	Style    lipgloss.Style
}

// NotificationType defines notification types
type NotificationType int

const (
	NotificationInfo NotificationType = iota
	NotificationSuccess
	NotificationWarning
	NotificationError
)

// NotificationPosition defines where notifications appear
type NotificationPosition int

const (
	NotificationTopRight NotificationPosition = iota
	NotificationTopLeft
	NotificationBottomRight
	NotificationBottomLeft
	NotificationCenter
)

// NewNotificationManager creates a new notification manager
func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		notifications: make([]*Notification, 0),
		maxVisible:    5,
		position:      NotificationTopRight,
		style:         lipgloss.NewStyle().Padding(1).Margin(1),
	}
}

// Show displays a notification
func (nm *NotificationManager) Show(notificationType NotificationType, title, message string, duration time.Duration) {
	notification := &Notification{
		ID:       fmt.Sprintf("notification-%d", time.Now().UnixNano()),
		Type:     notificationType,
		Title:    title,
		Message:  message,
		Duration: duration,
		ShowTime: time.Now(),
		Style:    nm.getStyleForType(notificationType),
	}

	nm.notifications = append(nm.notifications, notification)

	// Remove old notifications if we exceed the limit
	if len(nm.notifications) > nm.maxVisible {
		nm.notifications = nm.notifications[len(nm.notifications)-nm.maxVisible:]
	}

	// Auto-remove after duration
	if duration > 0 {
		go func() {
			time.Sleep(duration)
			nm.Remove(notification.ID)
		}()
	}
}

// Remove removes a notification by ID
func (nm *NotificationManager) Remove(id string) {
	for i, notification := range nm.notifications {
		if notification.ID == id {
			nm.notifications = append(nm.notifications[:i], nm.notifications[i+1:]...)
			break
		}
	}
}

// Clear removes all notifications
func (nm *NotificationManager) Clear() {
	nm.notifications = make([]*Notification, 0)
}

// getStyleForType returns the style for a notification type
func (nm *NotificationManager) getStyleForType(notificationType NotificationType) lipgloss.Style {
	base := nm.style.Copy()

	switch notificationType {
	case NotificationSuccess:
		return base.BorderForeground(lipgloss.Color("46")).Foreground(lipgloss.Color("46"))
	case NotificationWarning:
		return base.BorderForeground(lipgloss.Color("226")).Foreground(lipgloss.Color("226"))
	case NotificationError:
		return base.BorderForeground(lipgloss.Color("196")).Foreground(lipgloss.Color("196"))
	default: // Info
		return base.BorderForeground(lipgloss.Color("39")).Foreground(lipgloss.Color("39"))
	}
}

// View renders all visible notifications
func (nm *NotificationManager) View() string {
	if len(nm.notifications) == 0 {
		return ""
	}

	var lines []string
	for _, notification := range nm.notifications {
		content := notification.Title
		if notification.Message != "" {
			content += "\n" + notification.Message
		}
		lines = append(lines, notification.Style.Render(content))
	}

	return strings.Join(lines, "\n")
}
