package views

import (
	"encoding/base64"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDataViewConfigMapInitialization(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		BinaryData: map[string][]byte{
			"binary1": []byte("binary data"),
		},
	}

	view := NewConfigMapView(cm)

	if view == nil {
		t.Fatal("NewConfigMapView returned nil")
	}

	if view.resourceType != "ConfigMap" {
		t.Errorf("resourceType = %q, want %q", view.resourceType, "ConfigMap")
	}

	if view.resourceName != "test-cm" {
		t.Errorf("resourceName = %q, want %q", view.resourceName, "test-cm")
	}

	if view.namespace != "default" {
		t.Errorf("namespace = %q, want %q", view.namespace, "default")
	}

	if !view.decoded {
		t.Error("ConfigMaps should always be decoded")
	}

	if len(view.keys) != 3 {
		t.Errorf("keys count = %d, want 3", len(view.keys))
	}

	// Check keys are sorted
	expectedKeys := []string{"binary1 (binary)", "key1", "key2"}
	for i, key := range view.keys {
		if key != expectedKeys[i] {
			t.Errorf("keys[%d] = %q, want %q", i, key, expectedKeys[i])
		}
	}
}

func TestDataViewSecretInitialization(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "kube-system",
		},
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("secret123"),
		},
	}

	view := NewSecretView(secret)

	if view == nil {
		t.Fatal("NewSecretView returned nil")
	}

	if view.resourceType != "Secret" {
		t.Errorf("resourceType = %q, want %q", view.resourceType, "Secret")
	}

	if view.resourceName != "test-secret" {
		t.Errorf("resourceName = %q, want %q", view.resourceName, "test-secret")
	}

	if view.namespace != "kube-system" {
		t.Errorf("namespace = %q, want %q", view.namespace, "kube-system")
	}

	if view.decoded {
		t.Error("Secrets should start encoded")
	}

	if len(view.keys) != 2 {
		t.Errorf("keys count = %d, want 2", len(view.keys))
	}
}

func TestDataViewInit(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}
	view := NewConfigMapView(cm)

	cmd := view.Init()
	if cmd != nil {
		t.Error("Init should return nil command")
	}
}

func TestDataViewKeyNavigation(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	tests := []struct {
		name         string
		keys         []string
		wantSelected int
	}{
		{
			name:         "tab to next key",
			keys:         []string{"tab"},
			wantSelected: 1,
		},
		{
			name:         "j to next key",
			keys:         []string{"j"},
			wantSelected: 1,
		},
		{
			name:         "shift+tab to previous key",
			keys:         []string{"tab", "tab", "shift+tab"},
			wantSelected: 1,
		},
		{
			name:         "k to previous key",
			keys:         []string{"j", "j", "k"},
			wantSelected: 1,
		},
		{
			name:         "stay at last key",
			keys:         []string{"tab", "tab", "tab", "tab"},
			wantSelected: 2,
		},
		{
			name:         "stay at first key",
			keys:         []string{"k"},
			wantSelected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewConfigMapView(cm)
			view.ready = true

			for _, key := range tt.keys {
				var msg tea.KeyMsg
				switch key {
				case "tab":
					msg = tea.KeyMsg{Type: tea.KeyTab}
				case "shift+tab":
					msg = tea.KeyMsg{Type: tea.KeyShiftTab}
				default:
					msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
				}

				model, _ := view.Update(msg)
				view = model.(*DataView)
			}

			if view.selectedKey != tt.wantSelected {
				t.Errorf("selectedKey = %d, want %d", view.selectedKey, tt.wantSelected)
			}
		})
	}
}

func TestDataViewSecretDecoding(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Data: map[string][]byte{
			"password": []byte("secret123"),
		},
	}

	view := NewSecretView(secret)
	view.ready = true
	view.SetSize(80, 24)

	// Initially encoded
	if view.decoded {
		t.Error("should start encoded")
	}

	// Press 'd' to toggle decode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	model, _ := view.Update(msg)
	view = model.(*DataView)

	if !view.decoded {
		t.Error("should be decoded after pressing 'd'")
	}

	// Press 'd' again to encode
	model, _ = view.Update(msg)
	view = model.(*DataView)

	if view.decoded {
		t.Error("should be encoded after pressing 'd' again")
	}
}

