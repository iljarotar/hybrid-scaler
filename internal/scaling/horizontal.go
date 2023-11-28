package scaling

import (
	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
)

// Recommends new number of replicas
func Horizontal(state *strategy.State) *strategy.ScalingDecision {
	// TODO: implement
	// while agent compares current usage with limits, here the request utilization should be used

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

	if cpuRequests.Cmp(zero) != 0 {
		cpuPercentage.QuoRound(cpuUsage, cpuRequests, 8, inf.RoundHalfUp)
	}

	if memoryRequests.Cmp(zero) != 0 {
		memoryPercentage.QuoRound(memoryUsage, memoryRequests, 8, inf.RoundHalfUp)
	}

	currentReplicas := inf.NewDec(int64(state.Replicas), 0)

	desiredReplicasCpu := new(inf.Dec).Mul(currentReplicas, cpuPercentage.QuoRound(cpuPercentage, state.TargetUtilization.CPU, 8, inf.RoundCeil))

	desiredReplicasMemory := new(inf.Dec).Mul(currentReplicas, memoryPercentage.QuoRound(memoryPercentage, state.TargetUtilization.Memory, 8, inf.RoundCeil))

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

	// TODO:
	// check calculation once more
	// check divisions by zero and return error in that case

	return &strategy.ScalingDecision{
		Replicas:           int32(replicas),
		ContainerResources: containerResources,
	}
}
