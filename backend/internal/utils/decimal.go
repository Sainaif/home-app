package utils

import (
	"fmt"
	"math"
	"strconv"
)

// FloatToDecimalString converts float to decimal string for SQLite storage
func FloatToDecimalString(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

// DecimalStringToFloat converts decimal string back to float for math
func DecimalStringToFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// RoundBankers does banker's rounding (round half to even)
func RoundBankers(value float64, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	rounded := value * multiplier
	integer := math.Floor(rounded)
	fraction := rounded - integer

	if fraction > 0.5 {
		return math.Ceil(rounded) / multiplier
	} else if fraction < 0.5 {
		return math.Floor(rounded) / multiplier
	}

	// fraction == 0.5: round to nearest even
	if int(integer)%2 == 0 {
		return integer / multiplier
	}
	return math.Ceil(rounded) / multiplier
}

// RoundPLN rounds money to 2 decimals
func RoundPLN(value float64) float64 {
	return RoundBankers(value, 2)
}

// RoundUnits rounds kWh/m³ to 3 decimals
func RoundUnits(value float64) float64 {
	return RoundBankers(value, 3)
}

// RoundToTwoDecimals rounds to 2 decimal places
func RoundToTwoDecimals(value float64) float64 {
	return RoundBankers(value, 2)
}

// RoundToThreeDecimals rounds to 3 decimal places
func RoundToThreeDecimals(value float64) float64 {
	return RoundBankers(value, 3)
}
