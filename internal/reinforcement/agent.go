package reinforcement

import (
	"fmt"

	"github.com/iljarotar/hybrid-scaler/internal/scaling"
	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
)

type action string

const (
	actionNone                  action = "NONE"
	actionVertical              action = "VERTICAL"
	actionHorizontal            action = "HORIZONAL"
	actionVerticalHorizontalUp  action = "VERTICAL_HORIZONTAL_UP"
	actionVerticalHorizontaDown action = "VERTICAL_HORIZONTAL_DOWN"
)

type actions []action

type learningMethod interface {
	Update(s state, a action, alpha, gamma float64) error
	GetGreedyActionsAmong(actions) actions
}

// percentageQuantum is used to discretize the resource usage very roughly
var percentageQuantum = inf.NewDec(25, 0)

// stateName represents a state as a string of the form
// <replicas>_<cpu-usage>_<memory-usage>_<latency-threshold-exceeded>
type stateName string

type state struct {
	Name                     stateName
	Replicas                 int32
	LatencyThresholdExceeded bool

	// CpuUsage of pod in `percentageQuantum`% steps
	CpuUsage int64

	// MemoryUsage of pod in `percentageQuantum`% steps
	MemoryUsage int64
}

type scalingAgent struct {
	method                learningMethod
	epsilon, alpha, gamma float64
	previousAction        action
}

func NewScalingAgent() *scalingAgent {
	method := QLearning{}
	var a action

	return &scalingAgent{
		method:         &method,
		epsilon:        0,
		alpha:          0,
		gamma:          0,
		previousAction: a,
	}
}

func (a *scalingAgent) MakeDecision(state *strategy.State) (*strategy.ScalingDecision, error) {
	s := convertState(*state)

	err := a.method.Update(s, a.previousAction, a.alpha, a.gamma)
	if err != nil {
		return nil, err
	}

	actions := getPossibleActionsForState(s)
	if iAmGreedy(a.epsilon) {
		actions = a.method.GetGreedyActionsAmong(actions)
	}

	action := getRandomActionFrom(actions)
	decision := convertAction(action, state)

	a.previousAction = action

	return decision, nil
}

func convertState(s strategy.State) state {
	cpuUsageInPercent := inf.NewDec(0, 0)
	memoryUsageInPercent := inf.NewDec(0, 0)
	zero := inf.NewDec(0, 0)

	// TODO: put percentage calculation into scaling package to reuse
	podCpuLimits := s.PodMetrics.Limits.CPU
	podCpuUsage := s.PodMetrics.ResourceUsage.CPU
	podMemoryLimits := s.PodMetrics.Limits.Memory
	podMemoryUsage := s.PodMetrics.ResourceUsage.Memory

	if podCpuLimits.Cmp(zero) != 0 {
		cpuUsageInPercent.QuoRound(podCpuUsage, podCpuLimits, 8, inf.RoundHalfUp)
	}

	if podMemoryLimits.Cmp(zero) != 0 {
		memoryUsageInPercent.QuoRound(podMemoryUsage, podMemoryLimits, 8, inf.RoundHalfUp)
	}

	cpuUsageQuantized := quantizePercentage(cpuUsageInPercent, percentageQuantum)
	memoryUsageQuantized := quantizePercentage(memoryUsageInPercent, percentageQuantum)

	// FIXME: get from metrics
	latencyThresholdExceeded := false

	name := fmt.Sprintf("%d_%d_%d_%v", s.Replicas, cpuUsageQuantized, memoryUsageQuantized, latencyThresholdExceeded)

	return state{
		Name:                     stateName(name),
		Replicas:                 s.Replicas,
		LatencyThresholdExceeded: latencyThresholdExceeded,
		CpuUsage:                 cpuUsageQuantized,
		MemoryUsage:              memoryUsageQuantized,
	}
}

func convertAction(a action, s *strategy.State) *strategy.ScalingDecision {
	containerResources := make(strategy.ContainerResources)

	for name, metrics := range s.ContainerMetrics {
		containerResources[name] = metrics.Resources
	}

	noChange := &strategy.ScalingDecision{
		Replicas:           s.Replicas,
		ContainerResources: containerResources,
	}

	switch a {
	case actionNone:
		return noChange
	case actionHorizontal:
		return scaling.Horizontal(s)
	case actionVertical:
		return scaling.Vertical(s)
	case actionVerticalHorizontalUp:
		return scaling.HybridHorizontalUp(s)
	case actionVerticalHorizontaDown:
		return scaling.HybridHorizontalDown(s)
	default:
		return noChange
	}
}

func getPossibleActionsForState(state) actions {
	actions := make(actions, 0)

	// TODO: implement

	return actions
}

func getRandomActionFrom(actions) action {
	var a action

	// TODO: implement

	return a
}

func iAmGreedy(epsilon float64) bool {
	var greedy bool

	// TODO: implement

	return greedy
}

func quantizePercentage(value, quantum *inf.Dec) int64 {
	quantity := new(inf.Dec).QuoRound(value, quantum, 0, inf.RoundDown)
	quantized := new(inf.Dec).Mul(quantity, quantum)

	return quantized.UnscaledBig().Int64()
}
