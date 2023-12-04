package reinforcement

import "gopkg.in/inf.v0"

type QLearning struct {
	qTable
	cpuCost, memoryCost, performancePenalty, alpha, gamma *inf.Dec
	allActions                                            actions
}

func NewQLearning(cpuCost, memoryCost, performancePenalty, alpha, gamma *inf.Dec, possibleActions actions) *QLearning {
	return &QLearning{
		qTable:             map[stateName]qTableRow{},
		allActions:         possibleActions,
		cpuCost:            cpuCost,
		memoryCost:         memoryCost,
		performancePenalty: performancePenalty,
		alpha:              alpha,
		gamma:              gamma,
	}
}

type qTable map[stateName]qTableRow
type qTableRow map[action]*inf.Dec

var initialValue = inf.NewDec(0, 0)

func (l *QLearning) Update(previousState, currentState *state, previousAction *action) error {
	if previousAction == nil || previousState == nil {
		return nil
	}

	if _, ok := l.qTable[previousState.Name]; !ok {
		l.initializeRow(previousState.Name)
	}

	var currentValue *inf.Dec

	_, ok := l.qTable[previousState.Name][*previousAction]
	if ok {
		currentValue = l.qTable[previousState.Name][*previousAction]
	} else {
		currentValue = initialValue
	}

	bestNextValue := l.bestActionValueInState(currentState)
	discountedBestNextValue := new(inf.Dec).Mul(l.gamma, bestNextValue)
	currentNegative := new(inf.Dec).Neg(currentValue)
	cost := l.evaluateCost(currentState)
	newCostEstimate := new(inf.Dec).Add(cost, new(inf.Dec).Add(discountedBestNextValue, currentNegative))
	difference := new(inf.Dec).Mul(l.alpha, newCostEstimate)

	newValue := new(inf.Dec).Add(currentValue, difference)

	l.qTable[previousState.Name][*previousAction] = newValue

	return nil
}

func (l *QLearning) GetGreedyActions(s *state) actions {
	greedyActions := make(actions, 0)
	bestValue := l.bestActionValueInState(s)

	if _, ok := l.qTable[s.Name]; !ok {
		return l.allActions
	}

	row := l.qTable[s.Name]
	for _, a := range l.allActions {
		value, ok := row[a]
		if ok && value.Cmp(bestValue) <= 0 {
			greedyActions = append(greedyActions, a)
		}
	}

	return greedyActions
}

func (l *QLearning) evaluateCost(s *state) *inf.Dec {
	replicas := inf.NewDec(int64(s.Replicas), 0)

	te := 0
	if s.LatencyThresholdExceeded {
		te = 1
	}

	thresholdExceeded := inf.NewDec(int64(te), 0)

	cpuCosts := new(inf.Dec).Mul(l.cpuCost, new(inf.Dec).Mul(s.CpuRequests, replicas))
	memoryCosts := new(inf.Dec).Mul(l.memoryCost, new(inf.Dec).Mul(s.MemoryRequests, replicas))
	penalty := new(inf.Dec).Mul(l.performancePenalty, thresholdExceeded)

	return new(inf.Dec).Add(cpuCosts, new(inf.Dec).Add(memoryCosts, penalty))
}

func (l *QLearning) bestActionValueInState(s *state) *inf.Dec {
	minCost := initialValue

	if _, ok := l.qTable[s.Name]; !ok {
		return minCost
	}

	row := l.qTable[s.Name]
	for _, value := range row {
		if value.Cmp(minCost) < 0 {
			minCost = value
		}
	}

	return minCost
}

func (l *QLearning) initializeRow(name stateName) {
	row := make(map[action]*inf.Dec)

	for _, a := range l.allActions {
		row[a] = initialValue
	}

	l.qTable[name] = row
}
