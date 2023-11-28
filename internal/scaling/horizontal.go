package scaling

import "github.com/iljarotar/hybrid-scaler/internal/strategy"

// Recommends new number of replicas
func Horizontal(state *strategy.State) *strategy.ScalingDecision {
	// TODO: implement
	// while agent compares current usage with limits, here the request utilization should be used

	return &strategy.ScalingDecision{}
}
