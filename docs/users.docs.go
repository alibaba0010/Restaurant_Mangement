package docs

// Users API endpoints documentation

const usersPaths = `

	"/user": {
		"get": {
			"tags": ["Users"],
			"summary": "Get current authenticated user",
			"description": "Retrieve info about the currently authenticated user",
			"operationId": "currentUser",
			"security": [ { "Bearer": [] } ],
			"responses": {
				"200": { "description": "Successful operation", "schema": { "$ref": "#/definitions/User" } },
				"401": { "description": "Unauthorized", "schema": { "$ref": "#/definitions/Error" } },
				"500": { "description": "Internal server error", "schema": { "$ref": "#/definitions/Error" } }
			}
		},
		"patch": {
			"tags": ["Users"],
			"summary": "Update current user's address",
			"description": "Updates the authenticated user's address",
			"operationId": "updateUserAddress",
			"security": [ { "Bearer": [] } ],
			"parameters": [{"in":"body","name":"body","required":true,"schema":{"$ref":"#/definitions/UpdateAddressInput"}}],
			"responses": {
				"200": {"description":"Updated","schema":{"$ref":"#/definitions/UpdateAddressResponse"}},
				"400": {"description":"Validation error","schema":{"$ref":"#/definitions/Error"}},
				"401": {"description":"Unauthorized","schema":{"$ref":"#/definitions/Error"}},
				"500": {"description":"Internal server error","schema":{"$ref":"#/definitions/Error"}}
			}
		},
		"post": {
			"tags": ["Users"],
			"summary": "Logout current user",
			"description": "Revokes refresh tokens and clears cookie",
			"operationId": "logout",
			"security": [ { "Bearer": [] } ],
			"responses": {
				"200": {"description":"Logged out","schema":{"$ref":"#/definitions/LogoutResponse"}},
				"401": {"description":"Unauthorized","schema":{"$ref":"#/definitions/Error"}},
				"500": {"description":"Internal server error","schema":{"$ref":"#/definitions/Error"}}
			}
		}
	},

	"/user/{id}": {
		"get": {
			"tags": ["Users"],
			"summary": "Get user by ID",
			"description": "Returns a single user (admin role required)",
			"operationId": "getUser",
			"security": [ { "Bearer": [] } ],
			"parameters": [{"name":"id","in":"path","description":"ID of user to return","required":true,"type":"string"}],
			"responses": {
				"200": {"description":"Successful operation","schema":{"$ref":"#/definitions/User"}},
				"401": {"description":"Unauthorized","schema":{"$ref":"#/definitions/Error"}},
				"403": {"description":"Forbidden - requires admin role","schema":{"$ref":"#/definitions/Error"}},
				"404": {"description":"User not found","schema":{"$ref":"#/definitions/Error"}},
				"500": {"description":"Internal server error","schema":{"$ref":"#/definitions/Error"}}
			}
		}
	},

	"/users": {
		"get": {
			"tags": ["Users"],
			"summary": "Get all users",
			"description": "Returns a paginated list of users (admin role required)",
			"operationId": "listUsers",
			"security": [ { "Bearer": [] } ],
			"parameters": [
				{"name":"page","in":"query","type":"integer"},
				{"name":"page_size","in":"query","type":"integer"},
				{"name":"q","in":"query","type":"string","description":"search name/email"},
				{"name":"role","in":"query","type":"string"},
				{"name":"sort_by","in":"query","type":"string"},
				{"name":"order","in":"query","type":"string","enum":["asc","desc"]}
			],
			"responses": {
				"200": {"description":"Successful operation","schema":{"$ref":"#/definitions/UsersListResponse"}},
				"401": {"description":"Unauthorized","schema":{"$ref":"#/definitions/Error"}},
				"403": {"description":"Forbidden - requires admin role","schema":{"$ref":"#/definitions/Error"}},
				"500": {"description":"Internal server error","schema":{"$ref":"#/definitions/Error"}}
			}
		}
	},
`
