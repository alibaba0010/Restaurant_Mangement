package types

// UserRole represents the different roles in the system
type UserRole string

const (
	// RoleUser is a regular user with basic permissions
	RoleUser UserRole = "user"

	// RoleAdmin is an administrator with full permissions
	RoleAdmin UserRole = "admin"

	// RoleManagement is a management role with elevated permissions
	RoleManagement UserRole = "management"
)

// String returns the string representation of the role
func (r UserRole) String() string {
	return string(r)
}

// IsValid checks if the role is a valid role
func (r UserRole) IsValid() bool {
	switch r {
	case RoleUser, RoleAdmin, RoleManagement:
		return true
	default:
		return false
	}
}

func (r UserRole) HasPermission(requiredRole UserRole) bool {
	switch r {
	case RoleAdmin:
		return true // Admin can access everything
	case RoleManagement:
		// Management can access Management and User endpoints
		return requiredRole == RoleManagement || requiredRole == RoleUser
	case RoleUser:
		// User can only access User endpoints
		return requiredRole == RoleUser
	default:
		return false
	}
}

// AllRoles returns all valid roles
func AllRoles() []UserRole {
	return []UserRole{RoleUser, RoleManagement, RoleAdmin}
}

// AllRolesAsStrings returns all valid roles as strings
func AllRolesAsStrings() []string {
	return []string{
		string(RoleUser),
		string(RoleManagement),
		string(RoleAdmin),
	}
}

// ToUserRole converts a string to UserRole
func ToUserRole(s string) (UserRole, bool) {
	role := UserRole(s)
	return role, role.IsValid()
}
