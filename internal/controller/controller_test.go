package controller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	scalingv1 "github.com/iljarotar/hybrid-scaler/api/v1"
	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func Test_getContainerResources(t *testing.T) {
	tests := []struct {
		name       string
		containers []corev1.Container
		want       map[string]scalingv1.ContainerResources
	}{
		{
			name: "correctly convert container array to map",
			containers: []corev1.Container{
				{
					Name: "container1",
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("2000Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("1000Mi"),
						},
					},
				},
				{
					Name: "container2",
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("300m"),
							corev1.ResourceMemory: resource.MustParse("3000Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("2000Mi"),
						},
					},
				}},
			want: map[string]scalingv1.ContainerResources{
				"container1": {
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("2000Mi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("1000Mi"),
					},
				},
				"container2": {
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("300m"),
						corev1.ResourceMemory: resource.MustParse("3000Mi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("2000Mi"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getContainerResources(tt.containers)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("getContainerResources() %v", diff)
			}
		})
	}
}

func Test_prepareState(t *testing.T) {
	minReplicas := int32(1)
	maxReplicas := int32(5)
	threshold := resource.MustParse("1M")

	tests := []struct {
		name    string
		status  scalingv1.HybridScalerStatus
		spec    scalingv1.HybridScalerSpec
		want    *strategy.State
		wantErr bool
	}{
		{
			name: "correctly convert spec and status to strategy state",
			status: scalingv1.HybridScalerStatus{
				Replicas: 3,
				ContainerResources: map[string]scalingv1.ContainerResources{
					"container1": {
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("5G"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("2"),
							corev1.ResourceMemory: resource.MustParse("6G"),
						},
					},
					"container2": {
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("250M"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("2G"),
						},
					},
				},
				ContainerMetrics: []v1beta1.ContainerMetrics{
					{
						Name: "container1",
						Usage: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("2G"),
						},
					},
					{
						Name: "container2",
						Usage: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("1G"),
						},
					},
					{
						Name: "container1",
						Usage: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("2G"),
						},
					},
					{
						Name: "container2",
						Usage: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("1G"),
						},
					},
					{
						Name: "container1",
						Usage: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("2G"),
						},
					},
					{
						Name: "container2",
						Usage: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("1G"),
						},
					},
				},
			},
			spec: scalingv1.HybridScalerSpec{
				MinReplicas: &minReplicas,
				MaxReplicas: &maxReplicas,
				ResourcePolicy: scalingv1.ResourcePolicy{
					MinAllowed: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100M"),
					},
					MaxAllowed: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("8G"),
					},
					TargetUtilization: map[corev1.ResourceName]int32{
						corev1.ResourceCPU:    50,
						corev1.ResourceMemory: 80,
					},
				},
				LatencyThreshold: &threshold,
			},
			want: &strategy.State{
				Replicas: 3,
				ContainerResources: strategy.ContainerResources{
					"container1": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(1, 0),
							Memory: inf.NewDec(5, -9),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(2, 0),
							Memory: inf.NewDec(6, -9),
						},
					},
					"container2": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(1, 1),
							Memory: inf.NewDec(250, -6),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(1, 0),
							Memory: inf.NewDec(2, -9),
						},
					},
				},
				Constraints: strategy.Constraints{
					MinReplicas: minReplicas,
					MaxReplicas: maxReplicas,
					MinResources: strategy.ResourcesList{
						CPU:    inf.NewDec(1, 1),
						Memory: inf.NewDec(100, -6),
					},
					MaxResources: strategy.ResourcesList{
						CPU:    inf.NewDec(4, 0),
						Memory: inf.NewDec(8, -9),
					},
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(7, 1),
						Memory: inf.NewDec(3, -9),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(11, 1),
							Memory: inf.NewDec(525, -7),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(3, 0),
							Memory: inf.NewDec(8, -9),
						},
					},
					LatencyThresholdExceeded: false,
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(80, 2),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := prepareState(tt.status, tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("prepareState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmp.Comparer(decComparer)); diff != "" {
				t.Errorf("prepareState() %v", diff)
			}
		})
	}
}

func Test_interpretResourceScaling(t *testing.T) {
	tests := []struct {
		name     string
		decision *strategy.ScalingDecision
		want     map[string]scalingv1.ContainerResources
	}{
		{
			name: "correctly interpret scaling decision",
			decision: &strategy.ScalingDecision{
				ContainerResources: strategy.ContainerResources{
					"container1": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(200, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(200, 0),
							Memory: inf.NewDec(200, 0),
						},
					},
					"container2": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(150, 0),
							Memory: inf.NewDec(250, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(250, 0),
							Memory: inf.NewDec(250, 0),
						},
					},
				},
			},
			want: map[string]scalingv1.ContainerResources{
				"container1": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewDecimalQuantity(*inf.NewDec(100, 0), resource.DecimalExponent),
						corev1.ResourceMemory: *resource.NewDecimalQuantity(*inf.NewDec(200, 0), resource.DecimalSI),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewDecimalQuantity(*inf.NewDec(200, 0), resource.DecimalExponent),
						corev1.ResourceMemory: *resource.NewDecimalQuantity(*inf.NewDec(200, 0), resource.DecimalSI),
					},
				},
				"container2": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewDecimalQuantity(*inf.NewDec(150, 0), resource.DecimalExponent),
						corev1.ResourceMemory: *resource.NewDecimalQuantity(*inf.NewDec(250, 0), resource.DecimalSI),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewDecimalQuantity(*inf.NewDec(250, 0), resource.DecimalExponent),
						corev1.ResourceMemory: *resource.NewDecimalQuantity(*inf.NewDec(250, 0), resource.DecimalSI),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := interpretResourceScaling(tt.decision)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("interpretResourceScaling() %v", diff)
			}
		})
	}
}

func decComparer(a, b *inf.Dec) bool {
	if a == nil && b != nil {
		return false
	}

	if b == nil && a != nil {
		return false
	}

	if a == nil && b == nil {
		return true
	}

	return a.Cmp(b) == 0
}
