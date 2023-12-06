package reinforcement

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/inf.v0"
)

func Test_encodeAndDecodeQTable(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name    string
		table   qTable
		want    qTable
		wantErr bool
	}{
		{
			name: "encoding and decoding does not alter the table",
			table: qTable{
				"state1": {
					"action1": inf.NewDec(10, 0),
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
			encoded, err := encodeQTable(tt.table)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeQTable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, err := decodeToQTable(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeToQTable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, *got, cmp.Comparer(decComparer)); diff != "" {
				t.Errorf("decoded q-table not as expected %v", diff)
			}
		})
	}
}

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
