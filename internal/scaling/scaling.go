package scaling

import (
	"fmt"

	"gopkg.in/inf.v0"
)

// calculates `percentage = usage / requests` and returns `ratio = percentage / targetUtilization`
func currentToTargetUtilizationRatio(usage, requests, targetUtilization *inf.Dec) (*inf.Dec, error) {
	percentage := inf.NewDec(0, 0)
	zero := inf.NewDec(0, 0)

	if requests.Cmp(zero) == 0 {
		return nil, fmt.Errorf("requests cannot be zero")
	}

	if targetUtilization.Cmp(zero) == 0 {
		return nil, fmt.Errorf("target utilization cannot be zero")
	}

	percentage.QuoRound(usage, requests, 8, inf.RoundHalfUp)
	ratio := new(inf.Dec).QuoRound(percentage, targetUtilization, 8, inf.RoundHalfUp)

	return ratio, nil
}
