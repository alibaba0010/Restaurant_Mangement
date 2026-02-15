package docs

// Orders API endpoints documentation
const ordersPaths = `
	"/orders": {
		"post": {
			"tags": ["Orders"],
			"summary": "Create a new order",
			"description": "Places a new order for a restaurant. Status starts as 'pending'.",
			"operationId": "createOrder",
			"security": [{"Bearer": []}],
			"parameters": [
				{
					"in": "body",
					"name": "body",
					"description": "Order creation request",
					"required": true,
					"schema": { "$ref": "#/definitions/CreateOrderInput" }
				}
			],
			"responses": {
				"201": {
					"description": "Order created successfully",
					"schema": { "$ref": "#/definitions/OrderResponse" }
				},
				"400": { "description": "Invalid input", "schema": { "$ref": "#/definitions/Error" } },
				"401": { "description": "Unauthorized", "schema": { "$ref": "#/definitions/Error" } },
				"500": { "description": "Internal server error", "schema": { "$ref": "#/definitions/Error" } }
			}
		},
		"get": {
			"tags": ["Orders"],
			"summary": "List authenticated user's orders",
			"description": "Returns a paginated list of orders placed by the current user.",
			"operationId": "listUserOrders",
			"security": [{"Bearer": []}],
			"parameters": [
				{"name":"page","in":"query","type":"integer"},
				{"name":"page_size","in":"query","type":"integer"},
				{"name":"cursor","in":"query","type":"string","description":"pagination cursor"}
			],
			"responses": {
				"200": {
					"description": "Successful operation",
					"schema": { "$ref": "#/definitions/UserOrdersResponse" }
				},
				"401": { "description": "Unauthorized", "schema": { "$ref": "#/definitions/Error" } },
				"500": { "description": "Internal server error", "schema": { "$ref": "#/definitions/Error" } }
			}
		}
	},

	"/orders/{id}": {
		"get": {
			"tags": ["Orders"],
			"summary": "Get order details by ID",
			"description": "Retrieves an order by its UUID. Users can only see their own orders unless they are Admin.",
			"operationId": "getOrder",
			"security": [{"Bearer": []}],
			"parameters": [
				{"name":"id","in":"path","description":"ID of order to return","required":true,"type":"string"}
			],
			"responses": {
				"200": {
					"description": "Successful operation",
					"schema": { "$ref": "#/definitions/OrderResponse" }
				},
				"403": { "description": "Forbidden", "schema": { "$ref": "#/definitions/Error" } },
				"404": { "description": "Order not found", "schema": { "$ref": "#/definitions/Error" } },
				"500": { "description": "Internal server error", "schema": { "$ref": "#/definitions/Error" } }
			}
		}
	},

	"/orders/{id}/status": {
		"patch": {
			"tags": ["Orders"],
			"summary": "Update order status",
			"description": "Updates the status of an order. Requires Admin or Management role (if owner).",
			"operationId": "updateOrderStatus",
			"security": [{"Bearer": []}],
			"parameters": [
				{"name":"id","in":"path","description":"ID of order to update","required":true,"type":"string"},
				{
					"in": "body",
					"name": "body",
					"description": "New status",
					"required": true,
					"schema": { "$ref": "#/definitions/UpdateOrderStatusInput" }
				}
			],
			"responses": {
				"200": {
					"description": "Status updated successfully",
					"schema": {
						"type": "object",
						"properties": {
							"message": { "type": "string" }
						}
					}
				},
				"403": { "description": "Insufficient permissions", "schema": { "$ref": "#/definitions/Error" } },
				"404": { "description": "Order not found", "schema": { "$ref": "#/definitions/Error" } },
				"500": { "description": "Internal server error", "schema": { "$ref": "#/definitions/Error" } }
			}
		}
	},
`
