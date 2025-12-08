package sqlite

import (
	"encoding/hex"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// parseObjectID converts a string ID to MongoDB ObjectID
// This is used for compatibility during migration period
func parseObjectID(id string) (primitive.ObjectID, error) {
	// If the ID is already a valid ObjectID hex string (24 chars)
	if len(id) == 24 {
		return primitive.ObjectIDFromHex(id)
	}

	// If it's a UUID format (36 chars with dashes), convert to ObjectID
	// by taking first 24 hex characters
	cleanID := ""
	for _, c := range id {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			cleanID += string(c)
		}
	}

	// Pad or truncate to 24 chars
	if len(cleanID) >= 24 {
		cleanID = cleanID[:24]
	} else {
		for len(cleanID) < 24 {
			cleanID += "0"
		}
	}

	return primitive.ObjectIDFromHex(cleanID)
}

// objectIDToString converts MongoDB ObjectID to string
func objectIDToString(oid primitive.ObjectID) string {
	return oid.Hex()
}

// bytesToHex converts byte slice to hex string for storage
func bytesToHex(b []byte) string {
	return hex.EncodeToString(b)
}

// hexToBytes converts hex string back to bytes
func hexToBytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// generateID creates a new UUID-based ID
func generateID() string {
	return primitive.NewObjectID().Hex()
}

// NullString helper for nullable strings
type NullString struct {
	Value string
	Valid bool
}

func (ns *NullString) Scan(value interface{}) error {
	if value == nil {
		ns.Value, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	ns.Value = value.(string)
	return nil
}

// StringPtr returns a pointer to the string, or nil if empty
func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// PtrString returns the string value or empty string if nil
func PtrString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
