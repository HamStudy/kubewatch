package views

import (
	"testing"

	"github.com/HamStudy/kubewatch/internal/core"
	"github.com/HamStudy/kubewatch/internal/template"
	"github.com/HamStudy/kubewatch/internal/transformers"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestContextColumnLogic(t *testing.T) {
	// Create a resource view
	state := core.NewState(&core.Config{})
	rv := NewResourceView(state, nil)

	// Test single context - should not show context column
	state.SetCurrentContexts([]string{"context1"})
	rv.updateColumnsForResourceType()

	if rv.showContextColumn {
		t.Error("Context column should not be shown with single context")
	}

	// Test multiple contexts - should show context column
	state.SetCurrentContexts([]string{"context1", "context2"})
	rv.isMultiContext = true
	rv.updateColumnsForResourceType()

	if !rv.showContextColumn {
		t.Error("Context column should be shown with multiple contexts")
	}

	// Test no contexts - should not show context column
	state.SetCurrentContexts([]string{})
	rv.updateColumnsForResourceType()

	if rv.showContextColumn {
		t.Error("Context column should not be shown with no contexts")
	}
}

func TestUniqKeyGeneration(t *testing.T) {
	// Create a deployment transformer
	transformer := transformers.NewDeploymentTransformer()
	engine := template.NewEngine()

	// Create test deployments
	deployment1 := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-app",
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "app",
							Image: "nginx:1.20",
						},
					},
				},
			},
		},
	}

	deployment2 := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-app",
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "app",
							Image: "nginx:1.21", // Different image
						},
					},
				},
			},
		},
	}

	// Generate unique keys
	key1, err := transformer.GetUniqKey(deployment1, engine)
	if err != nil {
		t.Fatalf("Failed to generate unique key for deployment1: %v", err)
	}

	key2, err := transformer.GetUniqKey(deployment2, engine)
	if err != nil {
		t.Fatalf("Failed to generate unique key for deployment2: %v", err)
	}

	// Keys should be different because images are different
	if key1 == key2 {
		t.Errorf("Expected different unique keys for deployments with different images, got: %s == %s", key1, key2)
	}

	// Test same deployment should have same key
	key3, err := transformer.GetUniqKey(deployment1, engine)
	if err != nil {
		t.Fatalf("Failed to generate unique key for deployment1 (second time): %v", err)
	}

	if key1 != key3 {
		t.Errorf("Expected same unique key for same deployment, got: %s != %s", key1, key3)
	}
}

func TestResourceGrouping(t *testing.T) {
	// Create a resource view
	state := core.NewState(&core.Config{})
	rv := NewResourceView(state, nil)
	rv.enableGrouping = true

	// Create test deployments with same name but different images
	deployment1 := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-app",
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "app",
							Image: "nginx:1.20",
						},
					},
				},
			},
		},
	}

	deployment2 := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-app",
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "app",
							Image: "nginx:1.20", // Same image - should group
						},
					},
				},
			},
		},
	}

	deployment3 := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-app",
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "app",
							Image: "nginx:1.21", // Different image - separate group
						},
					},
				},
			},
		},
	}

	resources := []interface{}{deployment1, deployment2, deployment3}

	// Group resources
	groups, err := rv.groupResources(resources, "Deployment")
	if err != nil {
		t.Fatalf("Failed to group resources: %v", err)
	}

	// Should have 2 groups: one with deployment1&2, one with deployment3
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}

	// Check that one group has 2 resources and one has 1
	groupSizes := make([]int, 0, len(groups))
	for _, group := range groups {
		groupSizes = append(groupSizes, len(group))
	}

	// Sort group sizes for consistent testing
	if len(groupSizes) == 2 {
		if groupSizes[0] > groupSizes[1] {
			groupSizes[0], groupSizes[1] = groupSizes[1], groupSizes[0]
		}
		if groupSizes[0] != 1 || groupSizes[1] != 2 {
			t.Errorf("Expected group sizes [1, 2], got %v", groupSizes)
		}
	}
}

func TestGroupingDisabled(t *testing.T) {
	// Create a resource view with grouping disabled
	state := core.NewState(&core.Config{})
	rv := NewResourceView(state, nil)
	rv.enableGrouping = false

	// Create test deployments
	deployment1 := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-app",
		},
	}

	deployment2 := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-app",
		},
	}

	resources := []interface{}{deployment1, deployment2}

	// Group resources
	groups, err := rv.groupResources(resources, "Deployment")
	if err != nil {
		t.Fatalf("Failed to group resources: %v", err)
	}

	// Should have 2 individual groups when grouping is disabled
	if len(groups) != 2 {
		t.Errorf("Expected 2 individual groups when grouping disabled, got %d", len(groups))
	}

	// Each group should have exactly 1 resource
	groupIndex := 0
	for _, group := range groups {
		if len(group) != 1 {
			t.Errorf("Group %d should have 1 resource when grouping disabled, got %d", groupIndex, len(group))
		}
		groupIndex++
	}
}
