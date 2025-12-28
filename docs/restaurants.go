package docs

// Restaurants API endpoints documentation
const restaurantsPaths = `
	"/restaurants": {
		"get": {
			"tags": ["Restaurants"],
			"summary": "List all restaurants",
			"description": "Returns a paginated list of restaurants",
			"operationId": "listRestaurants",
			"parameters": [
				{"name":"page","in":"query","type":"integer"},
				{"name":"page_size","in":"query","type":"integer"},
				{"name":"q","in":"query","type":"string","description":"search name/description"}
			],
			"responses": {
				"200": {
					"description": "Successful operation",
					"schema": { "$ref": "#/definitions/RestaurantsListResponse" }
				},
				"500": {
					"description": "Internal server error",
					"schema": { "$ref": "#/definitions/Error" }
				}
			}
		},
		"post": {
			"tags": ["Restaurants"],
			"summary": "Create a new restaurant",
			"description": "Adds a new restaurant to the system",
			"operationId": "createRestaurant",
			"security": [{"Bearer": []}],
			"parameters": [
				{
					"in": "body",
					"name": "restaurant",
					"description": "Restaurant object that needs to be created",
					"required": true,
					"schema": { "$ref": "#/definitions/RestaurantInput" }
				}
			],
			"responses": {
				"201": {
					"description": "Restaurant created successfully",
					"schema": { "$ref": "#/definitions/Restaurant" }
				},
				"400": {"description": "Invalid input", "schema": {"$ref": "#/definitions/Error"}},
				"401": {"description": "Unauthorized", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		}
	},

	"/restaurants/{id}": {
		"get": {
			"tags": ["Restaurants"],
			"summary": "Get restaurant by ID",
			"description": "Returns a single restaurant",
			"operationId": "getRestaurant",
			"parameters": [
				{"name":"id","in":"path","description":"ID of restaurant to return","required":true,"type":"string"}
			],
			"responses": {
				"200": {
					"description": "Successful operation",
					"schema": { "$ref": "#/definitions/Restaurant" }
				},
				"404": {"description": "Restaurant not found", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		},
		"put": {
			"tags": ["Restaurants"],
			"summary": "Update a restaurant",
			"description": "Updates an existing restaurant's details",
			"operationId": "updateRestaurant",
			"security": [{"Bearer": []}],
			"parameters": [
				{"name":"id","in":"path","description":"ID of restaurant to update","required":true,"type":"string"},
				{
					"in": "body",
					"name": "restaurant",
					"description": "Updated restaurant object",
					"required": true,
					"schema": { "$ref": "#/definitions/RestaurantInput" }
				}
			],
			"responses": {
				"200": {
					"description": "Restaurant updated successfully",
					"schema": { "$ref": "#/definitions/Restaurant" }
				},
				"400": {"description": "Invalid input", "schema": {"$ref": "#/definitions/Error"}},
				"401": {"description": "Unauthorized", "schema": {"$ref": "#/definitions/Error"}},
				"404": {"description": "Restaurant not found", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		},
		"patch": {
			"tags": ["Restaurants"],
			"summary": "Partially update a restaurant",
			"description": "Updates an existing restaurant's details partially",
			"operationId": "patchRestaurant",
			"security": [{"Bearer": []}],
			"parameters": [
				{"name":"id","in":"path","description":"ID of restaurant to update","required":true,"type":"string"},
				{
					"in": "body",
					"name": "restaurant",
					"description": "Fields to update",
					"required": true,
					"schema": { "$ref": "#/definitions/RestaurantInput" }
				}
			],
			"responses": {
				"200": {
					"description": "Restaurant updated successfully",
					"schema": { "$ref": "#/definitions/Restaurant" }
				},
				"400": {"description": "Invalid input", "schema": {"$ref": "#/definitions/Error"}},
				"401": {"description": "Unauthorized", "schema": {"$ref": "#/definitions/Error"}},
				"404": {"description": "Restaurant not found", "schema": {"$ref": "#/definitions/Error"}},
				"500": {"description": "Internal server error", "schema": {"$ref": "#/definitions/Error"}}
			}
		}
	},

`
