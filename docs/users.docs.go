package docs

// Users API endpoints documentation

const usersPaths = `

	"/user": {
		"get": {
			"tags": [
				"Users"
			],
			"summary": "Get current authenticated user",
			"description": "Retrieve info about the currently authenticated user",
			"operationId": "currentUser",
			"security": [ { "Bearer": [] } ],
			"responses": {
				"200": { "description": "Successful operation", "schema": { "$ref": "#/definitions/User" } },
				"401": { "description": "Unauthorized", "schema": { "$ref": "#/definitions/Error" } },
				"500": { "description": "Internal server error", "schema": { "$ref": "#/definitions/Error" } }
			}
		}
	},

	"/users": {
		"get": {
			"tags": [
				"Users"
			],
			"summary": "List all users",
			"description": "Returns a list of users in the system",
			"operationId": "listUsers",
			"security": [
				{
					"Bearer": []
				}
			],
			"responses": {
				"200": {
					"description": "Successful operation",
					"schema": {
						"type": "object",
						"properties": {
							"title": { "type": "string", "example": "Success" },
							"data": { "type": "array", "items": { "$ref": "#/definitions/User" } }
						},
						"required": ["title","data"]
					}
				},
				"401": {
					"description": "Unauthorized",
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
	"/users/{id}": {
		"get": {
			"tags": [
				"Users"
			],
			"summary": "Get user by ID",
			"description": "Returns a single user",
			"operationId": "getUser",
			"parameters": [
				{
					"name": "id",
					"in": "path",
					"description": "ID of user to return",
					"required": true,
					"type": "string"
				}
			],
			"responses": {
				"200": {
					"description": "Successful operation",
					"schema": {
						"$ref": "#/definitions/User"
					}
				},
				"404": {
					"description": "User not found",
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
`
