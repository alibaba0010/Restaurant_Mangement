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
		},
		"MenuResponse": {
			"type": "object",
			"properties": {
				"id": { "type": "string", "format": "uuid" },
				"name": { "type": "string" },
				"description": { "type": "string" },
				"price": { "type": "number", "format": "float" },
				"image_urls": { "type": "array", "items": { "type": "string" } },
				"video_url": { "type": "string" },
				"restaurant_id": { "type": "string", "format": "uuid" },
				"is_available": { "type": "boolean" },
				"prep_time_minutes": { "type": "integer" },
				"calories": { "type": "integer" },
				"created_at": { "type": "string", "format": "date-time" },
				"updated_at": { "type": "string", "format": "date-time" }
			}
		},
		"CreateMenuInput": {
			"type": "object",
			"properties": {
				"name": { "type": "string", "minLength": 2 },
				"description": { "type": "string" },
				"price": { "type": "number" },
				"image_urls": { "type": "array", "items": { "type": "string" } },
				"video_url": { "type": "string" },
				"restaurant_id": { "type": "string", "format": "uuid" },
				"is_available": { "type": "boolean" },
				"prep_time_minutes": { "type": "integer" },
				"calories": { "type": "integer" }
			},
			"required": ["name", "price", "restaurant_id"]
		},
		"UpdateMenuInput": {
			"type": "object",
			"properties": {
				"name": { "type": "string" },
				"description": { "type": "string" },
				"price": { "type": "number" },
				"image_urls": { "type": "array", "items": { "type": "string" } },
				"video_url": { "type": "string" },
				"is_available": { "type": "boolean" },
				"prep_time_minutes": { "type": "integer" },
				"calories": { "type": "integer" }
			}
		},
		"MenusListResponse": {
			"type": "object",
			"properties": {
				"title": { "type": "string" },
				"data": { "type": "array", "items": { "$ref": "#/definitions/MenuResponse" } },
				"meta": { "$ref": "#/definitions/PaginationMeta" }
			}
		},
		"CreateOrderItemInput": {
			"type": "object",
			"properties": {
				"menu_id": { "type": "string", "format": "uuid" },
				"quantity": { "type": "integer", "minimum": 1 }
			},
			"required": ["menu_id", "quantity"]
		},
		"CreateOrderInput": {
			"type": "object",
			"properties": {
				"restaurant_id": { "type": "string", "format": "uuid" },
				"delivery_address": { "type": "string" },
				"items": { "type": "array", "items": { "$ref": "#/definitions/CreateOrderItemInput" } }
			},
			"required": ["restaurant_id", "delivery_address", "items"]
		},
		"OrderItemResponse": {
			"type": "object",
			"properties": {
				"id": { "type": "string", "format": "uuid" },
				"menu_id": { "type": "string", "format": "uuid" },
				"name": { "type": "string" },
				"quantity": { "type": "integer" },
				"price": { "type": "number" }
			}
		},
		"OrderResponse": {
			"type": "object",
			"properties": {
				"id": { "type": "string", "format": "uuid" },
				"user_id": { "type": "string", "format": "uuid" },
				"restaurant_id": { "type": "string", "format": "uuid" },
				"total_amount": { "type": "number" },
				"status": { "type": "string" },
				"delivery_address": { "type": "string" },
				"created_at": { "type": "string", "format": "date-time" },
				"updated_at": { "type": "string", "format": "date-time" },
				"items": { "type": "array", "items": { "$ref": "#/definitions/OrderItemResponse" } }
			}
		},
		"UpdateOrderStatusInput": {
			"type": "object",
			"properties": {
				"status": { "type": "string", "enum": ["pending", "processing", "completed", "cancelled"] }
			},
			"required": ["status"]
		},
		"UserOrdersResponse": {
			"type": "object",
			"properties": {
				"orders": { "type": "array", "items": { "$ref": "#/definitions/OrderResponse" } },
				"next_cursor": { "type": "string" },
				"has_more": { "type": "boolean" }
			}
		},
		"InitiatePaymentRequest": {
			"type": "object",
			"properties": {
				"order_id": { "type": "string", "format": "uuid" },
				"provider": { "type": "string", "enum": ["monnify", "paystack", "flutterwave"] },
				"callback_url": { "type": "string", "format": "url" }
			},
			"required": ["order_id", "provider", "callback_url"]
		},
		"InitiatePaymentResponse": {
			"type": "object",
			"properties": {
				"payment_id": { "type": "string", "format": "uuid" },
				"authorization_url": { "type": "string" },
				"access_code": { "type": "string" },
				"reference": { "type": "string" }
			}
		},
		"PaymentResponse": {
			"type": "object",
			"properties": {
				"id": { "type": "string", "format": "uuid" },
				"order_id": { "type": "string", "format": "uuid" },
				"amount": { "type": "number" },
				"currency": { "type": "string" },
				"provider": { "type": "string" },
				"status": { "type": "string" },
				"reference": { "type": "string" },
				"created_at": { "type": "string", "format": "date-time" }
			}
		},
		"CursorMeta": {
			"type": "object",
			"properties": {
				"next_cursor": { "type": "string" },
				"has_more": { "type": "boolean" },
				"total": { "type": "integer" }
			}
		},
		"CreateCategoryInput": {
			"type": "object",
			"properties": {
				"restaurant_id": { "type": "string", "format": "uuid" },
				"name": { "type": "string", "minLength": 2, "maxLength": 100 },
				"description": { "type": "string", "maxLength": 500 },
				"sort_order": { "type": "integer", "minimum": 0 }
			},
			"required": ["restaurant_id", "name"]
		},
		"UpdateCategoryInput": {
			"type": "object",
			"properties": {
				"name": { "type": "string", "minLength": 2, "maxLength": 100 },
				"description": { "type": "string", "maxLength": 500 },
				"sort_order": { "type": "integer", "minimum": 0 }
			}
		},
		"CategoryResponse": {
			"type": "object",
			"properties": {
				"id": { "type": "string", "format": "uuid" },
				"restaurant_id": { "type": "string", "format": "uuid" },
				"name": { "type": "string" },
				"description": { "type": "string" },
				"sort_order": { "type": "integer" },
				"created_at": { "type": "string", "format": "date-time" },
				"updated_at": { "type": "string", "format": "date-time" },
				"menus": { "type": "array", "items": { "$ref": "#/definitions/MenuResponse" } }
			}
		},
		"CategoryListResponse": {
			"type": "object",
			"properties": {
				"data": { "type": "array", "items": { "$ref": "#/definitions/CategoryResponse" } },
				"meta": { "$ref": "#/definitions/CursorMeta" }
			}
		}
	}

`
