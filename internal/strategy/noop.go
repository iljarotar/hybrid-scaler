package strategy

type NoOp struct{}

func (n *NoOp) MakeDecision(state *State, learningState []byte) (*ScalingDecision, []byte, error) {
	return &ScalingDecision{
		Replicas:           state.Replicas,
		ContainerResources: state.ContainerResources,
	}, learningState, nil
}
