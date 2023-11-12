package reinforcement

import (
	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
)

type ScalingAgent struct {
}

func (a *ScalingAgent) MakeDecision(state *strategy.State) *strategy.ScalingDecision {
	containerResources := make(strategy.ContainerResources)

	for name := range state.ContainerResources {
		containerResources[name] = strategy.Resources{
			Requests: strategy.ResourcesList{
				CPU:    inf.NewDec(340, 3),
				Memory: &inf.Dec{},
			},
			Limits: strategy.ResourcesList{
				CPU:    inf.NewDec(475, 3),
				Memory: &inf.Dec{},
			},
		}
	}

	decision := &strategy.ScalingDecision{
		Replicas:           2,
		ContainerResources: containerResources,
	}

	return decision
}
