package strategy

type ScalingStrategy interface {
	MakeDecision(state State) ScalingDecision
}

// State represents the current state
type State struct {
	Replicas           int
	ContainerResources map[string]Resources
	ContainerMetrics   map[string]ResourceUsage
}

// ScalingDecision represents the next desired state
type ScalingDecision struct {
	Replicas           int
	ContainerResources map[string]Resources
}

type Resources struct {
	Requests ResourcesList
	Limits   ResourcesList
}

type ResourcesList struct {
	CPU    float64
	Memory float64
}

type ResourceUsage struct {
	CPU    float64
	Memory float64
}
