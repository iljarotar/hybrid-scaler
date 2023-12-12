package reinforcement

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"math"

	"github.com/go-logr/logr"
	"gopkg.in/inf.v0"
)

type QLearning struct {
	cpuCost, memoryCost, underprovisioningPenalty, alpha, gamma *inf.Dec
	allActions                                                  actions
	logger                                                      logr.Logger
}

func NewQLearning(cpuCost, memoryCost, underprovisioningPenalty, alpha, gamma *inf.Dec, possibleActions actions, logger logr.Logger) *QLearning {
	return &QLearning{
		logger:                   logger,
		allActions:               possibleActions,
		cpuCost:                  cpuCost,
		memoryCost:               memoryCost,
		underprovisioningPenalty: underprovisioningPenalty,
		alpha:                    alpha,
		gamma:                    gamma,
	}
}

type qTable map[stateName]qTableRow
type qTableRow map[action]*inf.Dec

type learningState struct {
	Table          qTable
	PreviousState  *state
	PreviousAction *action
}

var initialValue = inf.NewDec(0, 0)

func (l *QLearning) Update(currentState *state, currentAction *action, learningStateEncoded []byte) ([]byte, error) {
	ls, err := decodeToLearningState(learningStateEncoded)
	if err != nil {
		return nil, fmt.Errorf("cannot decode learning state, %w", err)
	}

	l.logger.Info("current learning state", "learning state", ls)

	table := ls.Table
	previousState := ls.PreviousState
	previousAction := ls.PreviousAction

	if previousState == nil || previousAction == nil {
		previousState = currentState
		previousAction = currentAction

		newLearningState := &learningState{
			Table:          make(qTable),
			PreviousState:  previousState,
			PreviousAction: previousAction,
		}

		encoded, err := encodeLearningState(newLearningState)
		if err != nil {
			return nil, fmt.Errorf("cannot encode learning state, %w", err)
		}
		return encoded, nil
	}

	if table == nil {
		table = make(qTable)
	}

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

	newValue, err := l.newQValue(currentValue, l.alpha, l.gamma, currentState, table)
	if err != nil {
		return nil, fmt.Errorf("cannot calculate new value for q table, %w", err)
	}
	table[previousState.Name][*previousAction] = newValue

	ls.Table = table
	ls.PreviousAction = currentAction
	ls.PreviousState = currentState

	encoded, err := encodeLearningState(ls)
	if err != nil {
		return nil, fmt.Errorf("cannot encode learning state, %w", err)
	}

	return encoded, nil
}

func (l *QLearning) GetGreedyActions(state stateName, learningState []byte) (actions, error) {
	ls, err := decodeToLearningState(learningState)
	if err != nil {
		return nil, fmt.Errorf("cannot decode learning state, %w", err)
	}
	table := ls.Table

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

func (l *QLearning) evaluateCost(s *state) (*inf.Dec, error) {
	zero := inf.NewDec(0, 0)

	if s.CpuTargetUtilization.Cmp(zero) == 0 {
		return nil, fmt.Errorf("cpu target utilization cannot be zero")
	}

	if s.MemoryTargetUtilization.Cmp(zero) == 0 {
		return nil, fmt.Errorf("memory target utilization cannot be zero")
	}

	replicas := inf.NewDec(int64(s.Replicas), 0)

	cpuCosts := new(inf.Dec).Mul(l.cpuCost, s.CpuRequests)
	memoryCosts := new(inf.Dec).Mul(l.memoryCost, s.MemoryRequests)

	cpuPenalty := inf.NewDec(0, 0)
	if s.CpuUtilization.Cmp(s.CpuTargetUtilization) > 0 {
		cpuUsage := new(inf.Dec).Mul(s.CpuRequests, s.CpuUtilization)
		targetCpuRequests := new(inf.Dec).QuoRound(cpuUsage, s.CpuTargetUtilization, 8, inf.RoundHalfUp)
		difference := new(inf.Dec).Add(targetCpuRequests, new(inf.Dec).Neg(s.CpuRequests))
		penalty := new(inf.Dec).Mul(l.cpuCost, l.underprovisioningPenalty)
		cpuPenalty = new(inf.Dec).Mul(difference, penalty)
	}

	memoryPenalty := inf.NewDec(0, 0)
	if s.MemoryUtilization.Cmp(s.MemoryTargetUtilization) > 0 {
		memoryUsage := new(inf.Dec).Mul(s.MemoryRequests, s.MemoryUtilization)
		targetMemoryRequests := new(inf.Dec).QuoRound(memoryUsage, s.MemoryTargetUtilization, 8, inf.RoundHalfUp)
		difference := new(inf.Dec).Add(targetMemoryRequests, new(inf.Dec).Neg(s.MemoryRequests))
		penalty := new(inf.Dec).Mul(l.memoryCost, l.underprovisioningPenalty)
		memoryPenalty = new(inf.Dec).Mul(difference, penalty)
	}

	podCpuCost := new(inf.Dec).Add(cpuCosts, cpuPenalty)
	podMemoryCost := new(inf.Dec).Add(memoryCosts, memoryPenalty)
	totalPodCost := new(inf.Dec).Add(podCpuCost, podMemoryCost)
	totalCost := new(inf.Dec).Mul(totalPodCost, replicas)

	return totalCost, nil
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

func decodeToLearningState(encoded []byte) (*learningState, error) {
	table := new(learningState)

	buffer := bytes.NewBuffer(encoded)
	decoder := gob.NewDecoder(buffer)

	err := decoder.Decode(table)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return table, nil
}

func encodeLearningState(s *learningState) ([]byte, error) {
	buffer := new(bytes.Buffer)
	encoder := gob.NewEncoder(buffer)

	err := encoder.Encode(s)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (l *QLearning) newQValue(currentValue, alpha, gamma *inf.Dec, s *state, table qTable) (*inf.Dec, error) {
	bestNextValue := bestActionValueInState(s.Name, table)
	discountedBestNextValue := new(inf.Dec).Mul(l.gamma, bestNextValue)
	currentNegative := new(inf.Dec).Neg(currentValue)

	cost, err := l.evaluateCost(s)
	if err != nil {
		return nil, err
	}

	newCostEstimate := new(inf.Dec).Add(cost, new(inf.Dec).Add(discountedBestNextValue, currentNegative))
	difference := new(inf.Dec).Mul(l.alpha, newCostEstimate)

	newValue := new(inf.Dec).Add(currentValue, difference)
	newValue.Round(newValue, 4, inf.RoundHalfUp)
	return newValue, nil
}
