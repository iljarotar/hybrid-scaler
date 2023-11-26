package reinforcement

import "fmt"

type QLearning struct {
	qTable
}

type qTable map[stateName]qTableRow

type qTableRow map[actionName]float64

func (l *QLearning) Update(s state, a action, alpha, gamma float64) error {
	// TODO: implement q-table update (see RLbook p.131)

	_, ok := l.qTable[s.Name][a.Name]
	if !ok {
		return fmt.Errorf("q-table entry for state-action pair not found, state: %s, action: %s", s.Name, a.Name)
	}

	return nil
}

func (l *QLearning) GetGreedyActionsAmong(actions) actions {
	actions := make(actions, 0)

	// TODO: implement

	return actions
}
