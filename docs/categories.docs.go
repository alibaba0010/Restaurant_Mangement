package docs

// Categories API endpoints documentation
const categoryPaths = `
	"/categories": {
		"get": {
			"tags": ["Categories"],
			"summary": "List categories by restaurant",
			"description": "Returns all categories for a given restaurant.",
			"operationId": "listCategories",
			"parameters": [
				{"name":"restaurant_id","in":"query","description":"ID of restaurant","required":true,"type":"string"}
			],
			"responses": {
				"200": {
					"description": "Successful operation",
					"schema": { "$ref": "#/definitions/CategoryListResponse" }
				},
				"400": { "description": "Invalid restaurant ID", "schema": { "$ref": "#/definitions/Error" } },
				"500": { "description": "Internal server error", "schema": { "$ref": "#/definitions/Error" } }
			}
		},
		"post": {
			"tags": ["Categories"],
			"summary": "Create a new category",
			"description": "Adds a new category for a restaurant. Requires Management role.",
			"operationId": "createCategory",
			"security": [{"Bearer": []}],
			"parameters": [
				{
					"in": "body",
					"name": "body",
					"description": "Category creation request",
					"required": true,
					"schema": { "$ref": "#/definitions/CreateCategoryInput" }
				}
			],
			"responses": {
				"201": {
					"description": "Category created successfully",
					"schema": { "$ref": "#/definitions/CategoryResponse" }
				},
				"400": { "description": "Invalid input", "schema": { "$ref": "#/definitions/Error" } },
				"401": { "description": "Unauthorized", "schema": { "$ref": "#/definitions/Error" } },
				"403": { "description": "Forbidden", "schema": { "$ref": "#/definitions/Error" } },
				"500": { "description": "Internal server error", "schema": { "$ref": "#/definitions/Error" } }
			}
		}
	},

	"/categories/{id}": {
		"put": {
			"tags": ["Categories"],
			"summary": "Update a category",
			"description": "Updates an existing category. Requires Management role.",
			"operationId": "updateCategory",
			"security": [{"Bearer": []}],
			"parameters": [
				{"name":"id","in":"path","description":"ID of category to update","required":true,"type":"string"},
				{
					"in": "body",
					"name": "body",
					"description": "Update fields",
					"required": true,
					"schema": { "$ref": "#/definitions/UpdateCategoryInput" }
				}
			],
			"responses": {
				"200": {
					"description": "Category updated successfully",
					"schema": { "$ref": "#/definitions/CategoryResponse" }
				},
				"400": { "description": "Invalid input", "schema": { "$ref": "#/definitions/Error" } },
				"401": { "description": "Unauthorized", "schema": { "$ref": "#/definitions/Error" } },
				"403": { "description": "Forbidden", "schema": { "$ref": "#/definitions/Error" } },
				"404": { "description": "Category not found", "schema": { "$ref": "#/definitions/Error" } },
				"500": { "description": "Internal server error", "schema": { "$ref": "#/definitions/Error" } }
			}
		},
		"patch": {
			"tags": ["Categories"],
			"summary": "Partially update a category",
			"description": "Partially updates an existing category. Requires Management role.",
			"operationId": "patchCategory",
			"security": [{"Bearer": []}],
			"parameters": [
				{"name":"id","in":"path","description":"ID of category to update","required":true,"type":"string"},
				{
					"in": "body",
					"name": "body",
					"description": "Update fields",
					"required": true,
					"schema": { "$ref": "#/definitions/UpdateCategoryInput" }
				}
			],
			"responses": {
				"200": {
					"description": "Category updated successfully",
					"schema": { "$ref": "#/definitions/CategoryResponse" }
				},
				"400": { "description": "Invalid input", "schema": { "$ref": "#/definitions/Error" } },
				"401": { "description": "Unauthorized", "schema": { "$ref": "#/definitions/Error" } },
				"403": { "description": "Forbidden", "schema": { "$ref": "#/definitions/Error" } },
				"404": { "description": "Category not found", "schema": { "$ref": "#/definitions/Error" } },
				"500": { "description": "Internal server error", "schema": { "$ref": "#/definitions/Error" } }
			}
		},
		"delete": {
			"tags": ["Categories"],
			"summary": "Delete a category",
			"description": "Deletes a category and its associations. Requires Management role.",
			"operationId": "deleteCategory",
			"security": [{"Bearer": []}],
			"parameters": [
				{"name":"id","in":"path","description":"ID of category to delete","required":true,"type":"string"}
			],
			"responses": {
				"200": {
					"description": "Category deleted successfully",
					"schema": { "$ref": "#/definitions/MessageResponse" }
				},
				"401": { "description": "Unauthorized", "schema": { "$ref": "#/definitions/Error" } },
				"403": { "description": "Forbidden", "schema": { "$ref": "#/definitions/Error" } },
				"404": { "description": "Category not found", "schema": { "$ref": "#/definitions/Error" } },
				"500": { "description": "Internal server error", "schema": { "$ref": "#/definitions/Error" } }
			}
		}
	},
`
