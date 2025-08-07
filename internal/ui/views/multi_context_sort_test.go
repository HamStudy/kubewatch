package views

import (
	"testing"
	"time"

	"github.com/HamStudy/kubewatch/internal/core"
	"github.com/HamStudy/kubewatch/internal/k8s"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMultiContextSortOrderConsistency(t *testing.T) {
	// Test that sort order remains consistent across refreshes in multi-context mode
	t.Run("pods maintain consistent sort order across refreshes", func(t *testing.T) {
		state := &core.State{
			CurrentResourceType: core.ResourceTypePod,
			CurrentNamespace:    "default",
			SortColumn:          "CONTEXT", // Sort by context
			SortAscending:       true,
		}

		view := NewResourceView(state, nil)
		view.isMultiContext = true
		view.showContextColumn = true

		// Create pods with different contexts
		// First refresh - pods come in one order
		podsRefresh1 := []k8s.PodWithContext{
			{
				Context: "context-b",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: "default",
						UID:       "uid-1",
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
			{
				Context: "context-a",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-2",
						Namespace: "default",
						UID:       "uid-2",
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
			{
				Context: "context-c",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-3",
						Namespace: "default",
						UID:       "uid-3",
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
		}

		// First update
		view.updateTableWithPodsMultiContext(podsRefresh1)

		// Capture the order after first refresh
		firstOrder := make([]string, len(view.rows))
		for i, row := range view.rows {
			firstOrder[i] = row[0] + ":" + row[1] // context:name
		}

		// Verify alphabetical order by context
		assert.Equal(t, "context-a:pod-2", firstOrder[0])
		assert.Equal(t, "context-b:pod-1", firstOrder[1])
		assert.Equal(t, "context-c:pod-3", firstOrder[2])

		// Select the middle item
		view.selectedRow = 1
		view.saveSelectedIdentity()
		selectedIdentity := view.selectedIdentity
		assert.NotNil(t, selectedIdentity)
		assert.Equal(t, "context-b", selectedIdentity.Context)
		assert.Equal(t, "pod-1", selectedIdentity.Name)

		// Second refresh - pods come in different order
		podsRefresh2 := []k8s.PodWithContext{
			{
				Context: "context-c",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-3",
						Namespace: "default",
						UID:       "uid-3",
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
			{
				Context: "context-a",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-2",
						Namespace: "default",
						UID:       "uid-2",
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
			{
				Context: "context-b",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: "default",
						UID:       "uid-1",
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
		}

		// Second update
		view.updateTableWithPodsMultiContext(podsRefresh2)

		// Capture the order after second refresh
		secondOrder := make([]string, len(view.rows))
		for i, row := range view.rows {
			secondOrder[i] = row[0] + ":" + row[1] // context:name
		}

		// Order should be the same regardless of input order
		assert.Equal(t, firstOrder, secondOrder, "Sort order should be consistent across refreshes")

		// Selection should be maintained
		assert.Equal(t, 1, view.selectedRow, "Selection should remain on the same item")
		assert.Equal(t, "context-b", view.rows[view.selectedRow][0])
		assert.Equal(t, "pod-1", view.rows[view.selectedRow][1])
	})

	t.Run("sort by name column maintains consistency", func(t *testing.T) {
		state := &core.State{
			CurrentResourceType: core.ResourceTypePod,
			CurrentNamespace:    "default",
			SortColumn:          "NAME", // Sort by name
			SortAscending:       true,
		}

		view := NewResourceView(state, nil)
		view.isMultiContext = true
		view.showContextColumn = true

		// Create pods with same names but different contexts
		podsRefresh1 := []k8s.PodWithContext{
			{
				Context: "context-b",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "alpha-pod",
						Namespace: "default",
						UID:       "uid-1",
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
			{
				Context: "context-a",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "beta-pod",
						Namespace: "default",
						UID:       "uid-2",
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
			{
				Context: "context-c",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "alpha-pod", // Same name as first, different context
						Namespace: "default",
						UID:       "uid-3",
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
		}

		// First update
		view.updateTableWithPodsMultiContext(podsRefresh1)

		// When sorting by name, capture the initial order for consistency testing
		// The exact order may vary based on implementation, but should be consistent
		initialOrder := make([]string, len(view.rows))
		for i, row := range view.rows {
			initialOrder[i] = row[0] + ":" + row[1] // context:name
		}

		// Select the first alpha-pod
		view.selectedRow = 0
		view.saveSelectedIdentity()

		// Shuffle the input order
		podsRefresh2 := []k8s.PodWithContext{
			podsRefresh1[2], // context-c alpha-pod
			podsRefresh1[0], // context-b alpha-pod
			podsRefresh1[1], // context-a beta-pod
		}

		// Second update
		view.updateTableWithPodsMultiContext(podsRefresh2)

		// Capture the order after second refresh
		secondOrder := make([]string, len(view.rows))
		for i, row := range view.rows {
			secondOrder[i] = row[0] + ":" + row[1] // context:name
		}

		// Order should remain consistent across refreshes
		assert.Equal(t, initialOrder, secondOrder, "Sort order should be consistent across refreshes")

		// Selection should be maintained
		assert.Equal(t, 0, view.selectedRow)
	})

	t.Run("adding and removing contexts maintains sort order", func(t *testing.T) {
		state := &core.State{
			CurrentResourceType: core.ResourceTypePod,
			CurrentNamespace:    "default",
			SortColumn:          "CONTEXT", // Sort by context
			SortAscending:       true,
		}

		view := NewResourceView(state, nil)
		view.isMultiContext = true
		view.showContextColumn = true

		// Start with 2 contexts
		podsRefresh1 := []k8s.PodWithContext{
			{
				Context: "context-b",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-b",
						Namespace: "default",
						UID:       "uid-b",
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
			{
				Context: "context-a",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-a",
						Namespace: "default",
						UID:       "uid-a",
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
		}

		view.updateTableWithPodsMultiContext(podsRefresh1)
		assert.Equal(t, "context-a", view.rows[0][0])
		assert.Equal(t, "context-b", view.rows[1][0])

		// Select context-b pod
		view.selectedRow = 1
		view.saveSelectedIdentity()

		// Add a new context in the middle (alphabetically)
		podsRefresh2 := []k8s.PodWithContext{
			podsRefresh1[0], // context-b
			{
				Context: "context-ab", // New context between a and b
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "pod-ab",
						Namespace:         "default",
						UID:               "uid-ab",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
			podsRefresh1[1], // context-a
		}

		view.updateTableWithPodsMultiContext(podsRefresh2)

		// Verify order is maintained alphabetically
		assert.Equal(t, "context-a", view.rows[0][0])
		assert.Equal(t, "context-ab", view.rows[1][0])
		assert.Equal(t, "context-b", view.rows[2][0])

		// Selection should move with the item
		assert.Equal(t, 2, view.selectedRow, "Selection should follow context-b to its new position")
		assert.Equal(t, "context-b", view.rows[view.selectedRow][0])
	})
}

func TestMultiContextSortStability(t *testing.T) {
	// Test that when primary sort values are equal, secondary sort is stable
	t.Run("stable sort when primary column values are equal", func(t *testing.T) {
		state := &core.State{
			CurrentResourceType: core.ResourceTypePod,
			CurrentNamespace:    "default",
			SortColumn:          "STATUS", // Sort by STATUS column
			SortAscending:       true,
		}

		view := NewResourceView(state, nil)
		view.isMultiContext = true
		view.showContextColumn = true

		// Create pods with same status but different contexts/names
		now := time.Now()
		pods := []k8s.PodWithContext{
			{
				Context: "ctx-2",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "pod-b",
						Namespace:         "default",
						UID:               "uid-2",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
			{
				Context: "ctx-1",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "pod-a",
						Namespace:         "default",
						UID:               "uid-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
			{
				Context: "ctx-3",
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "pod-c",
						Namespace:         "default",
						UID:               "uid-3",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Status: v1.PodStatus{Phase: v1.PodRunning},
				},
			},
		}

		// Do multiple updates with different input orders
		for i := 0; i < 5; i++ {
			// Shuffle the input order each time
			shuffled := make([]k8s.PodWithContext, len(pods))
			copy(shuffled, pods)
			// Simple rotation for testing
			if i%2 == 0 {
				shuffled[0], shuffled[1], shuffled[2] = pods[2], pods[0], pods[1]
			} else {
				shuffled[0], shuffled[1], shuffled[2] = pods[1], pods[2], pods[0]
			}

			view.updateTableWithPodsMultiContext(shuffled)

			// When all have same status, should fall back to context then name
			// So order should always be: ctx-1/pod-a, ctx-2/pod-b, ctx-3/pod-c
			assert.Equal(t, "ctx-1", view.rows[0][0], "First row context in iteration %d", i)
			assert.Equal(t, "pod-a", view.rows[0][1], "First row name in iteration %d", i)
			assert.Equal(t, "ctx-2", view.rows[1][0], "Second row context in iteration %d", i)
			assert.Equal(t, "pod-b", view.rows[1][1], "Second row name in iteration %d", i)
			assert.Equal(t, "ctx-3", view.rows[2][0], "Third row context in iteration %d", i)
			assert.Equal(t, "pod-c", view.rows[2][1], "Third row name in iteration %d", i)
		}
	})
}
