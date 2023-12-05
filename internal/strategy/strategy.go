package strategy

import "gopkg.in/inf.v0"

type ScalingStrategy interface {
	MakeDecision(state *State, learningState []byte) (*ScalingDecision, []byte, error)
}

// State represents the current state
type State struct {
	Replicas int32
	ContainerResources
	Constraints
	PodMetrics        Metrics
	TargetUtilization ResourcesList
}

// ScalingDecision represents the next desired state
type ScalingDecision struct {
	Replicas int32
	ContainerResources
}

// Metrics stores a pod's allocated and average used resources
type Metrics struct {
	ResourceUsage ResourcesList
	Resources
}

// Resources stores a container's resource requests and limits
type Resources struct {
	Requests ResourcesList
	Limits   ResourcesList
}

// ResourcesList stores cpu and memory quantities as decimal values
type ResourcesList struct {
	CPU    *inf.Dec
	Memory *inf.Dec
}

// ContainerResources maps a container's name to its allocated resources
type ContainerResources map[string]Resources

// Constraints represents the scaling constraints
type Constraints struct {
	MinReplicas  int32
	MaxReplicas  int32
	MinResources ResourcesList
	MaxResources ResourcesList
}
