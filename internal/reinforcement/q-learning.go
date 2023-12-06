package reinforcement

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"

	"gopkg.in/inf.v0"
)

type QLearning struct {
	cpuCost, memoryCost, performancePenalty, alpha, gamma *inf.Dec
	allActions                                            actions
}

func NewQLearning(cpuCost, memoryCost, performancePenalty, alpha, gamma *inf.Dec, possibleActions actions) *QLearning {
	return &QLearning{
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

func (l *QLearning) Update(previousState, currentState *state, previousAction *action, learningState []byte) ([]byte, error) {
	if previousAction == nil || previousState == nil {
		return learningState, nil
	}

	decoded, err := decodeToQTable(learningState)
	if err != nil {
		return nil, fmt.Errorf("cannot decode learning state, %w", err)
	}
	table := *decoded

	if _, ok := table[previousState.Name]; !ok {
		l.initializeRow(previousState.Name, table)
	}

	var currentValue *inf.Dec
	_, ok := table[previousState.Name][*previousAction]
	if ok {
		currentValue = table[previousState.Name][*previousAction]
	} else {
		currentValue = initialValue
	}

	newValue := l.newQValue(currentValue, l.alpha, l.gamma, currentState, table)
	table[previousState.Name][*previousAction] = newValue

	encoded, err := encodeQTable(table)
	if err != nil {
		return nil, fmt.Errorf("cannot encode learning state, %w", err)
	}

	return encoded, nil
}

func (l *QLearning) GetGreedyActions(state stateName, learningState []byte) (actions, error) {
	decoded, err := decodeToQTable(learningState)
	if err != nil {
		return nil, fmt.Errorf("cannot decode learning state, %w", err)
	}
	table := *decoded

	greedyActions := make(actions, 0)
	bestValue := bestActionValueInState(state, table)

	if _, ok := table[state]; !ok {
		return l.allActions, nil
	}

	row := table[state]
	for _, a := range l.allActions {
		value, ok := row[a]
		if ok && value.Cmp(bestValue) <= 0 {
			greedyActions = append(greedyActions, a)
		}
	}

	return greedyActions, nil
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

func bestActionValueInState(state stateName, table qTable) *inf.Dec {
	minCost := inf.NewDec(math.MaxInt64, 0)

	if _, ok := table[state]; !ok {
		return initialValue
	}

	row := table[state]
	if len(row) == 0 {
		return initialValue
	}

	for _, value := range row {
		if value.Cmp(minCost) < 0 {
			minCost = value
		}
	}

	return minCost
}

func (l *QLearning) initializeRow(name stateName, table qTable) {
	row := make(map[action]*inf.Dec)

	for _, a := range l.allActions {
		row[a] = initialValue
	}

	table[name] = row
}

func decodeToQTable(encoded []byte) (*qTable, error) {
	table := new(qTable)

	buffer := bytes.NewBuffer(encoded)
	decoder := gob.NewDecoder(buffer)

	err := decoder.Decode(table)
	if err != nil {
		return nil, err
	}

	return table, nil
}

func encodeQTable(table qTable) ([]byte, error) {
	buffer := new(bytes.Buffer)
	encoder := gob.NewEncoder(buffer)

	err := encoder.Encode(table)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (l *QLearning) newQValue(currentValue, alpha, gamma *inf.Dec, s *state, table qTable) *inf.Dec {
	bestNextValue := bestActionValueInState(s.Name, table)
	discountedBestNextValue := new(inf.Dec).Mul(l.gamma, bestNextValue)
	currentNegative := new(inf.Dec).Neg(currentValue)
	cost := l.evaluateCost(s)
	newCostEstimate := new(inf.Dec).Add(cost, new(inf.Dec).Add(discountedBestNextValue, currentNegative))
	difference := new(inf.Dec).Mul(l.alpha, newCostEstimate)

	newValue := new(inf.Dec).Add(currentValue, difference)
	return newValue
}
