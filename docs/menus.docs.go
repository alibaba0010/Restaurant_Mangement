package docs

// Menus API endpoints documentation
const menusPaths = `
	"/menus": {
		"get": {
			"tags": ["Menus"],
			"summary": "List all menu items",
			"description": "Returns a paginated list of menu items with optional filters",
			"operationId": "listMenus",
			"parameters": [
				{"name":"page","in":"query","type":"integer"},
				{"name":"page_size","in":"query","type":"integer"},
				{"name":"q","in":"query","type":"string","description":"search name/description"},
				{"name":"restaurant_id","in":"query","type":"string","description":"filter by restaurant"},
				{"name":"min_price","in":"query","type":"number"},
				{"name":"max_price","in":"query","type":"number"},
				{"name":"is_available","in":"query","type":"boolean"}
			],
			"responses": {
				"200": {
					"description": "Successful operation",
					"schema": { "$ref": "#/definitions/MenusListResponse" }
				},
				"500": {
					"description": "Internal server error",
					"schema": { "$ref": "#/definitions/Error" }
				}
			}
		},
		"post": {
			"tags": ["Menus"],
			"summary": "Create a new menu item",
			"description": "Adds a new menu item to a restaurant",
			"operationId": "createMenu",
			"security": [{"Bearer": []}],
			"parameters": [
				{
					"in": "body",
					"name": "menu",
					"description": "Menu item object that needs to be created",
					"required": true,
					"schema": { "$ref": "#/definitions/CreateMenuInput" }
				}
			],
			"responses": {
				"201": {
					"description": "Menu item created successfully",
					"schema": { "$ref": "#/definitions/MenuResponse" }
				},
				"400": {"description": "Invalid input", "schema": {"$ref": "#/definitions/Error"}},
				"401": {"description": "Unauthorized", "schema": {"$ref": "#/definitions/Error"}},
				"403": {"description": "Forbidden", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		}
	},

	"/menus/{id}": {
		"get": {
			"tags": ["Menus"],
			"summary": "Get menu item by ID",
			"description": "Returns a single menu item",
			"operationId": "getMenu",
			"parameters": [
				{"name":"id","in":"path","description":"ID of menu item to return","required":true,"type":"string"}
			],
			"responses": {
				"200": {
					"description": "Successful operation",
					"schema": { "$ref": "#/definitions/MenuResponse" }
				},
				"404": {"description": "Menu item not found", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		},
		"patch": {
			"tags": ["Menus"],
			"summary": "Partially update a menu item",
			"description": "Updates an existing menu item's details partially",
			"operationId": "patchMenu",
			"security": [{"Bearer": []}],
			"parameters": [
				{"name":"id","in":"path","description":"ID of menu item to update","required":true,"type":"string"},
				{
					"in": "body",
					"name": "menu",
					"description": "Fields to update",
					"required": true,
					"schema": { "$ref": "#/definitions/UpdateMenuInput" }
				}
			],
			"responses": {
				"200": {
					"description": "Menu item updated successfully",
					"schema": { "$ref": "#/definitions/MenuResponse" }
				},
				"400": {"description": "Invalid input", "schema": {"$ref": "#/definitions/Error"}},
				"401": {"description": "Unauthorized", "schema": {"$ref": "#/definitions/Error"}},
				"403": {"description": "Forbidden", "schema": {"$ref": "#/definitions/Error"}},
				"404": {"description": "Menu item not found", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		}
	},

	"/menus/upload": {
		"post": {
			"tags": ["Menus"],
			"summary": "Upload menu media",
			"description": "Uploads an image or video for a menu item",
			"operationId": "uploadMenuMedia",
			"security": [{"Bearer": []}],
			"consumes": ["multipart/form-data"],
			"parameters": [
				{
					"name": "file",
					"in": "formData",
					"description": "File to upload",
					"required": true,
					"type": "file"
				}
			],
			"responses": {
				"200": {
					"description": "File uploaded successfully",
					"schema": {
						"type": "object",
						"properties": {
							"title": { "type": "string" },
							"data": {
								"type": "object",
								"properties": {
									"url": { "type": "string" }
								}
							}
						}
					}
				},
				"400": {"description": "Invalid input", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		}
	},
`
