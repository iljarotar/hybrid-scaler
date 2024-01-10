package scaling

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
)

func TestHybrid(t *testing.T) {
	tests := []struct {
		name                                                  string
		state                                                 *strategy.State
		cpuLimitsToRequestsRatio, memoryLimitsToRequestsRatio *inf.Dec
		want                                                  *strategy.ScalingDecision
		wantErr                                               bool
	}{
		{
			name: "no scaling",
			state: &strategy.State{
				Replicas: 1,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(110, 0),
							Memory: inf.NewDec(110, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(220, 0),
							Memory: inf.NewDec(220, 0),
						},
					},
				},
				Constraints: strategy.Constraints{
					MinReplicas: 1,
					MaxReplicas: 10,
					MinResources: strategy.ResourcesList{
						CPU:    inf.NewDec(50, 0),
						Memory: inf.NewDec(50, 0),
					},
					MaxResources: strategy.ResourcesList{
						CPU:    inf.NewDec(500, 0),
						Memory: inf.NewDec(500, 0),
					},
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(50, 0),
						Memory: inf.NewDec(50, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(110, 0),
							Memory: inf.NewDec(110, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(220, 0),
							Memory: inf.NewDec(220, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			cpuLimitsToRequestsRatio:    inf.NewDec(2, 0),
			memoryLimitsToRequestsRatio: inf.NewDec(2, 0),
			want: &strategy.ScalingDecision{
				Replicas: 1,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(110, 0),
							Memory: inf.NewDec(110, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(220, 0),
							Memory: inf.NewDec(220, 0),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "all up",
			state: &strategy.State{
				Replicas: 1,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(200, 0),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
				Constraints: strategy.Constraints{
					MinReplicas: 1,
					MaxReplicas: 10,
					MinResources: strategy.ResourcesList{
						CPU:    inf.NewDec(50, 0),
						Memory: inf.NewDec(50, 0),
					},
					MaxResources: strategy.ResourcesList{
						CPU:    inf.NewDec(500, 0),
						Memory: inf.NewDec(500, 0),
					},
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(150, 0),
						Memory: inf.NewDec(150, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(200, 0),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			cpuLimitsToRequestsRatio:    inf.NewDec(2, 0),
			memoryLimitsToRequestsRatio: inf.NewDec(2, 0),
			want: &strategy.ScalingDecision{
				Replicas: 2,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(165, 0),
							Memory: inf.NewDec(165, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(330, 0),
							Memory: inf.NewDec(330, 0),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "replicas and cpu hit max",
			state: &strategy.State{
				Replicas: 10,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(250, 3),
							Memory: inf.NewDec(100, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(500, 3),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
				Constraints: strategy.Constraints{
					MinReplicas: 1,
					MaxReplicas: 10,
					MinResources: strategy.ResourcesList{
						CPU:    inf.NewDec(50, 3),
						Memory: inf.NewDec(50, 0),
					},
					MaxResources: strategy.ResourcesList{
						CPU:    inf.NewDec(500, 3),
						Memory: inf.NewDec(500, 0),
					},
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(500, 3),
						Memory: inf.NewDec(150, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(250, 3),
							Memory: inf.NewDec(100, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(500, 3),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			cpuLimitsToRequestsRatio:    inf.NewDec(2, 0),
			memoryLimitsToRequestsRatio: inf.NewDec(2, 0),
			want: &strategy.ScalingDecision{
				Replicas: 10,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(500, 3),
							Memory: inf.NewDec(330, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(500, 3),
							Memory: inf.NewDec(500, 0),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "all down",
			state: &strategy.State{
				Replicas: 9,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(200, 0),
							Memory: inf.NewDec(200, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(400, 0),
							Memory: inf.NewDec(400, 0),
						},
					},
				},
				Constraints: strategy.Constraints{
					MinReplicas: 1,
					MaxReplicas: 10,
					MinResources: strategy.ResourcesList{
						CPU:    inf.NewDec(50, 0),
						Memory: inf.NewDec(50, 0),
					},
					MaxResources: strategy.ResourcesList{
						CPU:    inf.NewDec(500, 0),
						Memory: inf.NewDec(500, 0),
					},
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(40, 0),
						Memory: inf.NewDec(40, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(200, 0),
							Memory: inf.NewDec(200, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(400, 0),
							Memory: inf.NewDec(400, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(60, 2),
					Memory: inf.NewDec(60, 2),
				},
			},
			cpuLimitsToRequestsRatio:    inf.NewDec(2, 0),
			memoryLimitsToRequestsRatio: inf.NewDec(2, 0),
			want: &strategy.ScalingDecision{
				Replicas: 6,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(110, 0),
							Memory: inf.NewDec(110, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(220, 0),
							Memory: inf.NewDec(220, 0),
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Hybrid(tt.state, tt.cpuLimitsToRequestsRatio, tt.memoryLimitsToRequestsRatio)
			if (err != nil) != tt.wantErr {
				t.Errorf("Hybrid() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmp.Comparer(decComparer)); diff != "" {
				t.Errorf("Hybrid() %v", diff)
			}
		})
	}
}

func TestVertical(t *testing.T) {
	tests := []struct {
		name                                                  string
		state                                                 *strategy.State
		cpuLimitsToRequestsRatio, memoryLimitsToRequestsRatio *inf.Dec
		want                                                  *strategy.ScalingDecision
		wantErr                                               bool
	}{
		{
			name: "replicas 0",
			state: &strategy.State{
				Replicas: 0,
			},
			cpuLimitsToRequestsRatio:    inf.NewDec(2, 0),
			memoryLimitsToRequestsRatio: inf.NewDec(2, 0),
			want:                        nil,
			wantErr:                     true,
		},
		{
			name:                        "no scaling",
			cpuLimitsToRequestsRatio:    inf.NewDec(2, 0),
			memoryLimitsToRequestsRatio: inf.NewDec(2, 0),
			state: &strategy.State{
				Replicas: 1,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(110, 0),
							Memory: inf.NewDec(110, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(220, 0),
							Memory: inf.NewDec(220, 0),
						},
					},
				},
				Constraints: strategy.Constraints{
					MinResources: strategy.ResourcesList{
						CPU:    inf.NewDec(50, 0),
						Memory: inf.NewDec(50, 0),
					},
					MaxResources: strategy.ResourcesList{
						CPU:    inf.NewDec(250, 0),
						Memory: inf.NewDec(250, 0),
					},
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(50, 0),
						Memory: inf.NewDec(50, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(110, 0),
							Memory: inf.NewDec(110, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(220, 0),
							Memory: inf.NewDec(220, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want: &strategy.ScalingDecision{
				Replicas: 1,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(110, 0),
							Memory: inf.NewDec(110, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(220, 0),
							Memory: inf.NewDec(220, 0),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:                        "scale both up",
			cpuLimitsToRequestsRatio:    inf.NewDec(2, 0),
			memoryLimitsToRequestsRatio: inf.NewDec(2, 0),
			state: &strategy.State{
				Replicas: 3,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(250, 3),
							Memory: inf.NewDec(100, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(500, 3),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
				Constraints: strategy.Constraints{
					MinResources: strategy.ResourcesList{
						CPU:    inf.NewDec(50, 3),
						Memory: inf.NewDec(50, 0),
					},
					MaxResources: strategy.ResourcesList{
						CPU:    inf.NewDec(500, 3),
						Memory: inf.NewDec(500, 0),
					},
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(500, 0),
						Memory: inf.NewDec(100, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(250, 3),
							Memory: inf.NewDec(100, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(500, 3),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want: &strategy.ScalingDecision{
				Replicas: 3,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(500, 3),
							Memory: inf.NewDec(220, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(500, 3),
							Memory: inf.NewDec(440, 0),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:                        "scale both down",
			cpuLimitsToRequestsRatio:    inf.NewDec(2, 0),
			memoryLimitsToRequestsRatio: inf.NewDec(2, 0),
			state: &strategy.State{
				Replicas: 3,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(200, 0),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
				Constraints: strategy.Constraints{
					MinResources: strategy.ResourcesList{
						CPU:    inf.NewDec(50, 0),
						Memory: inf.NewDec(50, 0),
					},
					MaxResources: strategy.ResourcesList{
						CPU:    inf.NewDec(500, 0),
						Memory: inf.NewDec(500, 0),
					},
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(25, 0),
						Memory: inf.NewDec(25, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(200, 0),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want: &strategy.ScalingDecision{
				Replicas: 3,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(55, 0),
							Memory: inf.NewDec(55, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(110, 0),
							Memory: inf.NewDec(110, 0),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:                        "cpu up to max, memory down to min",
			cpuLimitsToRequestsRatio:    inf.NewDec(2, 0),
			memoryLimitsToRequestsRatio: inf.NewDec(2, 0),
			state: &strategy.State{
				Replicas: 3,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(200, 0),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
				Constraints: strategy.Constraints{
					MinResources: strategy.ResourcesList{
						CPU:    inf.NewDec(50, 0),
						Memory: inf.NewDec(50, 0),
					},
					MaxResources: strategy.ResourcesList{
						CPU:    inf.NewDec(500, 0),
						Memory: inf.NewDec(500, 0),
					},
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(150, 0),
						Memory: inf.NewDec(125, 1),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(200, 0),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want: &strategy.ScalingDecision{
				Replicas: 3,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(330, 0),
							Memory: inf.NewDec(50, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(500, 0),
							Memory: inf.NewDec(100, 0),
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Vertical(tt.state, tt.cpuLimitsToRequestsRatio, tt.memoryLimitsToRequestsRatio)
			if (err != nil) != tt.wantErr {
				t.Errorf("Vertical() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmp.Comparer(decComparer)); diff != "" {
				t.Errorf("Vertical() %v", diff)
			}
		})
	}
}

func Test_currentToTargetUtilizationRatio(t *testing.T) {
	tests := []struct {
		name              string
		usage             *inf.Dec
		requests          *inf.Dec
		targetUtilization *inf.Dec
		want              *inf.Dec
		wantErr           bool
	}{
		{
			name:              "requests is zero",
			usage:             inf.NewDec(100, 0),
			requests:          inf.NewDec(0, 0),
			targetUtilization: inf.NewDec(10, 0),
			want:              nil,
			wantErr:           true,
		},
		{
			name:              "target utilization is zero",
			usage:             inf.NewDec(100, 0),
			requests:          inf.NewDec(10, 0),
			targetUtilization: inf.NewDec(0, 0),
			want:              nil,
			wantErr:           true,
		},
		{
			name:              "utilization below target",
			usage:             inf.NewDec(50, 0),
			requests:          inf.NewDec(1000, 1),
			targetUtilization: inf.NewDec(8, 1),
			want:              inf.NewDec(625, 3),
			wantErr:           false,
		},
		{
			name:              "utilization exceeds target",
			usage:             inf.NewDec(300, 0),
			requests:          inf.NewDec(2000, 1),
			targetUtilization: inf.NewDec(50, 2),
			want:              inf.NewDec(3, 0),
			wantErr:           false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := currentToTargetUtilizationRatio(tt.usage, tt.requests, tt.targetUtilization)
			if (err != nil) != tt.wantErr {
				t.Errorf("currentToTargetUtilizationRatio() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmp.Comparer(decComparer)); diff != "" {
				t.Errorf("currentToTargetUtilizationRatio() %v", diff)
			}
		})
	}
}

func TestHorizontal(t *testing.T) {
	tests := []struct {
		name    string
		state   *strategy.State
		want    *strategy.ScalingDecision
		wantErr bool
	}{
		{
			name: "no scaling required",
			state: &strategy.State{
				Replicas: 3,
				Constraints: strategy.Constraints{
					MinReplicas: 1,
					MaxReplicas: 10,
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(100, 0),
						Memory: inf.NewDec(100, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(200, 0),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want: &strategy.ScalingDecision{
				Replicas: 3,
			},
			wantErr: false,
		},
		{
			name: "scale up, no change to container resources",
			state: &strategy.State{
				Replicas: 3,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(200, 0),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
				Constraints: strategy.Constraints{
					MinReplicas: 1,
					MaxReplicas: 10,
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(100, 0),
						Memory: inf.NewDec(100, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want: &strategy.ScalingDecision{
				Replicas: 6,
				ContainerResources: strategy.ContainerResources{
					"container": {
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(200, 0),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "scale down",
			state: &strategy.State{
				Replicas: 3,
				Constraints: strategy.Constraints{
					MinReplicas: 1,
					MaxReplicas: 10,
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(50, 0),
						Memory: inf.NewDec(50, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(301, 0),
							Memory: inf.NewDec(301, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want: &strategy.ScalingDecision{
				Replicas: 1,
			},
			wantErr: false,
		},
		{
			name: "scale up to max",
			state: &strategy.State{
				Replicas: 3,
				Constraints: strategy.Constraints{
					MinReplicas: 1,
					MaxReplicas: 5,
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(300, 0),
						Memory: inf.NewDec(300, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(300, 0),
							Memory: inf.NewDec(300, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want: &strategy.ScalingDecision{
				Replicas: 5,
			},
			wantErr: false,
		},
		{
			name: "scale down to min",
			state: &strategy.State{
				Replicas: 3,
				Constraints: strategy.Constraints{
					MinReplicas: 2,
					MaxReplicas: 5,
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(50, 0),
						Memory: inf.NewDec(50, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(301, 0),
							Memory: inf.NewDec(301, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want: &strategy.ScalingDecision{
				Replicas: 2,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Horizontal(tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("Horizontal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmp.Comparer(decComparer)); diff != "" {
				t.Errorf("Horizontal() %v", diff)
			}
		})
	}
}

func Test_calculateDesiredReplicas(t *testing.T) {
	tests := []struct {
		name    string
		state   *strategy.State
		want    *inf.Dec
		wantErr bool
	}{
		{
			name:    "replicas 0",
			state:   &strategy.State{},
			want:    nil,
			wantErr: true,
		},
		{
			name: "scale up",
			state: &strategy.State{
				Replicas: 1,
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(200, 0),
						Memory: inf.NewDec(200, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(200, 0),
							Memory: inf.NewDec(200, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want:    inf.NewDec(2, 0),
			wantErr: false,
		},
		{
			name: "scale down",
			state: &strategy.State{
				Replicas: 10,
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(25, 0),
						Memory: inf.NewDec(25, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want:    inf.NewDec(5, 0),
			wantErr: false,
		},
		{
			name: "scale up based on cpu",
			state: &strategy.State{
				Replicas: 1,
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(150, 0),
						Memory: inf.NewDec(100, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want:    inf.NewDec(3, 0),
			wantErr: false,
		},
		{
			name: "scale down based on cpu",
			state: &strategy.State{
				Replicas: 5,
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(25, 0),
						Memory: inf.NewDec(10, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want:    inf.NewDec(3, 0),
			wantErr: false,
		},
		{
			name: "scale up based on memory",
			state: &strategy.State{
				Replicas: 2,
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(25, 0),
						Memory: inf.NewDec(75, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want:    inf.NewDec(3, 0),
			wantErr: false,
		},
		{
			name: "scale down based on memory",
			state: &strategy.State{
				Replicas: 6,
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(25, 0),
						Memory: inf.NewDec(33, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(100, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(50, 2),
				},
			},
			want:    inf.NewDec(4, 0),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculateDesiredReplicas(tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("calculateDesiredReplicas() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calculateDesiredReplicas() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_limitScalingValue(t *testing.T) {
	tests := []struct {
		name    string
		desired *inf.Dec
		min     *inf.Dec
		max     *inf.Dec
		want    *inf.Dec
	}{
		{
			name:    "no limit necessary",
			desired: inf.NewDec(5, 0),
			min:     inf.NewDec(4, 0),
			max:     inf.NewDec(6, 0),
			want:    inf.NewDec(5, 0),
		},
		{
			name:    "limit to min",
			desired: inf.NewDec(3, 0),
			min:     inf.NewDec(4, 0),
			max:     inf.NewDec(6, 0),
			want:    inf.NewDec(4, 0),
		},
		{
			name:    "limit to max",
			desired: inf.NewDec(7, 0),
			min:     inf.NewDec(4, 0),
			max:     inf.NewDec(6, 0),
			want:    inf.NewDec(6, 0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := limitValue(tt.desired, tt.min, tt.max); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("limitScalingValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecToInt64(t *testing.T) {
	tests := []struct {
		name  string
		value *inf.Dec
		want  int64
	}{
		{
			name:  "zero",
			value: inf.NewDec(0, 0),
			want:  0,
		},
		{
			name:  "positive integer with scale 0",
			value: inf.NewDec(10, 0),
			want:  10,
		},
		{
			name:  "positive integer with positive scale",
			value: inf.NewDec(10, 1),
			want:  1,
		},
		{
			name:  "positive integer with negative scale",
			value: inf.NewDec(10, -1),
			want:  100,
		},
		{
			name:  "negative integer with scale 0",
			value: inf.NewDec(-10, 0),
			want:  -10,
		},
		{
			name:  "negative integer with positive scale",
			value: inf.NewDec(-50, 1),
			want:  -5,
		},
		{
			name:  "negative integer with negative scale",
			value: inf.NewDec(-50, -1),
			want:  -500,
		},
		{
			name:  "positive rational to zero",
			value: inf.NewDec(33, 2),
			want:  0,
		},
		{
			name:  "negative rational to zero",
			value: inf.NewDec(-26, 2),
			want:  0,
		},
		{
			name:  "positive rational",
			value: inf.NewDec(246, 1),
			want:  24,
		},
		{
			name:  "negative rational",
			value: inf.NewDec(-534, 2),
			want:  -5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DecToInt64(tt.value); got != tt.want {
				t.Errorf("DecToInt64() = %v, want %v", got, tt.want)
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
