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

// RestaurantStatus represents the status of a restaurant
type RestaurantStatus string

const (
	RestaurantStatusActive   RestaurantStatus = "active"
	RestaurantStatusInactive RestaurantStatus = "inactive"
	RestaurantStatusBlocked  RestaurantStatus = "blocked"
	RestaurantStatusDeleted  RestaurantStatus = "deleted"
)

func (s RestaurantStatus) String() string {
	return string(s)
}

// OrderStatus represents the current state of an order
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusConfirmed  OrderStatus = "confirmed"
	OrderStatusPreparing  OrderStatus = "preparing"
	OrderStatusReady      OrderStatus = "ready"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

func (s OrderStatus) String() string {
	return string(s)
}

// PaymentStatus represents the state of a payment for an order
type PaymentStatus string

const (
	PaymentStatusPending           PaymentStatus = "pending"
	PaymentStatusProcessing        PaymentStatus = "processing"
	PaymentStatusSuccess           PaymentStatus = "success"
	PaymentStatusPaid              PaymentStatus = "paid" // Alias for success in some contexts
	PaymentStatusFailed            PaymentStatus = "failed"
	PaymentStatusCancelled          PaymentStatus = "cancelled"
	PaymentStatusRefunded          PaymentStatus = "refunded"
	PaymentStatusRefunding         PaymentStatus = "refunding"
	PaymentStatusPartiallyRefunded PaymentStatus = "partially_refunded"
)

func (s PaymentStatus) String() string {
	return string(s)
}

// PaymentProvider represents the payment provider
type PaymentProvider string

const (
	PaymentProviderMonnify    PaymentProvider = "monnify"
	PaymentProviderPaystack   PaymentProvider = "paystack"
	PaymentProviderFlutterwave PaymentProvider = "flutterwave"
)

func (s PaymentProvider) String() string {
	return string(s)
}

// OrderType represents the type of fulfillment for an order
type OrderType string


const (
	OrderTypeDelivery OrderType = "delivery"
	OrderTypePickup   OrderType = "pickup"
	OrderTypeDineIn   OrderType = "dine_in"
)

func (s OrderType) String() string {
	return string(s)
}

