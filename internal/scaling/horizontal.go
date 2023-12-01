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

	desiredReplicas, err := calculateDesiredReplicas(state)
	if err != nil {
		return nil, err
	}

	minReplicas := inf.NewDec(int64(state.MinReplicas), 0)
	maxReplicas := inf.NewDec(int64(state.MaxReplicas), 0)
	limitedReplicas := limitScalingValue(desiredReplicas, minReplicas, maxReplicas)

	replicas := DecToInt64(limitedReplicas)

	return &strategy.ScalingDecision{
		Replicas:           int32(replicas),
		ContainerResources: containerResources,
	}, nil
}

func calculateDesiredReplicas(state *strategy.State) (*inf.Dec, error) {
	currentReplicas := inf.NewDec(int64(state.Replicas), 0)
	zero := inf.NewDec(0, 0)

	if currentReplicas.Cmp(zero) == 0 {
		return nil, fmt.Errorf("cannot calculate new number of replicas, current replicas is zero")
	}

	averagePodCpuUsage := new(inf.Dec).QuoRound(state.PodMetrics.ResourceUsage.CPU, currentReplicas, 8, inf.RoundHalfUp)
	averagePodMemoryUsage := new(inf.Dec).QuoRound(state.PodMetrics.ResourceUsage.Memory, currentReplicas, 8, inf.RoundHalfUp)
	cpuRequests := state.PodMetrics.Requests.CPU
	memoryRequests := state.PodMetrics.Requests.Memory

	cpuCurrentToTargetRatio, err := currentToTargetUtilizationRatio(averagePodCpuUsage, cpuRequests, state.TargetUtilization.CPU)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate cpu current to target utilization ratio, %w", err)
	}

	desiredReplicasCpu := new(inf.Dec).Mul(currentReplicas, cpuCurrentToTargetRatio)
	desiredReplicasCpu.Round(desiredReplicasCpu, 0, inf.RoundCeil)

	memoryCurrentToTargetRatio, err := currentToTargetUtilizationRatio(averagePodMemoryUsage, memoryRequests, state.TargetUtilization.Memory)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate memory current to target utilization ratio, %w", err)
	}

	desiredReplicasMemory := new(inf.Dec).Mul(currentReplicas, memoryCurrentToTargetRatio)
	desiredReplicasMemory.Round(desiredReplicasMemory, 0, inf.RoundCeil)

	desiredReplicas := desiredReplicasCpu
	if desiredReplicas.Cmp(desiredReplicasMemory) < 0 {
		desiredReplicas = desiredReplicasMemory
	}

	return desiredReplicas, nil
}
