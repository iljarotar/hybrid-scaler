package reinforcement

import "github.com/iljarotar/hybrid-scaler/internal/strategy"

type ScalingAgent struct {
}

func (a *ScalingAgent) MakeDecision(state strategy.State) strategy.ScalingDecision {
	decision := strategy.ScalingDecision{}

	return decision
}
