package scaling

import (
	"reflect"
	"testing"

	"github.com/iljarotar/hybrid-scaler/internal/strategy"
	"gopkg.in/inf.v0"
)

func TestHorizontal(t *testing.T) {
	type args struct {
		state *strategy.State
	}
	tests := []struct {
		name    string
		args    args
		want    *strategy.ScalingDecision
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Horizontal(tt.args.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("Horizontal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Horizontal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHybridHorizontalUp(t *testing.T) {
	type args struct {
		state *strategy.State
	}
	tests := []struct {
		name    string
		args    args
		want    *strategy.ScalingDecision
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Hybrid(tt.args.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("HybridHorizontalUp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HybridHorizontalUp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHybridHorizontalDown(t *testing.T) {
	type args struct {
		state *strategy.State
	}
	tests := []struct {
		name    string
		args    args
		want    *strategy.ScalingDecision
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HybridInverse(tt.args.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("HybridHorizontalDown() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HybridHorizontalDown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_currentToTargetUtilizationRatio(t *testing.T) {
	type args struct {
		usage             *inf.Dec
		requests          *inf.Dec
		targetUtilization *inf.Dec
	}
	tests := []struct {
		name    string
		args    args
		want    *inf.Dec
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := currentToTargetUtilizationRatio(tt.args.usage, tt.args.requests, tt.args.targetUtilization)
			if (err != nil) != tt.wantErr {
				t.Errorf("currentToTargetUtilizationRatio() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("currentToTargetUtilizationRatio() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVertical(t *testing.T) {
	type args struct {
		state *strategy.State
	}
	tests := []struct {
		name    string
		args    args
		want    *strategy.ScalingDecision
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Vertical(tt.args.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("Vertical() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Vertical() = %v, want %v", got, tt.want)
			}
		})
	}
}
