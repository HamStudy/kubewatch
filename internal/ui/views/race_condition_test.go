package views

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/HamStudy/kubewatch/internal/core"
	"github.com/HamStudy/kubewatch/internal/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TestRaceConditionFix verifies that the race condition in ResourceView has been fixed
func TestRaceConditionFix(t *testing.T) {
	// Create a mock state
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		SortColumn:          "NAME",
		SortAscending:       true,
	}

	// Create a mock k8s client
	client := &k8s.Client{}

	// Create ResourceView
	rv := NewResourceView(state, client)

	// Create test pods
	pods := []v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
				UID:       types.UID("uid1"),
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: "default",
				UID:       types.UID("uid2"),
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod3",
				Namespace: "default",
				UID:       types.UID("uid3"),
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
			},
		},
	}

	// Test concurrent access to resourceMap
	const numGoroutines = 20
	const numIterations = 50

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	// Start multiple goroutines that concurrently update the table
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				func() {
					defer func() {
						if r := recover(); r != nil {
							errors <- fmt.Errorf("update goroutine %d iteration %d panicked: %v", goroutineID, j, r)
						}
					}()

					// This should be safe now with mutex protection
					rv.updateTableWithPods(pods)
				}()
			}
		}(i)
	}

	// Start goroutines that read from resourceMap
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				func() {
					defer func() {
						if r := recover(); r != nil {
							errors <- fmt.Errorf("reader goroutine %d iteration %d panicked: %v", goroutineID, j, r)
						}
					}()

					// This should be safe now with mutex protection
					rv.SetSelectedRow(j % 3) // Cycle through rows
				}()
			}
		}(i)
	}

	// Start goroutines that trigger sorting
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				func() {
					defer func() {
						if r := recover(); r != nil {
							errors <- fmt.Errorf("sort goroutine %d iteration %d panicked: %v", goroutineID, j, r)
						}
					}()

					// Change sort order to trigger sorting
					sortColumn, _ := state.GetSortState()
					if j%2 == 0 {
						state.SetSortState(sortColumn, true)
					} else {
						state.SetSortState(sortColumn, false)
					}

					rv.updateTableWithPods(pods)
				}()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		// Success
	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out - possible deadlock")
	}

	// Check for errors
	close(errors)
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	if errorCount == 0 {
		t.Logf("Race condition test completed successfully with %d goroutines and %d iterations each", numGoroutines, numIterations)
	} else {
		t.Fatalf("Race condition test failed with %d errors", errorCount)
	}
}

// TestConcurrentResourceMapAccess tests concurrent access to resourceMap specifically
func TestConcurrentResourceMapAccess(t *testing.T) {
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		SortColumn:          "NAME",
		SortAscending:       true,
	}

	client := &k8s.Client{}
	rv := NewResourceView(state, client)

	// Create many test pods to increase chance of race conditions
	pods := make([]v1.Pod, 100)
	for i := 0; i < 100; i++ {
		pods[i] = v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod%03d", i),
				Namespace: "default",
				UID:       types.UID(fmt.Sprintf("uid%03d", i)),
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
			},
		}
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Concurrent table updates
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("table update goroutine %d panicked: %v", goroutineID, r)
				}
			}()

			for j := 0; j < 20; j++ {
				rv.updateTableWithPods(pods)
				time.Sleep(time.Millisecond) // Small delay to increase interleaving
			}
		}(i)
	}

	// Concurrent selection changes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("selection goroutine %d panicked: %v", goroutineID, r)
				}
			}()

			for j := 0; j < 50; j++ {
				rv.SetSelectedRow(j % 100)
				time.Sleep(time.Microsecond * 100) // Very small delay
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	if errorCount == 0 {
		t.Log("Concurrent resourceMap access test completed successfully")
	} else {
		t.Fatalf("Concurrent resourceMap access test failed with %d errors", errorCount)
	}
}
