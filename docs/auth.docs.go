package docs

// Auth API endpoints documentation
const authPaths = `
	"/auth/signup": {
		"post": {
			"tags": [
				"Auth"
			],
			"summary": "User Signup",
			"description": "Creates a new user account",
			"operationId": "signup",
			"parameters": [
				{
					"in": "body",
					"name": "body",
					"description": "Signup request",
					"required": true,
					"schema": {
						"$ref": "#/definitions/SignupInput"
					}
				}
			],
			"responses": {
						"201": {
							"description": "Signup accepted - verification email sent",
							"schema": {
								"type": "object",
								"properties": {
									"title": { "type": "string", "example": "Success" },
									"message": { "type": "string", "example": "Please check your email for a verification link" }
								},
								"required": ["title","message"]
							}
						},
				"400": {
					"description": "Validation error",
					"schema": {
						"$ref": "#/definitions/Error"
					}
				},
				"409": {
					"description": "Duplicate email",
					"schema": {
						"$ref": "#/definitions/Error"
					}
				},
				"500": {
					"description": "Internal server error",
					"schema": {
						"$ref": "#/definitions/Error"
					}
				}
			}
		}
	},

	"/auth/verify": {
		"get": {
			"tags": ["Auth"],
			"summary": "Activate user",
			"description": "Activates a user account using a token",
			"operationId": "activateUser",
			"parameters": [
				{
					"name": "token",
					"in": "query",
					"description": "Activation token",
					"required": true,
					"type": "string"
				}
			],
			"responses": {
				"200": {
					"description": "User activated successfully",
					"schema": {"$ref": "#/definitions/SignUpResponse"}
				},
				"400": {"description": "Validation error", "schema": {"$ref": "#/definitions/Error"}},
				"404": {"description": "User not found", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		}
	},

	"/auth/signin": {
		"post": {
			"tags": ["Auth"],
			"summary": "Authenticate user",
			"description": "Authenticates a user and returns a JWT token",
			"operationId": "signin",
			"parameters": [
				{
					"in": "body",
					"name": "body",
					"description": "Signin request",
					"required": true,
					"schema": {
						"type": "object",
						"properties": {
							"email": {"type": "string", "format": "email"},
							"password": {"type": "string", "format": "password"}
						},
						"required": ["email","password"]
					}
				}
			],
			"responses": {
				"200": {"description": "Authenticated", "schema": {"type": "object", "properties": {"token": {"type":"string"}}}},
				"400": {"description": "Validation error", "schema": {"$ref": "#/definitions/Error"}},
				"401": {"description": "Unauthorized", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		}
	},
`
