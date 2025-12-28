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
						"$ref": "#/definitions/SigninInput"
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

	"/auth/forgot-password": {
		"post": {
			"tags": ["Auth"],
			"summary": "Request password reset",
			"description": "Sends a password reset email to the user",
			"operationId": "forgotPassword",
			"parameters": [
				{
					"in": "body",
					"name": "body",
					"required": true,
					"schema": { "$ref": "#/definitions/ForgotPasswordInput" }
				}
			],
			"responses": {
				"200": {"description": "Reset email sent", "schema": {"$ref": "#/definitions/MessageResponse"}},
				"400": {"description": "Validation error", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		}
	},

	"/auth/reset-password": {
		"post": {
			"tags": ["Auth"],
			"summary": "Reset password",
			"description": "Resets the user password using a token",
			"operationId": "resetPassword",
			"parameters": [
				{
					"in": "body",
					"name": "body",
					"required": true,
					"schema": { "$ref": "#/definitions/ResetPasswordInput" }
				}
			],
			"responses": {
				"200": {"description": "Password reset successful", "schema": {"$ref": "#/definitions/MessageResponse"}},
				"400": {"description": "Validation error", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		}
	},

	"/auth/{provider}/login": {
		"get": {
			"tags": ["Auth"],
			"summary": "Initiate OAuth login",
			"description": "Returns the OAuth provider's login URL",
			"operationId": "initiateOAuth",
			"parameters": [
				{
					"name": "provider",
					"in": "path",
					"required": true,
					"type": "string",
					"enum": ["google", "facebook"]
				}
			],
			"responses": {
				"200": {"description": "OAuth URL", "schema": {"$ref": "#/definitions/OAuthLoginResponse"}},
				"400": {"description": "Unsupported provider", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		}
	},

	"/auth/{provider}/verify": {
		"post": {
			"tags": ["Auth"],
			"summary": "Verify OAuth callback",
			"description": "Exchanges code for user info and returns a login response",
			"operationId": "verifyOAuth",
			"parameters": [
				{
					"name": "provider",
					"in": "path",
					"required": true,
					"type": "string",
					"enum": ["google", "facebook"]
				},
				{
					"in": "body",
					"name": "body",
					"required": true,
					"schema": { "$ref": "#/definitions/VerifyOAuthInput" }
				}
			],
			"responses": {
				"200": {"description": "Login successful", "schema": {"$ref": "#/definitions/SigninResponse"}},
				"400": {"description": "Validation error", "schema": {"$ref": "#/definitions/Error"}},
				"403": {"description": "Forbidden", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		}
	},

`
