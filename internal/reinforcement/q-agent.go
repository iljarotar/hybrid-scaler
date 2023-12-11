package reinforcement

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/iljarotar/hybrid-scaler/internal/scaling"
	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
)

type action string

const (
	actionNone          action = "NONE"
	actionVertical      action = "VERTICAL"
	actionHorizontal    action = "HORIZONAL"
	actionHybrid        action = "HYBRID"
	actionHybridInverse action = "HYBRID_INVERSE"
)

type actions []action

var (
	// percentageQuantum is used to discretize the resource usage very roughly
	percentageQuantum = inf.NewDec(25, 0)
	allActions        = []action{actionNone, actionHorizontal, actionVertical, actionHybrid, actionHybridInverse}
)

// stateName represents a state as a string of the form
// <replicas>_<cpu-requests>_<memory-requests>_<cpu-utilization>_<memory-utilization>
type stateName string

type state struct {
	Name     stateName
	Replicas int32

	CpuRequests, MemoryRequests                   *inf.Dec
	CpuUtilization, MemoryUtilization             *inf.Dec
	CpuTargetUtilization, MemoryTargetUtilization *inf.Dec
}

type qAgent struct {
	QLearning
	epsilon         float64
	previousAction  *action
	possibleActions actions
	previousState   *state
	// these ratios are needed in case one of requests or limits at some point hits a limit and thereby changes the initial ratio
	cpuLimitsToRequestsRatio, memoryLimitsToRequestsRatio *inf.Dec
}

func NewQAgent(cpuCost, memoryCost, underprovisioningPenalty, alpha, gamma *inf.Dec) *qAgent {
	possibleActions := allActions
	qLearning := NewQLearning(cpuCost, memoryCost, underprovisioningPenalty, alpha, gamma, possibleActions)

	return &qAgent{
		QLearning:       *qLearning,
		epsilon:         0,
		possibleActions: possibleActions,
	}
}

func (a *qAgent) MakeDecision(state *strategy.State, learningState []byte) (*strategy.ScalingDecision, []byte, error) {
	if a.cpuLimitsToRequestsRatio == nil || a.memoryLimitsToRequestsRatio == nil {
		err := a.initializeLimitsToRequestRatios(state)
		if err != nil {
			return nil, nil, err
		}
	}

	s, err := convertState(state)
	if err != nil {
		return nil, nil, err
	}

	newLearningState, err := a.Update(a.previousState, s, a.previousAction, learningState)
	if err != nil {
		return nil, nil, err
	}

	greedy, err := iAmGreedy(a.epsilon)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot decide which action to choose, %w", err)
	}

	possibleActions := a.possibleActions

	if greedy {
		possibleActions, err = a.GetGreedyActions(s.Name, learningState)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot get greedy actions, %w", err)
		}
	}

	action, err := getRandomActionFrom(possibleActions)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot decide which action to choose, %w", err)
	}

	decision, err := a.convertAction(*action, state)
	if err != nil {
		return nil, nil, err
	}

	a.previousAction = action
	a.previousState = s

	return decision, newLearningState, nil
}

