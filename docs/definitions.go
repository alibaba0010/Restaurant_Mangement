package docs

// API definitions/schemas documentation
const definitions = `
	"definitions": {
		"Error": {
			"type": "object",
			"properties": {
				"title": { "type": "string" },
				"message": { "type": "string" }
			},
			"required": ["title", "message"]
		},
		"MessageResponse": {
			"type": "object",
			"properties": {
				"title": { "type": "string" },
				"message": { "type": "string" }
			},
			"required": ["title", "message"]
		},
		"User": {
			"type": "object",
			"properties": {
				"id": { "type": "string", "format": "uuid" },
				"name": { "type": "string" },
				"email": { "type": "string", "format": "email" },
				"address": { "type": "string" },
				"phone_number": { "type": "string" },
				"avatar_url": { "type": "string" },
				"role": { "type": "string", "example": "user" },
				"status": { "type": "string", "example": "active" },
				"created_at": { "type": "string", "format": "date-time" },
				"updated_at": { "type": "string", "format": "date-time" }
			},
			"required": ["id", "name", "email", "role", "status"]
		},
		"SignupInput": {
			"type": "object",
			"properties": {
				"name": { "type": "string", "minLength": 3, "maxLength": 50 },
				"email": { "type": "string", "format": "email" },
				"password": { "type": "string", "format": "password", "minLength": 6 },
				"confirmPassword": { "type": "string", "format": "password", "minLength": 6 },
				"role": { "type": "string", "enum": ["user", "admin", "management"] }
			},
			"required": ["name", "email", "password", "confirmPassword"]
		},
		"SigninInput": {
			"type": "object",
			"properties": {
				"email": { "type": "string", "format": "email" },
				"password": { "type": "string", "format": "password" }
			},
			"required": ["email", "password"]
		},
		"SigninResponse": {
			"type": "object",
			"properties": {
				"title": { "type": "string" },
				"data": { "$ref": "#/definitions/User" }
			},
			"required": ["title", "data"]
		},
		"SignUpResponse": {
			"type": "object",
			"properties": {
				"title": { "type": "string" },
				"data": { "$ref": "#/definitions/User" }
			},
			"required": ["title", "data"]
		},
		"UpdateUserInput": {
			"type": "object",
			"properties": {
				"address": { "type": "string", "minLength": 5, "maxLength": 255 },
				"phone_number": { "type": "string", "minLength": 10, "maxLength": 15 }
			}
		},
		"UpdateUserRoleInput": {
			"type": "object",
			"properties": {
				"role": { "type": "string", "enum": ["user", "admin", "management"] }
			},
			"required": ["role"]
		},
		"ForgotPasswordInput": {
			"type": "object",
			"properties": {
				"email": { "type": "string", "format": "email" }
			},
			"required": ["email"]
		},
		"ResetPasswordInput": {
			"type": "object",
			"properties": {
				"token": { "type": "string" },
				"password": { "type": "string", "format": "password" },
				"confirmPassword": { "type": "string", "format": "password" }
			},
			"required": ["token", "password", "confirmPassword"]
		},
		"VerifyOAuthInput": {
			"type": "object",
			"properties": {
				"code": { "type": "string" },
				"state": { "type": "string" }
			},
			"required": ["code", "state"]
		},
		"OAuthLoginResponse": {
			"type": "object",
			"properties": {
				"url": { "type": "string" }
			},
			"required": ["url"]
		},
		"PaginationMeta": {
			"type": "object",
			"properties": {
				"page": { "type": "integer" },
				"page_size": { "type": "integer" },
				"total": { "type": "integer" },
				"total_pages": { "type": "integer" }
			}
		},
		"UsersListResponse": {
			"type": "object",
			"properties": {
				"title": { "type": "string" },
				"data": { "type": "array", "items": { "$ref": "#/definitions/User" } },
				"meta": { "$ref": "#/definitions/PaginationMeta" }
			}
		},
		"Restaurant": {
			"type": "object",
			"properties": {
				"id": { "type": "string", "format": "uuid" },
				"name": { "type": "string" },
				"description": { "type": "string" },
				"address": { "type": "string" },
				"cuisine_type": { "type": "string" },
				"rating": { "type": "number", "format": "float" },
				"created_at": { "type": "string", "format": "date-time" },
				"updated_at": { "type": "string", "format": "date-time" }
			},
			"required": ["id", "name", "address"]
		},
		"RestaurantInput": {
			"type": "object",
			"properties": {
				"name": { "type": "string", "minLength": 2, "maxLength": 100 },
				"description": { "type": "string", "maxLength": 500 },
				"address": { "type": "string", "minLength": 5, "maxLength": 200 },
				"cuisine_type": { "type": "string" }
			},
			"required": ["name", "address"]
		},
		"RestaurantsListResponse": {
			"type": "object",
			"properties": {
				"title": { "type": "string" },
				"data": { "type": "array", "items": { "$ref": "#/definitions/Restaurant" } },
				"meta": { "$ref": "#/definitions/PaginationMeta" }
			}
		}
	}

`
