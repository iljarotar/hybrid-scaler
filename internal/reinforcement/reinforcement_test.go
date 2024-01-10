package reinforcement

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
)

func decComparer(a, b *inf.Dec) bool {
	if a == nil && b != nil {
		return false
	}

	if b == nil && a != nil {
		return false
	}

	if a == nil && b == nil {
		return true
	}

	return a.Cmp(b) == 0
}

func Test_encodeAndDecodeQTable(t *testing.T) {
	tests := []struct {
		name         string
		learninState *learningState
		want         qTable
		wantErr      bool
	}{
		{
			name: "encoding and decoding does not alter the table",
			learninState: &learningState{
				Table: qTable{
					"state1": {
						"action1": inf.NewDec(10, 0),
					},
				},
			},
			want: qTable{
				"state1": {
					"action1": inf.NewDec(10, 0),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := encodeLearningState(tt.learninState)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeQTable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, err := decodeToLearningState(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeToQTable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got.Table, cmp.Comparer(decComparer)); diff != "" {
				t.Errorf("decoded q-table not as expected %v", diff)
			}
		})
	}
}

func TestQLearning_initializeRow(t *testing.T) {
	l := &QLearning{
		allActions: allActions,
	}
	tests := []struct {
		name  string
		state stateName
		table qTable
		want  qTable
	}{
		{
			name:  "all actions initialized with initial value",
			state: "state2",
			table: qTable{
				"state1": {},
			},
			want: qTable{
				"state1": {},
				"state2": {
					actionNone:       initialValue,
					actionHorizontal: initialValue,
					actionVertical:   initialValue,
					actionHybrid:     initialValue,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l.initializeRow(tt.state, tt.table)

			if diff := cmp.Diff(tt.want, tt.table, cmp.Comparer(decComparer)); diff != "" {
				t.Errorf("initializeRow() %v", diff)
			}
		})
	}
}

func TestQLearning_bestActionValueInState(t *testing.T) {
	tests := []struct {
		name  string
		state stateName
		table qTable
		want  *inf.Dec
	}{
		{
			name:  "no entry for this state yet",
			state: "state2",
			table: qTable{
				"state1": {
					actionNone: inf.NewDec(-1, 0),
				},
			},
			want: initialValue,
		},
		{
			name:  "no entries for any actions in this state yet",
			state: "state1",
			table: qTable{
				"state1": {},
			},
			want: initialValue,
		},
		{
			name:  "choose cheapest action",
			state: "state1",
			table: qTable{
				"state1": {
					actionNone:       inf.NewDec(1, 0),
					actionHorizontal: inf.NewDec(2, 0),
					actionVertical:   inf.NewDec(3, 0),
					actionHybrid:     inf.NewDec(4, 0),
				},
			},
			want: inf.NewDec(1, 0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bestActionValueInState(tt.state, tt.table)

			if diff := cmp.Diff(tt.want, got, cmp.Comparer(decComparer)); diff != "" {
				t.Errorf("bestActionValueInState() %v", diff)
			}
		})
	}
}

func TestQLearning_GetGreedyActions(t *testing.T) {
	l := &QLearning{
		allActions: allActions,
	}
	tests := []struct {
		name          string
		state         stateName
		learningState *learningState
		want          actions
		wantErr       bool
	}{
		{
			name:  "no entry for this state yet",
			state: "state1",
			learningState: &learningState{
				Table: qTable{},
			},
			want:    allActions,
			wantErr: false,
		},
		{
			name:  "choose cheapest actions",
			state: "state1",
			learningState: &learningState{
				Table: qTable{
					"state1": {
						actionNone:       inf.NewDec(1, 0),
						actionHorizontal: inf.NewDec(1, 0),
						actionVertical:   inf.NewDec(2, 0),
						actionHybrid:     inf.NewDec(1, 0),
					},
				},
			},
			want: actions{
				actionNone,
				actionHorizontal,
				actionHybrid,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			learningState, err := encodeLearningState(tt.learningState)
			if err != nil {
				t.Errorf("QLearning.GetGreedyActions() q-table encoding error = %v", err)
			}

			got, err := l.GetGreedyActions(tt.state, learningState)
			if (err != nil) != tt.wantErr {
				t.Errorf("QLearning.GetGreedyActions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("QLearning.GetGreedyActions() %v", diff)
			}
		})
	}
}

func TestQLearning_evaluateCost(t *testing.T) {
	l := &QLearning{
		cpuCost:                  inf.NewDec(1, 0),
		memoryCost:               inf.NewDec(1, 9),
		underprovisioningPenalty: inf.NewDec(2, 0),
	}
	tests := []struct {
		name    string
		state   *state
		want    *inf.Dec
		wantErr bool
	}{
		{
			name: "cpu threshold exceeded",
			state: &state{
				Replicas:                3,
				CpuRequests:             inf.NewDec(100, 3),
				MemoryRequests:          inf.NewDec(1, -6),
				CpuUtilization:          inf.NewDec(60, 2),
				MemoryUtilization:       inf.NewDec(70, 2),
				CpuTargetUtilization:    inf.NewDec(50, 2),
				MemoryTargetUtilization: inf.NewDec(80, 2),
			},
			want:    inf.NewDec(423, 3),
			wantErr: false,
		},
		{
			name: "thresholds not exceeded",
			state: &state{
				Replicas:                3,
				CpuRequests:             inf.NewDec(100, 3),
				MemoryRequests:          inf.NewDec(1, -6),
				CpuUtilization:          inf.NewDec(60, 2),
				MemoryUtilization:       inf.NewDec(80, 2),
				CpuTargetUtilization:    inf.NewDec(60, 2),
				MemoryTargetUtilization: inf.NewDec(80, 2),
			},
			want:    inf.NewDec(303, 3),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := l.evaluateCost(tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("QLearning.evaluateCost() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.want, got, cmp.Comparer(decComparer)); diff != "" {
				t.Errorf("QLearning.evaluateCost() %v", diff)
			}
		})
	}
}

func Test_quantizePercentage(t *testing.T) {
	tests := []struct {
		name    string
		value   *inf.Dec
		quantum *inf.Dec
		want    int64
	}{
		{
			name:    "quantize to 0",
			value:   inf.NewDec(14, 0),
			quantum: inf.NewDec(20, 0),
			want:    0,
		},
		{
			name:    "quantize to 20%",
			value:   inf.NewDec(29, 0),
			quantum: inf.NewDec(20, 0),
			want:    20,
		},
		{
			name:    "quantize to 100%",
			value:   inf.NewDec(101, 0),
			quantum: inf.NewDec(20, 0),
			want:    100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quantizePercentage(tt.value, tt.quantum); got != tt.want {
				t.Errorf("quantizePercentage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertState(t *testing.T) {
	tests := []struct {
		name    string
		state   *strategy.State
		want    *state
		wantErr bool
	}{
		{
			name: "correctly convert strategy state to q-learning state",
			state: &strategy.State{
				Replicas: 4,
				Constraints: strategy.Constraints{
					MaxResources: strategy.ResourcesList{
						CPU:    inf.NewDec(500, 0),
						Memory: inf.NewDec(800, 0),
					},
				},
				PodMetrics: strategy.PodMetrics{
					ResourceUsage: strategy.ResourcesList{
						CPU:    inf.NewDec(600, 0),
						Memory: inf.NewDec(180, 0),
					},
					Resources: strategy.Resources{
						Requests: strategy.ResourcesList{
							CPU:    inf.NewDec(100, 0),
							Memory: inf.NewDec(250, 0),
						},
						Limits: strategy.ResourcesList{
							CPU:    inf.NewDec(500, 0),
							Memory: inf.NewDec(700, 0),
						},
					},
				},
				TargetUtilization: strategy.ResourcesList{
					CPU:    inf.NewDec(50, 2),
					Memory: inf.NewDec(80, 2),
				},
			},
			want: &state{
				Name:                    "4_100_75_100_75",
				Replicas:                4,
				CpuRequests:             inf.NewDec(100, 0),
				MemoryRequests:          inf.NewDec(250, 0),
				CpuUtilization:          inf.NewDec(6, 0),
				MemoryUtilization:       inf.NewDec(72, 2),
				CpuTargetUtilization:    inf.NewDec(50, 2),
				MemoryTargetUtilization: inf.NewDec(80, 2),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertState(tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmp.Comparer(decComparer)); diff != "" {
				t.Errorf("convertState() %v", diff)
			}
		})
	}
}
