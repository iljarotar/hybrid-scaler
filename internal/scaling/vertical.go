package scaling

import (
	"fmt"

	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
)

// Recommends new resource requests and limits keeping ratios between both and each container's share of the pod's resources
func Vertical(state *strategy.State) (*strategy.ScalingDecision, error) {
	containerResources := make(strategy.ContainerResources)
	for name, metrics := range state.ContainerMetrics {
		containerResources[name] = metrics.Resources
	}

	zero := inf.NewDec(0, 0)
	cpuUsage := state.PodMetrics.ResourceUsage.CPU
	memoryUsage := state.PodMetrics.ResourceUsage.Memory

	podCpuRequests := state.PodMetrics.Requests.CPU
	podMemoryRequests := state.PodMetrics.Requests.Memory

	podCpuLimits := state.PodMetrics.Limits.CPU
	podMemoryLimits := state.PodMetrics.Limits.Memory

	cpuCurrentToTargetRatio, err := currentToTargetUtilizationRatio(cpuUsage, podCpuRequests, state.TargetUtilization.CPU)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate cpu current to target utilization ratio, %w", err)
	}

	desiredPodCpuRequests := new(inf.Dec).Mul(podCpuRequests, cpuCurrentToTargetRatio)
	minCpu := state.Constraints.MinResources.CPU
	maxCpu := state.Constraints.MaxResources.CPU

	if desiredPodCpuRequests.Cmp(minCpu) < 0 {
		desiredPodCpuRequests = minCpu
	}

	if desiredPodCpuRequests.Cmp(maxCpu) > 0 {
		desiredPodCpuRequests = maxCpu
	}

	if podCpuRequests.Cmp(zero) == 0 {
		return nil, fmt.Errorf("cpu requests cannot be zero")
	}

	desiredPodCpuLimits := new(inf.Dec).Mul(desiredPodCpuRequests, new(inf.Dec).QuoRound(podCpuLimits, podCpuRequests, 8, inf.RoundHalfUp))

	if desiredPodCpuLimits.Cmp(minCpu) < 0 {
		desiredPodCpuLimits = minCpu
	}

	if desiredPodCpuLimits.Cmp(maxCpu) > 0 {
		desiredPodCpuLimits = maxCpu
	}

	memoryCurrentToTargetRatio, err := currentToTargetUtilizationRatio(memoryUsage, podMemoryRequests, state.TargetUtilization.Memory)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate memory current to target utilization ratio, %w", err)
	}

	desiredPodMemoryRequests := new(inf.Dec).Mul(podMemoryRequests, memoryCurrentToTargetRatio)
	minMemory := state.Constraints.MinResources.Memory
	maxMemory := state.Constraints.MaxResources.Memory

	if desiredPodMemoryRequests.Cmp(minMemory) < 0 {
		desiredPodMemoryRequests = minMemory
	}

	if desiredPodMemoryRequests.Cmp(maxMemory) > 0 {
		desiredPodMemoryRequests = maxMemory
	}

	if podMemoryRequests.Cmp(zero) == 0 {
		return nil, fmt.Errorf("memory requests cannot be zero")
	}

	desiredPodMemoryLimits := new(inf.Dec).Mul(desiredPodMemoryRequests, new(inf.Dec).QuoRound(podMemoryLimits, podMemoryRequests, 8, inf.RoundHalfUp))

	if desiredPodMemoryLimits.Cmp(minMemory) < 0 {
		desiredPodMemoryLimits = minMemory
	}

	if desiredPodMemoryLimits.Cmp(maxMemory) > 0 {
		desiredPodMemoryLimits = maxMemory
	}

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

	for name, resources := range containerResources {
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
		Replicas:           state.Replicas,
		ContainerResources: containerResources,
	}, nil
}
