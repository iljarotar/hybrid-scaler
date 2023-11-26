package reinforcement

import (
	"github.com/iljarotar/hybrid-scaler/internal/strategy"
)

type quantum float64

const (
	// usageQuantum is a 5% step
	usageQuantum quantum = 0.05

	// cpuQuantum is a 1m step
	cpuQuantum quantum = 0.001

	// memoryQuantum is a 1k step
	memoryQuantum quantum = 1000
)

type scalingDirection string

const (
	scalingDirectionUp    scalingDirection = "UP"
	scalingDirectionDown  scalingDirection = "DOWN"
	scalingDirectionStays scalingDirection = "STAYS"
)

type scalingResource string

const (
	scalingResourceMemory   scalingResource = "MEMORY"
	scalingResourceCPU      scalingResource = "CPU"
	scalingResourceReplicas scalingResource = "REPLICAS"
)

type actionName string

type action struct {
	Name      actionName
	decisions map[scalingResource]scalingDirection
}

type actions []action

type learningMethod interface {
	Update(s state, a action, alpha, gamma float64) error
	GetGreedyActionsAmong(actions) actions
}

type stateName string

type state struct {
	Name               stateName
	Replicas           uint32
	ContainerResources resources
	ContainerMetrics   resourcesList
}

type resources struct {
	Requests resourcesList
	Limits   resourcesList
}

// represented in k*quantum, where k is an integer and quantum is a discretizing step
// use usageQuantum for cpu and memory metrics
// use cpuQuantum and memoryQuantum for cpu and memory resources respectively
type resourcesList struct {
	CPU    uint32
	Memory uint32
}

type scalingAgent struct {
	method                learningMethod
	epsilon, alpha, gamma float64
	previousAction        action
}

func NewScalingAgent() *scalingAgent {
	method := QLearning{}

	return &scalingAgent{
		method:         &method,
		epsilon:        0,
		alpha:          0,
		gamma:          0,
		previousAction: action{},
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
	action := action{}

	// TODO: implement

	return action
}

func iAmGreedy(epsilon float64) bool {
	var greedy bool

	// TODO: implement

	return greedy
}
