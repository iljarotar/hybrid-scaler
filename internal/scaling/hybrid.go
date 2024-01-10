package scaling

import (
	"fmt"

	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
)

// Recommends vertical and horizontal scaling
func Hybrid(s *strategy.State, cpuLimitsToRequestsRatio, memoryLimitsToRequestsRatio *inf.Dec) (*strategy.ScalingDecision, error) {
	horizontalRecommendation, err := calculateDesiredReplicas(s)
	if err != nil {
		return nil, err
	}

	currentReplicas := inf.NewDec(int64(s.Replicas), 0)
	difference := new(inf.Dec).Add(horizontalRecommendation, new(inf.Dec).Neg(currentReplicas))
	difference.QuoRound(difference, inf.NewDec(2, 0), 0, inf.RoundUp)

	desiredReplicas := new(inf.Dec).Add(currentReplicas, difference)
	minReplicas := inf.NewDec(int64(s.MinReplicas), 0)
	maxReplicas := inf.NewDec(int64(s.MaxReplicas), 0)
	limitedReplicas := limitValue(desiredReplicas, minReplicas, maxReplicas)

	replicas := DecToInt64(limitedReplicas)

	hypotheticalState := *s
	hypotheticalState.Replicas = int32(replicas)

	zero := inf.NewDec(0, 0)
	if limitedReplicas.Cmp(zero) == 0 {
		return nil, fmt.Errorf("attempting to scale to zero replicas, please provide min and max values for replicas to prevent this")
	}

	replicasRatio := new(inf.Dec).QuoRound(currentReplicas, limitedReplicas, 8, inf.RoundHalfUp)
	podCpuUsage := new(inf.Dec).Mul(s.PodMetrics.ResourceUsage.CPU, replicasRatio)
	podMemoryUsage := new(inf.Dec).Mul(s.PodMetrics.ResourceUsage.Memory, replicasRatio)

	hypotheticalState.PodMetrics.ResourceUsage.CPU = podCpuUsage
	hypotheticalState.PodMetrics.ResourceUsage.Memory = podMemoryUsage

	return Vertical(&hypotheticalState, cpuLimitsToRequestsRatio, memoryLimitsToRequestsRatio)
}
