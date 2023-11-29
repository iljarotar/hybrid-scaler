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

	cpuPercentage := inf.NewDec(0, 0)
	memoryPercentage := inf.NewDec(0, 0)
	zero := inf.NewDec(0, 0)

	if cpuRequests.Cmp(zero) == 0 {
		return nil, fmt.Errorf("cpu requests should not be zero")
	}
	cpuPercentage.QuoRound(cpuUsage, cpuRequests, 8, inf.RoundHalfUp)

	if memoryRequests.Cmp(zero) == 0 {
		return nil, fmt.Errorf("memory requests should not be zero")
	}
	memoryPercentage.QuoRound(memoryUsage, memoryRequests, 8, inf.RoundHalfUp)

	currentReplicas := inf.NewDec(int64(state.Replicas), 0)

	if state.TargetUtilization.CPU.Cmp(zero) == 0 {
		return nil, fmt.Errorf("cpu target utilization should not be zero")
	}
	desiredReplicasCpu := new(inf.Dec).Mul(currentReplicas, cpuPercentage.QuoRound(cpuPercentage, state.TargetUtilization.CPU, 8, inf.RoundHalfUp))
	desiredReplicasCpu.Round(desiredReplicasCpu, 0, inf.RoundCeil)

	if state.TargetUtilization.Memory.Cmp(zero) == 0 {
		return nil, fmt.Errorf("memory target utilization should not be zero")
	}
	desiredReplicasMemory := new(inf.Dec).Mul(currentReplicas, memoryPercentage.QuoRound(memoryPercentage, state.TargetUtilization.Memory, 8, inf.RoundHalfUp))
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