func TestDataViewConfigMapNoDecoding(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Data:       map[string]string{"key": "value"},
	}

	view := NewConfigMapView(cm)
	view.ready = true

	// ConfigMap should always be decoded
	if !view.decoded {
		t.Error("ConfigMap should always be decoded")
	}

	// Press 'd' should not change decode state for ConfigMap
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	model, _ := view.Update(msg)
	view = model.(*DataView)

	if !view.decoded {
		t.Error("ConfigMap decode state should not change")
	}
}

func TestDataViewScrolling(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Data: map[string]string{
			"key": strings.Repeat("Long content\n", 100),
		},
	}

	view := NewConfigMapView(cm)
	view.ready = true
	view.SetSize(80, 24)

	tests := []struct {
		name    string
		key     string
		keyType tea.KeyType
	}{
		{"home with g", "g", 0},
		{"home with Home key", "", tea.KeyHome},
		{"end with G", "G", 0},
		{"end with End key", "", tea.KeyEnd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg tea.KeyMsg
			if tt.keyType != 0 {
				msg = tea.KeyMsg{Type: tt.keyType}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			model, _ := view.Update(msg)
			view = model.(*DataView)

			// Should handle scrolling without crashing
		})
	}
}

func TestDataViewExit(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}

	view := NewConfigMapView(cm)

	exitKeys := []string{"esc", "q"}

	for _, key := range exitKeys {
		t.Run("exit with "+key, func(t *testing.T) {
			var msg tea.KeyMsg
			if key == "esc" {
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
			}

			model, _ := view.Update(msg)
			if model != view {
				t.Error("should return same view for parent to handle exit")
			}
		})
	}
}

func TestDataViewWindowResize(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}

	view := NewConfigMapView(cm)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	model, _ := view.Update(msg)
	view = model.(*DataView)

	if view.width != 100 || view.height != 50 {
		t.Errorf("size = (%d, %d), want (100, 50)", view.width, view.height)
	}

	if !view.ready {
		t.Error("should be ready after window size message")
	}

	if view.viewport.Width != 100 {
		t.Errorf("viewport width = %d, want 100", view.viewport.Width)
	}

	if view.viewport.Height != 46 { // 50 - 4 for header and footer
		t.Errorf("viewport height = %d, want 46", view.viewport.Height)
	}
}

func TestDataViewSetSize(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}

	view := NewConfigMapView(cm)
	view.SetSize(120, 40)

	if view.width != 120 || view.height != 40 {
		t.Errorf("size = (%d, %d), want (120, 40)", view.width, view.height)
	}

	if !view.ready {
		t.Error("should be ready after SetSize")
	}
}

func TestDataViewRendering(t *testing.T) {
	tests := []struct {
		name         string
		isSecret     bool
		decoded      bool
		data         map[string]string
		wantContains []string
	}{
		{
			name:     "renders ConfigMap header",
			isSecret: false,
			data:     map[string]string{"key": "value"},
			wantContains: []string{
				"ConfigMap:",
				"test/default",
			},
		},
		{
			name:     "renders Secret header with encoded state",
			isSecret: true,
			decoded:  false,
			data:     map[string]string{"key": "value"},
			wantContains: []string{
				"Secret:",
				"[ENCODED]",
			},
		},
		{
			name:     "renders Secret header with decoded state",
			isSecret: true,
			decoded:  true,
			data:     map[string]string{"key": "value"},
			wantContains: []string{
				"Secret:",
				"[DECODED]",
			},
		},
		{
			name:     "shows key info",
			isSecret: false,
			data:     map[string]string{"mykey": "value"},
			wantContains: []string{
				"Key 1/1: mykey",
			},
		},
		{
			name:     "shows controls for ConfigMap",
			isSecret: false,
			data:     map[string]string{"key": "value"},
			wantContains: []string{
				"Navigate keys",
				"Scroll",
				"Top/Bottom",
				"Close",
			},
		},
		{
			name:     "shows decode control for Secret",
			isSecret: true,
			data:     map[string]string{"key": "value"},
			wantContains: []string{
				"Toggle decode",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var view *DataView

			if tt.isSecret {
				secret := &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
					Data: make(map[string][]byte),
				}
				for k, v := range tt.data {
					secret.Data[k] = []byte(v)
				}
				view = NewSecretView(secret)
				view.decoded = tt.decoded
			} else {
				cm := &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
					Data: tt.data,
				}
				view = NewConfigMapView(cm)
			}

			view.ready = true
			view.SetSize(80, 24)

			output := view.View()

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output does not contain %q", want)
				}
			}
		})
	}
}

func TestDataViewNotReady(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}

	view := NewConfigMapView(cm)
	view.ready = false

	output := view.View()

	if output != "Loading..." {
		t.Errorf("not ready view = %q, want %q", output, "Loading...")
	}
}

