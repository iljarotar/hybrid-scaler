package reinforcement

import "gopkg.in/inf.v0"

type QLearning struct {
	qTable
	CpuCost, MemoryCost, PerformancePenalty *inf.Dec
}

type qTable map[stateName]qTableRow

type qTableRow map[action]*inf.Dec

func (l *QLearning) Update(s *state, a action, alpha, gamma float64) error {
	// TODO: implement q-table update (see RLbook p.131)

	return nil
}

func (l *QLearning) GetGreedyActionsAmong(as actions) actions {
	actions := make(actions, 0)

	// TODO: implement

	return actions
}

func (l *QLearning) evaluateCost(s *state) *inf.Dec {
	replicas := inf.NewDec(int64(s.Replicas), 0)

	te := 0
	if s.LatencyThresholdExceeded {
		te = 1
	}

	thresholdExceeded := inf.NewDec(int64(te), 0)

	cpuCosts := new(inf.Dec).Mul(l.CpuCost, new(inf.Dec).Mul(s.CpuRequests, replicas))
	memoryCosts := new(inf.Dec).Mul(l.MemoryCost, new(inf.Dec).Mul(s.MemoryRequests, replicas))
	penalty := new(inf.Dec).Mul(l.PerformancePenalty, thresholdExceeded)

	return new(inf.Dec).Add(cpuCosts, new(inf.Dec).Add(memoryCosts, penalty))
}
