package scaling

import "github.com/iljarotar/hybrid-scaler/internal/strategy"

// Recommends new resource requests and limits and a lower number of replicas
func HybridHorizontalUp(state *strategy.State) (*strategy.ScalingDecision, error) {
	// TODO: implement

	return &strategy.ScalingDecision{}, nil
}

// Recommends new resource requests and limits and a higher number of replicas
func HybridHorizontalDown(state *strategy.State) (*strategy.ScalingDecision, error) {
	// TODO: implement

	return &strategy.ScalingDecision{}, nil
}
