package scaling

import (
	"fmt"

	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
)

// Recommends new resource requests and limits keeping ratios between both and each container's share of the pod's resources
func Vertical(s *strategy.State, cpuLimitsToRequestsRatio, memoryLimitsToRequestsRatio *inf.Dec) (*strategy.ScalingDecision, error) {
	if cpuLimitsToRequestsRatio == nil || memoryLimitsToRequestsRatio == nil {
		return nil, fmt.Errorf("no limits to requests ratios provided")
	}

	containerResources := make(strategy.ContainerResources)
	currentReplicas := inf.NewDec(int64(s.Replicas), 0)
	zero := inf.NewDec(0, 0)

	if currentReplicas.Cmp(zero) == 0 {
		return nil, fmt.Errorf("unable to calculate new pod resources, current number of replicas is zero")
	}

	podCpuRequests := s.PodMetrics.Requests.CPU
	podMemoryRequests := s.PodMetrics.Requests.Memory

	podCpuLimits := s.PodMetrics.Limits.CPU
	podMemoryLimits := s.PodMetrics.Limits.Memory

	cpuCurrentToTargetRatio, err := currentToTargetUtilizationRatio(s.PodMetrics.ResourceUsage.CPU, podCpuRequests, s.TargetUtilization.CPU)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate cpu current to target utilization ratio, %w", err)
	}

	desiredPodCpuRequests := new(inf.Dec).Mul(podCpuRequests, cpuCurrentToTargetRatio)
	minCpu := s.Constraints.MinResources.CPU
	maxCpu := s.Constraints.MaxResources.CPU

	desiredPodCpuRequests = limitValue(desiredPodCpuRequests, minCpu, maxCpu)
	desiredPodCpuLimits := new(inf.Dec).Mul(desiredPodCpuRequests, cpuLimitsToRequestsRatio)
	desiredPodCpuLimits = limitValue(desiredPodCpuLimits, minCpu, maxCpu)

	memoryCurrentToTargetRatio, err := currentToTargetUtilizationRatio(s.PodMetrics.ResourceUsage.Memory, podMemoryRequests, s.TargetUtilization.Memory)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate memory current to target utilization ratio, %w", err)
	}

	desiredPodMemoryRequests := new(inf.Dec).Mul(podMemoryRequests, memoryCurrentToTargetRatio)
	minMemory := s.Constraints.MinResources.Memory
	maxMemory := s.Constraints.MaxResources.Memory

	desiredPodMemoryRequests = limitValue(desiredPodMemoryRequests, minMemory, maxMemory)
	desiredPodMemoryLimits := new(inf.Dec).Mul(desiredPodMemoryRequests, memoryLimitsToRequestsRatio)
	desiredPodMemoryLimits = limitValue(desiredPodMemoryLimits, minMemory, maxMemory)

	cpuRounder := inf.RoundHalfUp
	memoryRounder := inf.RoundHalfUp

	if desiredPodCpuRequests.Cmp(minCpu) == 0 {
		cpuRounder = inf.RoundCeil
	}

	if desiredPodCpuLimits.Cmp(maxCpu) == 0 {
		cpuRounder = inf.RoundDown
	}

	if desiredPodMemoryRequests.Cmp(minMemory) == 0 {
		memoryRounder = inf.RoundCeil
	}

	if desiredPodMemoryLimits.Cmp(maxMemory) == 0 {
		memoryRounder = inf.RoundDown
	}

	for name, resources := range s.ContainerResources {
		cpuRequests := resources.Requests.CPU
		cpuLimits := resources.Limits.CPU

		desiredCpuRequests := new(inf.Dec).Mul(cpuRequests, new(inf.Dec).QuoRound(desiredPodCpuRequests, podCpuRequests, 8, cpuRounder))
		desiredCpuLimits := new(inf.Dec).Mul(cpuLimits, new(inf.Dec).QuoRound(desiredPodCpuLimits, podCpuLimits, 8, cpuRounder))

		memoryRequests := resources.Requests.Memory
		memoryLimits := resources.Limits.Memory

		desiredMemoryRequests := new(inf.Dec).Mul(memoryRequests, new(inf.Dec).QuoRound(desiredPodMemoryRequests, podMemoryRequests, 8, memoryRounder))
		desiredMemoryLimits := new(inf.Dec).Mul(memoryLimits, new(inf.Dec).QuoRound(desiredPodMemoryLimits, podMemoryLimits, 8, memoryRounder))

		containerResources[name] = strategy.Resources{
			Requests: strategy.ResourcesList{
				CPU:    desiredCpuRequests,
				Memory: desiredMemoryRequests,
			},
			Limits: strategy.ResourcesList{
				CPU:    desiredCpuLimits,
				Memory: desiredMemoryLimits,
			},
		}
	}

	return &strategy.ScalingDecision{
		Replicas:           s.Replicas,
		ContainerResources: containerResources,
	}, nil
}
