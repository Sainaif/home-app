package utils

import (
	"fmt"
	"math"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DecimalFromFloat converts float to Decimal128 for MongoDB
func DecimalFromFloat(f float64) (primitive.Decimal128, error) {
	return primitive.ParseDecimal128(fmt.Sprintf("%.2f", f))
}

// DecimalToFloat converts Decimal128 back to float for math
func DecimalToFloat(d primitive.Decimal128) (float64, error) {
	s := d.String()
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
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