package docs

// API definitions/schemas documentation
const definitions = `
	"definitions": {
		"Error": {
			"type": "object",
			"properties": {
				"title": {
					"type": "string"
				},
				"message": {
					"type": "string"
				}
			},
			"required": [
				"title",
				"message"
			]
		},
		"User": {
			"type": "object",
			"properties": {
				"id": {
					"type": "string",
					"format": "uuid"
				},
				"name": {
					"type": "string"
				},
				"email": {
					"type": "string",
					"format": "email"
				},
				"address": {
					"type": "string"
				},
				"role": {
					"type": "string",
					"example": "user"
				},
				"created_at": {
					"type": "string",
					"format": "date-time"
				},
				"updated_at": {
					"type": "string",
					"format": "date-time"
				}
			},
			"required": [
				"id",
				"name",
				"email"
			]
		},
		"SignupInput": {
			"type": "object",
			"properties": {
				"name": {
					"type": "string",
					"minLength": 3,
					"maxLength": 50
				},
				"email": {
					"type": "string",
					"format": "email"
				},
				"password": {
					"type": "string",
					"format": "password",
					"minLength": 6
				},
				"confirmPassword": {
					"type": "string",
					"format": "password",
					"minLength": 6
				}
			},
			"required": [
				"name",
				"email",
				"password",
				"confirmPassword"
			]
		},
		"SignUpResponse": {
			"type": "object",
			"properties": {
				"title": { "type": "string" },
				"data": {
					"type": "object",
					"properties": {
						"id": { "type": "string", "format": "uuid" },
						"name": { "type": "string" },
						"email": { "type": "string", "format": "email" },
						"role": { "type": "string" },
						"access_token": { "type": "string" },
						"refresh_token": { "type": "string" }
					},
					"required": ["id","name","email","role","access_token"]
				}
			},
			"required": ["title","data"]
		},
		"UserInput": {
			"type": "object",
			"properties": {
				"name": {
					"type": "string",
					"minLength": 2,
					"maxLength": 50
				},
				"email": {
					"type": "string",
					"format": "email"
				},
				"password": {
					"type": "string",
					"format": "password",
					"minLength": 8
				}
			},
			"required": [
				"name",
				"email",
				"password"
			]
		},
		"Restaurant": {
			"type": "object",
			"properties": {
				"id": {
					"type": "string",
					"format": "uuid"
				},
				"name": {
					"type": "string"
				},
				"description": {
					"type": "string"
				},
				"address": {
					"type": "string"
				},
				"cuisine_type": {
					"type": "string"
				},
				"rating": {
					"type": "number",
					"format": "float",
					"minimum": 0,
					"maximum": 5
				},
				"created_at": {
					"type": "string",
					"format": "date-time"
				},
				"updated_at": {
					"type": "string",
					"format": "date-time"
				}
			},
			"required": [
				"id",
				"name",
				"address"
			]
		},
		"RestaurantInput": {
			"type": "object",
			"properties": {
				"name": {
					"type": "string",
					"minLength": 2,
					"maxLength": 100
				},
				"description": {
					"type": "string",
					"maxLength": 500
				},
				"address": {
					"type": "string",
					"minLength": 5,
					"maxLength": 200
				},
				"cuisine_type": {
					"type": "string"
				}
			},
			"required": [
				"name",
				"address"
			]
		}
	}
`
