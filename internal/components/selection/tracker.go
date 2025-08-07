package selection

import (
	"strings"
	"sync"
)

// ResourceIdentity uniquely identifies a Kubernetes resource
type ResourceIdentity struct {
	Context   string // Kubernetes context (for multi-context mode)
	Namespace string // Kubernetes namespace
	Name      string // Resource name
	UID       string // Kubernetes UID (most unique identifier)
	Kind      string // Resource kind (Pod, Deployment, etc.)
}

// Tracker manages resource selection and persistence
type Tracker struct {
	selectedIdentity *ResourceIdentity
	resourceMap      map[int]*ResourceIdentity
	selectedRow      int
	mu               sync.RWMutex
}

// New creates a new selection tracker
func New() *Tracker {
	return &Tracker{
		resourceMap: make(map[int]*ResourceIdentity),
		selectedRow: 0,
	}
}

// SaveSelection stores the identity of the currently selected resource
func (t *Tracker) SaveSelection() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.selectedRow >= 0 && t.selectedRow < len(t.resourceMap) {
		if identity, exists := t.resourceMap[t.selectedRow]; exists {
			t.selectedIdentity = identity
		}
	}
}

// UpdateSelection updates the selected row and saves the identity
func (t *Tracker) UpdateSelection(row int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.selectedRow = row
	if row >= 0 && row < len(t.resourceMap) {
		if identity, exists := t.resourceMap[row]; exists {
			t.selectedIdentity = identity
		}
	}
}

// GetSelectedRow returns the currently selected row index
func (t *Tracker) GetSelectedRow() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.selectedRow
}

// GetSelectedIdentity returns the currently selected resource identity
func (t *Tracker) GetSelectedIdentity() *ResourceIdentity {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.selectedIdentity != nil {
		// Return a copy to prevent external modification
		return &ResourceIdentity{
			Context:   t.selectedIdentity.Context,
			Namespace: t.selectedIdentity.Namespace,
			Name:      t.selectedIdentity.Name,
			UID:       t.selectedIdentity.UID,
			Kind:      t.selectedIdentity.Kind,
		}
	}
	return nil
}

// SetResourceMap sets the mapping between row indices and resource identities
func (t *Tracker) SetResourceMap(resourceMap map[int]*ResourceIdentity) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.resourceMap = make(map[int]*ResourceIdentity)
	for k, v := range resourceMap {
		if v != nil {
			t.resourceMap[k] = &ResourceIdentity{
				Context:   v.Context,
				Namespace: v.Namespace,
				Name:      v.Name,
				UID:       v.UID,
				Kind:      v.Kind,
			}
		}
	}
}

// AddResource adds a resource identity at the specified row index
func (t *Tracker) AddResource(row int, identity *ResourceIdentity) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if identity != nil {
		t.resourceMap[row] = &ResourceIdentity{
			Context:   identity.Context,
			Namespace: identity.Namespace,
			Name:      identity.Name,
			UID:       identity.UID,
			Kind:      identity.Kind,
		}
	}
}

// RestoreSelection attempts to restore the previously selected resource
func (t *Tracker) RestoreSelection(totalRows int) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.selectedIdentity == nil {
		// No previous selection, keep current position if valid
		if t.selectedRow < totalRows {
			return t.selectedRow
		}
		if totalRows > 0 {
			t.selectedRow = totalRows - 1
			return t.selectedRow
		}
		t.selectedRow = 0
		return 0
	}

	// Save the previous selected row index for fallback
	previousRow := t.selectedRow

	// Try to find the resource by its identity (exact UID match)
	newIndex := t.findResourceByIdentity(t.selectedIdentity)
	if newIndex >= 0 {
		t.selectedRow = newIndex
		// Update selectedIdentity to match the found resource
		t.selectedIdentity = t.resourceMap[newIndex]
		return t.selectedRow
	}

	// If exact UID match not found, try to find by name and context (less precise)
	// This handles cases where a resource is recreated with the same name
	for rowIndex, identity := range t.resourceMap {
		if identity != nil &&
			identity.Name == t.selectedIdentity.Name &&
			identity.Context == t.selectedIdentity.Context &&
			identity.Namespace == t.selectedIdentity.Namespace {
			t.selectedRow = rowIndex
			// Update selectedIdentity to match the found resource
			t.selectedIdentity = identity
			return t.selectedRow
		}
	}

	// Resource not found by UID or name
	// Check if this looks like a complete refresh (all resources changed)
	allNewResources := t.checkIfAllResourcesNew()

	// If resource not found, handle intelligently
	if totalRows > 0 {
		if allNewResources {
			// All resources appear to be new, reset to top
			t.selectedRow = 0
		} else if previousRow < totalRows && previousRow >= 0 {
			// The previous index is still valid, stay there
			// This handles the case where a single resource is deleted
			t.selectedRow = previousRow
		} else {
			// Previous index out of bounds, select the last item
			t.selectedRow = totalRows - 1
		}

		// Update selectedIdentity to match new selection
		if identity, exists := t.resourceMap[t.selectedRow]; exists {
			t.selectedIdentity = identity
		} else {
			t.selectedIdentity = nil
		}
	} else {
		t.selectedRow = 0
		t.selectedIdentity = nil
	}

	return t.selectedRow
}

