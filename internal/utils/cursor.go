package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// CursorValue is a type alias for any to be explicit about potentially mixed types
type CursorValue any

// Cursor represents the decoded cursor data
type Cursor struct {
	LastValue CursorValue `json:"v"`  // Short key to save space
	LastID    string      `json:"id"` // Unique tie-breaker
}

// EncodeCursor generates a base64 encoded cursor string
func EncodeCursor(lastValue CursorValue, lastID string) string {
	c := Cursor{LastValue: lastValue, LastID: lastID}
	b, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

// DecodeCursor parses a base64 encoded cursor string
func DecodeCursor(cursorStr string) (*Cursor, error) {
	if cursorStr == "" {
		return nil, nil
	}
	b, err := base64.StdEncoding.DecodeString(cursorStr)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor format")
	}
	var c Cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("invalid cursor data")
	}
	return &c, nil
}

// GetCursorValueAsTime is a helper to safely cast the generic interface to time.Time
// JSON unmarshalling often results in string for time, so we need to parse it.
func GetCursorValueAsTime(val CursorValue) time.Time {
	if val == nil {
		return time.Time{}
	}
	// If it was unmarshalled from JSON, it's likely a string
	if s, ok := val.(string); ok {
		t, _ := time.Parse(time.RFC3339Nano, s) // standard json marshaling format
		return t
	}
	// Assuming it's already a time object (not coming from json)
	if t, ok := val.(time.Time); ok {
		return t
	}
	return time.Time{}
}

// GetCursorValueAsString helper
func GetCursorValueAsString(val CursorValue) string {
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}

// GetCursorValueAsFloat helper
func GetCursorValueAsFloat(val CursorValue) float64 {
	if f, ok := val.(float64); ok {
		return f
	}
	// int to float
	if i, ok := val.(int); ok {
		return float64(i)
	}
	return 0
}
