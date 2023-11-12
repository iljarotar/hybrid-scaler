package strategy

import "gopkg.in/inf.v0"

type ScalingStrategy interface {
	MakeDecision(state *State) *ScalingDecision
}

// State represents the current state
type State struct {
	Replicas int32
	ContainerResources
	ContainerMetrics []ContainerMetrics
}

// ScalingDecision represents the next desired state
type ScalingDecision struct {
	Replicas int32
	ContainerResources
}

type ContainerMetrics struct {
	Name string
	ResourceUsage
}

type Resources struct {
	Requests ResourcesList
	Limits   ResourcesList
}

type ResourcesList struct {
	CPU    *inf.Dec
	Memory *inf.Dec
}

type ResourceUsage struct {
	CPU    *inf.Dec
	Memory *inf.Dec
}

type ContainerResources map[string]Resources
