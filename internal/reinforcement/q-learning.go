package reinforcement

type QLearning struct {
	qTable
}

type qTable map[stateName]qTableRow

type qTableRow map[action]float64

func (l *QLearning) Update(s *state, a action, alpha, gamma float64) error {
	// TODO: implement q-table update (see RLbook p.131)

	return nil
}

func (l *QLearning) GetGreedyActionsAmong(as actions) actions {
	actions := make(actions, 0)

	// TODO: implement

	return actions
}
