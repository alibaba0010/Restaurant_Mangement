package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"go.uber.org/zap"
)

// CursorValue is a type alias for any to be explicit about potentially mixed types
type CursorValue any

// Cursor represents the decoded cursor data
type Cursor struct {
	Sort      string      `json:"s"`  // Sort column name, for validation
	LastValue CursorValue `json:"v"`  // last value for pagination
	LastID    string      `json:"id"` // unique identifier (tie-breaker)
}

// EncodeCursor generates a base64 encoded cursor string.
// Returns an empty string and logs an error if marshaling fails.
func EncodeCursor(lastValue CursorValue, lastID string, sort string) string {
	if lastID == "" {
		logger.Log.Warn("EncodeCursor: lastID is empty")
	}
	if sort == "" {
		logger.Log.Warn("EncodeCursor: sort field is empty")
	}

	c := Cursor{LastValue: lastValue, LastID: lastID, Sort: sort}
	b, err := json.Marshal(c)
	if err != nil {
		logger.Log.Error("failed to encode cursor", zap.Error(err), zap.String("lastID", lastID), zap.String("sort", sort))
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

// DecodeCursor parses a base64 URLEncoded cursor string
func DecodeCursor(cursorStr string) (*Cursor, error) {
	if cursorStr == "" {
		return nil, nil
	}

	// Try URLEncoding first (recommended), fallback to StdEncoding for backward compatibility
	b, err := base64.URLEncoding.DecodeString(cursorStr)
	if err != nil {
		b, err = base64.StdEncoding.DecodeString(cursorStr)
		if err != nil {
			return nil, errors.BadRequestError("invalid format")
		}
	}

	var c Cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, errors.BadRequestError("invalid data")
	}

	if err := c.Validate(); err != nil {
		return nil, errors.BadRequestError(err.Error())
	}

	return &c, nil
}

// Validate ensures the cursor contains necessary fields
func (c *Cursor) Validate() error {
	if c == nil {
		return fmt.Errorf("cursor is nil")
	}
	if c.LastID == "" {
		return fmt.Errorf("cursor lastID cannot be empty")
	}
	if c.Sort == "" {
		return fmt.Errorf("cursor sort field cannot be empty")
	}
	if c.LastValue == nil {
		return fmt.Errorf("cursor lastValue cannot be nil")
	}
	return nil
}

// GetCursorValueAsTime safely casts the generic interface to time.Time.
// Returns an error if conversion or parsing fails.
func GetCursorValueAsTime(val CursorValue) (time.Time, error) {
	if val == nil {
		return time.Time{}, fmt.Errorf("cursor value is nil")
	}

	// If it was unmarshalled from JSON, it's likely a string
	if s, ok := val.(string); ok {
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse time cursor: %w", err)
		}
		return t, nil
	}

	// If it's already a time.Time
	if t, ok := val.(time.Time); ok {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("cursor value is not a time type: %T", val)
}

// GetCursorValueAsString returns the value as string or error if nil
func GetCursorValueAsString(val CursorValue) (string, error) {
	if val == nil {
		return "", fmt.Errorf("cursor value is nil")
	}
	if s, ok := val.(string); ok {
		return s, nil
	}
	return fmt.Sprintf("%v", val), nil
}

// GetCursorValueAsFloat safely converts numeric types to float64.
func GetCursorValueAsFloat(val CursorValue) (float64, error) {
	if val == nil {
		return 0, fmt.Errorf("cursor value is nil")
	}

	if f, ok := val.(float64); ok {
		return f, nil
	}

	// Check for potential precision loss if needed, but for now support common types
	if i, ok := val.(int); ok {
		return float64(i), nil
	}
	if i64, ok := val.(int64); ok {
		return float64(i64), nil
	}

	return 0, fmt.Errorf("cursor value is not a numeric type: %T", val)
}

// GetCursorValueAsInt safely converts numeric types to int.
func GetCursorValueAsInt(val CursorValue) (int, error) {
	if val == nil {
		return 0, fmt.Errorf("cursor value is nil")
	}

	if f, ok := val.(float64); ok {
		return int(f), nil
	}
	if i, ok := val.(int); ok {
		return i, nil
	}
	if i64, ok := val.(int64); ok {
		return int(i64), nil
	}

	return 0, fmt.Errorf("cursor value is not a numeric type: %T", val)
}
