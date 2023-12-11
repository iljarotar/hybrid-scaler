package scaling

import (
	"fmt"
	"math"

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

// limits value to the range [min, max]
func limitValue(value, min, max *inf.Dec) *inf.Dec {
	if value.Cmp(min) < 0 {
		return min
	}

	if value.Cmp(max) > 0 {
		return max
	}

	return value
}

// truncates fractional digits and converts `value` to `int64`
func DecToInt64(value *inf.Dec) int64 {
	scale := value.Scale()
	factor := math.Pow10(-int(scale))
	floatValue := float64(value.UnscaledBig().Int64()) * factor

	return int64(floatValue)
}