func convertState(s *strategy.State) (*state, error) {
	zero := inf.NewDec(0, 0)
	hundred := inf.NewDec(100, 0)

	podCpuLimits := s.PodMetrics.Limits.CPU
	podMemoryLimits := s.PodMetrics.Limits.Memory
	podCpuUsage := s.PodMetrics.ResourceUsage.CPU
	podMemoryUsage := s.PodMetrics.ResourceUsage.Memory

	podCpuRequests := s.PodMetrics.Requests.CPU
	if podCpuRequests.Cmp(zero) == 0 {
		return nil, fmt.Errorf("cpu requests cannot be zero")
	}

	podMemoryRequests := s.PodMetrics.Requests.Memory
	if podMemoryRequests.Cmp(zero) == 0 {
		return nil, fmt.Errorf("memory requests cannot be zero")
	}

	maxCpu := s.Constraints.MaxResources.CPU
	if maxCpu.Cmp(zero) == 0 {
		return nil, fmt.Errorf("max cpu cannot be zero")
	}

	maxMemory := s.Constraints.MaxResources.Memory
	if maxMemory.Cmp(zero) == 0 {
		return nil, fmt.Errorf("max memory cannot be zero")
	}

	cpuTargetUtilization := s.TargetUtilization.CPU
	if cpuTargetUtilization.Cmp(zero) == 0 {
		return nil, fmt.Errorf("cpu target utilization cannot be zero")
	}

	memoryTargetUtilization := s.TargetUtilization.Memory
	if memoryTargetUtilization.Cmp(zero) == 0 {
		return nil, fmt.Errorf("memory target utilization cannot be zero")
	}

	cpuUsageInPercent := new(inf.Dec).QuoRound(podCpuUsage, podCpuRequests, 8, inf.RoundHalfUp)
	memoryUsageInPercent := new(inf.Dec).QuoRound(podMemoryUsage, podMemoryRequests, 8, inf.RoundHalfUp)
	cpuUtilizationRatio := new(inf.Dec).QuoRound(cpuUsageInPercent, cpuTargetUtilization, 8, inf.RoundHalfUp)
	memoryUtilizationRatio := new(inf.Dec).QuoRound(memoryUsageInPercent, memoryTargetUtilization, 8, inf.RoundHalfUp)
	cpuLimitsInPercentOfMax := new(inf.Dec).QuoRound(podCpuLimits, maxCpu, 8, inf.RoundHalfUp)
	memoryLimitsInPercentOfMax := new(inf.Dec).QuoRound(podMemoryLimits, maxMemory, 8, inf.RoundHalfUp)

	cpuUtilizationRatioQuantized := quantizePercentage(new(inf.Dec).Mul(cpuUtilizationRatio, hundred), percentageQuantum)
	if cpuUtilizationRatioQuantized > 100 {
		cpuUtilizationRatioQuantized = 100
	}

	memoryUtilizationRatioQuantized := quantizePercentage(new(inf.Dec).Mul(memoryUtilizationRatio, hundred), percentageQuantum)
	if memoryUtilizationRatioQuantized > 100 {
		memoryUtilizationRatioQuantized = 100
	}

	cpuLimitsQuantized := quantizePercentage(new(inf.Dec).Mul(cpuLimitsInPercentOfMax, hundred), percentageQuantum)
	if cpuLimitsQuantized > 100 {
		cpuLimitsQuantized = 100
	}

	memoryLimitsQuantized := quantizePercentage(new(inf.Dec).Mul(memoryLimitsInPercentOfMax, hundred), percentageQuantum)
	if memoryLimitsQuantized > 100 {
		memoryLimitsQuantized = 100
	}

	name := fmt.Sprintf("%d_%d_%d_%d_%d", s.Replicas, cpuLimitsQuantized, memoryLimitsQuantized, cpuUtilizationRatioQuantized, memoryUtilizationRatioQuantized)

	return &state{
		Name:                    stateName(name),
		Replicas:                s.Replicas,
		CpuRequests:             s.PodMetrics.Requests.CPU,
		MemoryRequests:          s.PodMetrics.Requests.Memory,
		CpuUtilization:          cpuUsageInPercent,
		MemoryUtilization:       memoryUsageInPercent,
		CpuTargetUtilization:    cpuTargetUtilization,
		MemoryTargetUtilization: memoryTargetUtilization,
	}, nil
}

func (a *qAgent) convertAction(chosenAction action, s *strategy.State) (*strategy.ScalingDecision, error) {
	noChange := &strategy.ScalingDecision{
		Replicas:           s.Replicas,
		ContainerResources: s.ContainerResources,
	}

	switch chosenAction {
	case actionNone:
		return noChange, nil
	case actionHorizontal:
		return scaling.Horizontal(s)
	case actionVertical:
		return scaling.Vertical(s, a.cpuLimitsToRequestsRatio, a.memoryLimitsToRequestsRatio)
	case actionHybrid:
		return scaling.Hybrid(s, a.cpuLimitsToRequestsRatio, a.memoryLimitsToRequestsRatio)
	case actionHybridInverse:
		return scaling.HybridInverse(s, a.cpuLimitsToRequestsRatio, a.memoryLimitsToRequestsRatio)
	default:
		return noChange, nil
	}
}

func getRandomActionFrom(as actions) (*action, error) {
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(as))))
	if err != nil {
		return nil, err
	}

	idx := int(index.Int64())
	a := as[idx]

	return &a, nil
}

func iAmGreedy(epsilon float64) (bool, error) {
	random, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		return false, err
	}

	val := float64(random.Int64())

	return val >= epsilon*100, nil
}

func quantizePercentage(value, quantum *inf.Dec) int64 {
	quantity := new(inf.Dec).QuoRound(value, quantum, 0, inf.RoundDown)
	quantized := new(inf.Dec).Mul(quantity, quantum)
	return scaling.DecToInt64(quantized)
}

func (a *qAgent) initializeLimitsToRequestRatios(s *strategy.State) error {
	zero := inf.NewDec(0, 0)

	if s.PodMetrics.Requests.CPU.Cmp(zero) == 0 {
		return fmt.Errorf("cpu requests cannot be zero")
	}

	if s.PodMetrics.Requests.Memory.Cmp(zero) == 0 {
		return fmt.Errorf("memory requests cannot be zero")
	}

	a.cpuLimitsToRequestsRatio = new(inf.Dec).QuoRound(s.PodMetrics.Limits.CPU, s.PodMetrics.Requests.CPU, 8, inf.RoundHalfUp)
	a.memoryLimitsToRequestsRatio = new(inf.Dec).QuoRound(s.PodMetrics.Limits.Memory, s.PodMetrics.Requests.Memory, 8, inf.RoundHalfUp)

	return nil
}
