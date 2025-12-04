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

		"/auth/refresh": {
			"post": {
				"tags": ["Auth"],
				"summary": "Refresh access token",
				"description": "Rotates and issues a new access token using the refresh token cookie",
				"operationId": "refreshToken",
				"responses": {
					"200": {"description": "New access token", "schema": {"type": "object", "properties": {"access_token": {"type": "string"}}}},
					"401": {"description": "Unauthorized or invalid refresh token", "schema": {"$ref": "#/definitions/Error"}},
					"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
				}
			}
		},

		"/auth/resend": {
			"post": {
				"tags": ["Auth"],
				"summary": "Resend verification email",
				"description": "Resends account verification email if a pending verification exists",
				"operationId": "resendVerification",
				"parameters": [{"in": "body", "name": "body", "required": true, "schema": {"type": "object", "properties": {"email": {"type": "string", "format": "email"}}, "required": ["email"]}}],
				"responses": {
					"200": {"description": "Verification email resent", "schema": {"type": "object", "properties": {"message": {"type": "string"}}}},
					"400": {"description": "Validation error", "schema": {"$ref": "#/definitions/Error"}},
					"404": {"description": "Verification token not found", "schema": {"$ref": "#/definitions/Error"}},
					"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
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
				"200": {"description": "Authenticated", "schema": {"$ref": "#/definitions/SigninResponse"}},
				"400": {"description": "Validation error", "schema": {"$ref": "#/definitions/Error"}},
				"401": {"description": "Unauthorized", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		}
	},
`
