package scaling

import (
	"fmt"

	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
)

// Horizontal recommends a new number of replicas based on the following calculation
// For each resource of cpu and memory it calculates the desired number of replicas as
// `desired = ceil(current * currentUtilization / targetUtilization)` (same formula as `HPA`)
// if picks the maximum of both values and compares that to the minimum and maximum allowed replicas
// the result will be max(min(desired, maxReplicas), minReplicas)
func Horizontal(state *strategy.State) (*strategy.ScalingDecision, error) {
	containerResources := make(strategy.ContainerResources)
	for name, metrics := range state.ContainerMetrics {
		containerResources[name] = metrics.Resources
	}

	cpuUsage := state.PodMetrics.ResourceUsage.CPU
	memoryUsage := state.PodMetrics.ResourceUsage.Memory
	cpuRequests := state.PodMetrics.Requests.CPU
	memoryRequests := state.PodMetrics.Requests.Memory

	currentReplicas := inf.NewDec(int64(state.Replicas), 0)

	cpuCurrentToTargetRatio, err := currentToTargetUtilizationRatio(cpuUsage, cpuRequests, state.TargetUtilization.CPU)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate cpu current to target utilization ratio, %w", err)
	}

	desiredReplicasCpu := new(inf.Dec).Mul(currentReplicas, cpuCurrentToTargetRatio)
	desiredReplicasCpu.Round(desiredReplicasCpu, 0, inf.RoundCeil)

	memoryCurrentToTargetRatio, err := currentToTargetUtilizationRatio(memoryUsage, memoryRequests, state.TargetUtilization.Memory)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate memory current to target utilization ratio, %w", err)
	}

	desiredReplicasMemory := new(inf.Dec).Mul(currentReplicas, memoryCurrentToTargetRatio)
	desiredReplicasMemory.Round(desiredReplicasMemory, 0, inf.RoundCeil)

	desiredReplicas := desiredReplicasCpu
	if desiredReplicas.Cmp(desiredReplicasMemory) < 0 {
		desiredReplicas = desiredReplicasMemory
	}

	maxReplicas := inf.NewDec(int64(state.Constraints.MaxReplicas), 0)
	if desiredReplicas.Cmp(maxReplicas) > 0 {
		desiredReplicas = maxReplicas
	}

	minReplicas := inf.NewDec(int64(state.Constraints.MinReplicas), 0)
	if desiredReplicas.Cmp(minReplicas) < 0 {
		desiredReplicas = minReplicas
	}

	replicas := desiredReplicas.UnscaledBig().Int64()

	return &strategy.ScalingDecision{
		Replicas:           int32(replicas),
		ContainerResources: containerResources,
	}, nil
}
