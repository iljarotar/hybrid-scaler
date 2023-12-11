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
func Horizontal(s *strategy.State) (*strategy.ScalingDecision, error) {
	desiredReplicas, err := calculateDesiredReplicas(s)
	if err != nil {
		return nil, err
	}

	minReplicas := inf.NewDec(int64(s.MinReplicas), 0)
	maxReplicas := inf.NewDec(int64(s.MaxReplicas), 0)
	limitedReplicas := limitValue(desiredReplicas, minReplicas, maxReplicas)

	replicas := DecToInt64(limitedReplicas)

	return &strategy.ScalingDecision{
		Replicas:           int32(replicas),
		ContainerResources: s.ContainerResources,
	}, nil
}

func calculateDesiredReplicas(s *strategy.State) (*inf.Dec, error) {
	currentReplicas := inf.NewDec(int64(s.Replicas), 0)
	zero := inf.NewDec(0, 0)

	if currentReplicas.Cmp(zero) == 0 {
		return nil, fmt.Errorf("cannot calculate new number of replicas, current replicas is zero")
	}

	cpuRequests := s.PodMetrics.Requests.CPU
	memoryRequests := s.PodMetrics.Requests.Memory

	cpuCurrentToTargetRatio, err := currentToTargetUtilizationRatio(s.PodMetrics.ResourceUsage.CPU, cpuRequests, s.TargetUtilization.CPU)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate cpu current to target utilization ratio, %w", err)
	}

	desiredReplicasCpu := new(inf.Dec).Mul(currentReplicas, cpuCurrentToTargetRatio)
	desiredReplicasCpu.Round(desiredReplicasCpu, 0, inf.RoundCeil)

	memoryCurrentToTargetRatio, err := currentToTargetUtilizationRatio(s.PodMetrics.ResourceUsage.Memory, memoryRequests, s.TargetUtilization.Memory)
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