func TestDataViewContentDisplay(t *testing.T) {
	tests := []struct {
		name        string
		isSecret    bool
		decoded     bool
		key         string
		value       string
		isBinary    bool
		wantContent string
	}{
		{
			name:        "shows ConfigMap value as-is",
			isSecret:    false,
			key:         "config",
			value:       "plain text value",
			wantContent: "plain text value",
		},
		{
			name:        "shows Secret encoded",
			isSecret:    true,
			decoded:     false,
			key:         "password",
			value:       "secret123",
			wantContent: base64.StdEncoding.EncodeToString([]byte("secret123")),
		},
		{
			name:        "shows Secret decoded",
			isSecret:    true,
			decoded:     true,
			key:         "password",
			value:       base64.StdEncoding.EncodeToString([]byte("secret123")),
			wantContent: "secret123",
		},
		{
			name:        "handles decode error",
			isSecret:    true,
			decoded:     true,
			key:         "invalid",
			value:       "not-base64!@#",
			wantContent: "Error decoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var view *DataView

			if tt.isSecret {
				secret := &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
					Data:       map[string][]byte{tt.key: []byte(tt.value)},
				}
				view = NewSecretView(secret)
				view.decoded = tt.decoded
			} else {
				cm := &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
					Data:       map[string]string{tt.key: tt.value},
				}
				view = NewConfigMapView(cm)
			}

			view.ready = true
			view.selectedKey = 0
			view.SetSize(80, 24)
			view.updateContent()

			// Check the rendered output instead
			output := view.View()
			if !strings.Contains(output, tt.wantContent) {
				t.Errorf("output does not contain %q", tt.wantContent)
			}
		})
	}
}

func TestDataViewEmptyData(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "empty"},
		Data:       map[string]string{},
	}

	view := NewConfigMapView(cm)
	view.ready = true
	view.SetSize(80, 24)
	view.updateContent()

	// Check the rendered output
	output := view.View()
	if !strings.Contains(output, "No data") {
		t.Error("should show 'No data' for empty ConfigMap")
	}
}

func TestDataViewBinaryData(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		BinaryData: map[string][]byte{
			"binary": {0x00, 0x01, 0x02, 0x03},
		},
	}

	view := NewConfigMapView(cm)

	// Should have binary key marked
	if len(view.keys) != 1 || view.keys[0] != "binary (binary)" {
		t.Errorf("binary key = %q, want %q", view.keys[0], "binary (binary)")
	}

	view.ready = true
	view.selectedKey = 0
	view.SetSize(80, 24)
	view.updateContent()

	// Binary data should be base64 encoded for display
	output := view.View()
	expectedBase64 := base64.StdEncoding.EncodeToString([]byte{0x00, 0x01, 0x02, 0x03})
	if !strings.Contains(output, expectedBase64) {
		t.Error("binary data should be displayed as base64")
	}
}

func TestDataViewEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *DataView
		action   func(*DataView)
		validate func(*testing.T, *DataView)
	}{
		{
			name: "handles very long key names",
			setup: func() *DataView {
				cm := &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
					Data: map[string]string{
						strings.Repeat("very-long-key-name-", 10): "value",
					},
				}
				return NewConfigMapView(cm)
			},
			action: func(v *DataView) {
				v.SetSize(80, 24)
			},
			validate: func(t *testing.T, v *DataView) {
				output := v.View()
				if output == "" {
					t.Error("should render with long key names")
				}
			},
		},
		{
			name: "handles selectedKey out of bounds",
			setup: func() *DataView {
				cm := &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
					Data:       map[string]string{"key": "value"},
				}
				view := NewConfigMapView(cm)
				view.selectedKey = 10 // Out of bounds
				return view
			},
			action: func(v *DataView) {
				v.updateContent()
			},
			validate: func(t *testing.T, v *DataView) {
				if v.selectedKey != 0 {
					t.Error("should reset selectedKey when out of bounds")
				}
			},
		},
		{
			name: "handles unknown key messages",
			setup: func() *DataView {
				cm := &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
				}
				return NewConfigMapView(cm)
			},
			action: func(v *DataView) {
				msg := tea.KeyMsg{Type: tea.KeyF1}
				model, _ := v.Update(msg)
				*v = *model.(*DataView)
			},
			validate: func(t *testing.T, v *DataView) {
				// Should not crash
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := tt.setup()
			view.ready = true
			tt.action(view)
			tt.validate(t, view)
		})
	}
}
