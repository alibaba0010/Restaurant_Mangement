package types

// UserStatus represents the different statuses a user can have
type UserStatus string

const (
	// StatusActive represents an active user
	StatusActive UserStatus = "active"

	// StatusInactive represents an inactive user
	StatusInactive UserStatus = "inactive"

	// StatusPending represents a user whose account is pending activation/verification
	StatusPending UserStatus = "pending"

	// StatusSuspended represents a user whose account has been suspended
	StatusSuspended UserStatus = "suspended"

	// StatusDeleted represents a user whose account has been marked as deleted
	StatusDeleted UserStatus = "deleted"
)

// String returns the string representation of the status
func (s UserStatus) String() string {
	return string(s)
}

// IsValid checks if the status is a valid status
func (s UserStatus) IsValid() bool {
	switch s {
	case StatusActive, StatusInactive, StatusPending, StatusSuspended, StatusDeleted:
		return true
	default:
		return false
	}
}

// AllStatuses returns all valid statuses
func AllStatuses() []UserStatus {
	return []UserStatus{
		StatusActive,
		StatusInactive,
		StatusPending,
		StatusSuspended,
		StatusDeleted,
	}
}

// AllStatusesAsStrings returns all valid statuses as strings
func AllStatusesAsStrings() []string {
	return []string{
		string(StatusActive),
		string(StatusInactive),
		string(StatusPending),
		string(StatusSuspended),
		string(StatusDeleted),
	}
}

// ToUserStatus converts a string to UserStatus
func ToUserStatus(s string) (UserStatus, bool) {
	status := UserStatus(s)
	return status, status.IsValid()
}
