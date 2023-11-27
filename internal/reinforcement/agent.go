package reinforcement

import (
	"github.com/iljarotar/hybrid-scaler/internal/strategy"
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
const percentageQuantum = 25

// stateName represents a state as a string of the form
// <replicas>_<cpu-usage>_<memory-usage>_<latency-threshold-exceeded in [0,1]>
type stateName string

type state struct {
	Name                     stateName
	Replicas                 uint32
	LatencyThresholdExceeded bool

	// CpuUsage in `percentageQuantum`% steps
	CpuUsage uint32

	// MemoryUsage in `percentageQuantum`% steps
	MemoryUsage uint32
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
	state := state{}

	// TODO: implement

	return state
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
