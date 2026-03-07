package docs

// Payments API endpoints documentation
const paymentsPaths = `
	"/payments/initiate": {
		"post": {
			"tags": ["Payments"],
			"summary": "Initiate a payment",
			"description": "Initiate a payment for an order using a specified provider",
			"operationId": "initiatePayment",
			"security": [{"Bearer": []}],
			"parameters": [
				{
					"in": "body",
					"name": "body",
					"required": true,
					"schema": { "$ref": "#/definitions/InitiatePaymentRequest" }
				}
			],
			"responses": {
				"200": {
					"description": "Payment initiated successfully",
					"schema": { "$ref": "#/definitions/InitiatePaymentResponse" }
				},
				"400": { "description": "Bad Request", "schema": { "$ref": "#/definitions/Error" } },
				"401": { "description": "Unauthorized", "schema": { "$ref": "#/definitions/Error" } },
				"500": { "description": "Internal server error", "schema": { "$ref": "#/definitions/Error" } }
			}
		}
	},

	"/payments/verify": {
		"get": {
			"tags": ["Payments"],
			"summary": "Verify a payment",
			"description": "Verify a payment status using its reference",
			"operationId": "verifyPayment",
			"security": [{"Bearer": []}],
			"parameters": [
				{"name":"reference","in":"query","required":true,"type":"string"}
			],
			"responses": {
				"200": {
					"description": "Payment details retrieved",
					"schema": { "$ref": "#/definitions/PaymentResponse" }
				},
				"401": { "description": "Unauthorized", "schema": { "$ref": "#/definitions/Error" } },
				"404": { "description": "Payment not found", "schema": { "$ref": "#/definitions/Error" } },
				"500": { "description": "Internal server error", "schema": { "$ref": "#/definitions/Error" } }
			}
		}
	},

	"/payments/webhook/{provider}": {
		"post": {
			"tags": ["Payments"],
			"summary": "Handle Payment Webhook",
			"description": "Receives webhook events from payment providers",
			"operationId": "handleWebhook",
			"parameters": [
				{"name":"provider","in":"path","required":true,"type":"string","enum":["monnify","paystack","flutterwave"]}
			],
			"responses": {
				"200": { "description": "OK" },
				"400": { "description": "Bad Request" }
			}
		}
	}
`
