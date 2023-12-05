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

type learningMethod interface {
	Update(previousState, currentState *state, previousAction *action) error
	GetGreedyActions(s *state) actions
}

var (
	// percentageQuantum is used to discretize the resource usage very roughly
	percentageQuantum = inf.NewDec(25, 0)

	cpuCost            = inf.NewDec(0, 0)
	memoryCost         = inf.NewDec(0, 0)
	performancePenalty = inf.NewDec(0, 0)

	alpha = inf.NewDec(0, 0)
	gamma = inf.NewDec(0, 0)
)

// stateName represents a state as a string of the form
// <replicas>_<cpu-usage>_<memory-usage>_<latency-threshold-exceeded>
type stateName string

type state struct {
	Name                        stateName
	Replicas                    int32
	LatencyThresholdExceeded    bool
	CpuRequests, MemoryRequests *inf.Dec
	// cpu and memory usage in [percentageQuantum]% steps
	CpuUsage, MemoryUsage int64
}

type scalingAgent struct {
	method          learningMethod
	epsilon         float64
	previousAction  *action
	possibleActions actions
	previousState   *state
}

func NewScalingAgent() *scalingAgent {
	possibleActions := []action{actionNone, actionHorizontal, actionVertical, actionHybrid, actionHybridInverse}
	method := NewQLearning(cpuCost, memoryCost, performancePenalty, alpha, gamma, possibleActions)

	return &scalingAgent{
		method:          method,
		epsilon:         0,
		possibleActions: possibleActions,
	}
}

func (a *scalingAgent) MakeDecision(state *strategy.State) (*strategy.ScalingDecision, error) {
	s, err := convertState(state)
	if err != nil {
		return nil, err
	}

	err = a.method.Update(a.previousState, s, a.previousAction)
	if err != nil {
		return nil, err
	}

	greedy, err := iAmGreedy(a.epsilon)
	if err != nil {
		return nil, fmt.Errorf("cannot decide which action to choose, %w", err)
	}

	possibleActions := a.possibleActions

	if greedy {
		possibleActions = a.method.GetGreedyActions(s)
	}

	action, err := getRandomActionFrom(possibleActions)
	if err != nil {
		return nil, fmt.Errorf("cannot decide which action to choose, %w", err)
	}

	decision, err := convertAction(*action, state)
	if err != nil {
		return nil, err
	}

	a.previousAction = action
	a.previousState = s

	return decision, nil
}

func convertState(s *strategy.State) (*state, error) {
	cpuUsageInPercent := inf.NewDec(0, 0)
	memoryUsageInPercent := inf.NewDec(0, 0)
	zero := inf.NewDec(0, 0)

	podCpuLimits := s.PodMetrics.Limits.CPU
	podCpuUsage := s.PodMetrics.ResourceUsage.CPU
	podMemoryLimits := s.PodMetrics.Limits.Memory
	podMemoryUsage := s.PodMetrics.ResourceUsage.Memory

	if podCpuLimits.Cmp(zero) == 0 {
		return nil, fmt.Errorf("cpu limits cannot be zero")
	}
	cpuUsageInPercent.QuoRound(podCpuUsage, podCpuLimits, 8, inf.RoundHalfUp)

	if podMemoryLimits.Cmp(zero) == 0 {
		return nil, fmt.Errorf("memory limits cannot be zero")
	}
	memoryUsageInPercent.QuoRound(podMemoryUsage, podMemoryLimits, 8, inf.RoundHalfUp)

	cpuUsageQuantized := quantizePercentage(cpuUsageInPercent, percentageQuantum)
	memoryUsageQuantized := quantizePercentage(memoryUsageInPercent, percentageQuantum)

	// FIXME: get from metrics
	latencyThresholdExceeded := false

	name := fmt.Sprintf("%d_%d_%d_%v", s.Replicas, cpuUsageQuantized, memoryUsageQuantized, latencyThresholdExceeded)

	return &state{
		Name:                     stateName(name),
		Replicas:                 s.Replicas,
		LatencyThresholdExceeded: latencyThresholdExceeded,
		CpuRequests:              s.PodMetrics.Requests.CPU,
		MemoryRequests:           s.PodMetrics.Requests.Memory,
		CpuUsage:                 cpuUsageQuantized,
		MemoryUsage:              memoryUsageQuantized,
	}, nil
}

func convertAction(a action, s *strategy.State) (*strategy.ScalingDecision, error) {
	noChange := &strategy.ScalingDecision{
		Replicas:           s.Replicas,
		ContainerResources: s.ContainerResources,
	}

	switch a {
	case actionNone:
		return noChange, nil
	case actionHorizontal:
		return scaling.Horizontal(s)
	case actionVertical:
		return scaling.Vertical(s)
	case actionHybrid:
		return scaling.Hybrid(s)
	case actionHybridInverse:
		return scaling.HybridInverse(s)
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
