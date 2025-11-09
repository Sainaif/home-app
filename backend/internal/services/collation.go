package services

import "go.mongodb.org/mongo-driver/mongo/options"

// caseInsensitiveEmailCollation ensures MongoDB queries treat email comparisons
// as case insensitive so users aren't locked out due to casing differences.
var caseInsensitiveEmailCollation = &options.Collation{
	Locale:   "en",
	Strength: 2, // Primary level comparison (case-insensitive)
}