// findResourceByIdentity searches for a resource by its identity and returns the row index
func (t *Tracker) findResourceByIdentity(identity *ResourceIdentity) int {
	if identity == nil {
		return -1
	}

	for rowIndex, resourceIdentity := range t.resourceMap {
		if resourceIdentity != nil &&
			resourceIdentity.UID == identity.UID &&
			resourceIdentity.Context == identity.Context &&
			resourceIdentity.Namespace == identity.Namespace &&
			resourceIdentity.Name == identity.Name {
			return rowIndex
		}
	}
	return -1
}

// checkIfAllResourcesNew determines if all resources appear to be new
func (t *Tracker) checkIfAllResourcesNew() bool {
	if t.selectedIdentity == nil {
		return false
	}

	// Extract the base name pattern (e.g., "test-pod" from "test-pod-1")
	selectedBaseName := t.selectedIdentity.Name
	if idx := strings.LastIndex(selectedBaseName, "-"); idx > 0 {
		selectedBaseName = selectedBaseName[:idx]
	}

	// Check if any resource has a similar name pattern
	for _, identity := range t.resourceMap {
		if identity != nil && strings.HasPrefix(identity.Name, selectedBaseName) {
			return false
		}
	}

	return true
}

// Clear clears all selection data
func (t *Tracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.selectedIdentity = nil
	t.resourceMap = make(map[int]*ResourceIdentity)
	t.selectedRow = 0
}

// GetResourceCount returns the number of tracked resources
func (t *Tracker) GetResourceCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.resourceMap)
}

// GetResourceAt returns the resource identity at the specified row
func (t *Tracker) GetResourceAt(row int) *ResourceIdentity {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if identity, exists := t.resourceMap[row]; exists && identity != nil {
		// Return a copy to prevent external modification
		return &ResourceIdentity{
			Context:   identity.Context,
			Namespace: identity.Namespace,
			Name:      identity.Name,
			UID:       identity.UID,
			Kind:      identity.Kind,
		}
	}
	return nil
}

// HasSelection returns true if there is a current selection
func (t *Tracker) HasSelection() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.selectedIdentity != nil
}

// GetSelectedResourceName returns the name of the currently selected resource
func (t *Tracker) GetSelectedResourceName() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.selectedIdentity != nil {
		return t.selectedIdentity.Name
	}
	return ""
}

// GetSelectedResourceContext returns the context of the currently selected resource
func (t *Tracker) GetSelectedResourceContext() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.selectedIdentity != nil {
		return t.selectedIdentity.Context
	}
	return ""
}

// GetSelectedResourceNamespace returns the namespace of the currently selected resource
func (t *Tracker) GetSelectedResourceNamespace() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.selectedIdentity != nil {
		return t.selectedIdentity.Namespace
	}
	return ""
}

// SetSelectedRow sets the selected row without updating identity
func (t *Tracker) SetSelectedRow(row int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.selectedRow = row
}

// MoveSelection moves the selection by the given delta
func (t *Tracker) MoveSelection(delta int, totalRows int) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	if totalRows == 0 {
		t.selectedRow = 0
		return 0
	}

	newRow := t.selectedRow + delta
	if newRow < 0 {
		newRow = 0
	} else if newRow >= totalRows {
		newRow = totalRows - 1
	}

	t.selectedRow = newRow

	// Update selected identity
	if identity, exists := t.resourceMap[t.selectedRow]; exists {
		t.selectedIdentity = identity
	}

	return t.selectedRow
}

// IsResourceSelected returns true if the specified resource is currently selected
func (t *Tracker) IsResourceSelected(identity *ResourceIdentity) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.selectedIdentity == nil || identity == nil {
		return false
	}

	return t.selectedIdentity.UID == identity.UID &&
		t.selectedIdentity.Context == identity.Context &&
		t.selectedIdentity.Namespace == identity.Namespace &&
		t.selectedIdentity.Name == identity.Name
}

// GetResourceMap returns a copy of the current resource map
func (t *Tracker) GetResourceMap() map[int]*ResourceIdentity {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[int]*ResourceIdentity)
	for k, v := range t.resourceMap {
		if v != nil {
			result[k] = &ResourceIdentity{
				Context:   v.Context,
				Namespace: v.Namespace,
				Name:      v.Name,
				UID:       v.UID,
				Kind:      v.Kind,
			}
		}
	}
	return result
}
