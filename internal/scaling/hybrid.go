package scaling

import (
	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
)

// Recommends vertical and horizontal scaling
func Hybrid(state *strategy.State) (*strategy.ScalingDecision, error) {
	horizontalRecommendation, err := calculateDesiredReplicas(state)
	if err != nil {
		return nil, err
	}

	currentReplicas := inf.NewDec(int64(state.Replicas), 0)
	difference := new(inf.Dec).Add(horizontalRecommendation, new(inf.Dec).Neg(currentReplicas))
	difference.QuoRound(difference, inf.NewDec(2, 0), 0, inf.RoundUp)

	desiredReplicas := new(inf.Dec).Add(currentReplicas, difference)
	minReplicas := inf.NewDec(int64(state.MinReplicas), 0)
	maxReplicas := inf.NewDec(int64(state.MaxReplicas), 0)
	limitedReplicas := limitScalingValue(desiredReplicas, minReplicas, maxReplicas)

	replicas := DecToInt64(limitedReplicas)

	hypotheticalState := *state
	hypotheticalState.Replicas = int32(replicas)

	return Vertical(&hypotheticalState)
}

// Recommends vertical and horizontal scaling in oposite directions
func HybridInverse(state *strategy.State) (*strategy.ScalingDecision, error) {
	horizontalRecommendation, err := calculateDesiredReplicas(state)
	if err != nil {
		return nil, err
	}

	currentReplicas := inf.NewDec(int64(state.Replicas), 0)
	difference := new(inf.Dec).Add(horizontalRecommendation, new(inf.Dec).Neg(currentReplicas))
	difference.QuoRound(difference, inf.NewDec(2, 0), 0, inf.RoundDown)
	difference.Neg(difference)

	desiredReplicas := new(inf.Dec).Add(currentReplicas, difference)
	minReplicas := inf.NewDec(int64(state.MinReplicas), 0)
	maxReplicas := inf.NewDec(int64(state.MaxReplicas), 0)
	limitedReplicas := limitScalingValue(desiredReplicas, minReplicas, maxReplicas)

	replicas := DecToInt64(limitedReplicas)

	hypotheticalState := *state
	hypotheticalState.Replicas = int32(replicas)

	return Vertical(&hypotheticalState)
}
