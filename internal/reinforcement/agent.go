package reinforcement

import (
	"fmt"

	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
)

type action string

const (
	actionNone                   action = "NONE"
	actionVertical               action = "VERTICAL"
	actionHorizontal             action = "HORIZONAL"
	actionVerticalHorizontalUp   action = "VERTICAL_HORIZONTAL_UP"
	actionnVerticalHorizontaDown action = "VERTICAL_HORIZONTAL_DOWN"
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
	decision := convertAction(action)

	a.previousAction = action

	return decision, nil
}

func convertState(s strategy.State) state {
	podCpuUsage := inf.NewDec(0, 0)
	podMemoryUsage := inf.NewDec(0, 0)

	// FIXME: what if limits are not specified
	podCpuLimits := inf.NewDec(0, 0)
	podMemoryLimits := inf.NewDec(0, 0)

	for _, metrics := range s.ContainerMetricsMap {
		cpuUsage := metrics.ResourceUsage.CPU
		memoryUsage := metrics.ResourceUsage.Memory
		podCpuUsage.Add(podCpuUsage, cpuUsage)
		memoryUsage.Add(podMemoryUsage, memoryUsage)

		cpuLimits := metrics.Limits.CPU
		memoryLimits := metrics.Limits.Memory
		podCpuLimits.Add(podCpuLimits, cpuLimits)
		podMemoryLimits.Add(podMemoryLimits, memoryLimits)
	}

	cpuUsageInPercent := inf.NewDec(0, 0)
	memoryUsageInPercent := inf.NewDec(0, 0)
	zero := inf.NewDec(0, 0)

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

func convertAction(action) *strategy.ScalingDecision {
	decision := &strategy.ScalingDecision{
		Replicas:           0,
		ContainerResources: map[string]strategy.Resources{},
	}

	// TODO: implement

	return decision
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
