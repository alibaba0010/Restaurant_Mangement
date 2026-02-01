package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": ["http", "https"],
    "swagger": "2.0",
    "info": {
        "description": "A robust REST API built with Go, featuring PostgreSQL integration, structured logging, and Swagger documentation for managing restaurant operations.",
        "title": "Restaurant Management API",
        "contact": {},
        "version": "1.0"
    },
    "host": "localhost:8001",
    "basePath": "/api/v1",
    "paths": {
` + systemPaths + authPaths + usersPaths + restaurantsPaths + menusPaths + ordersPaths + paymentsPaths + `
    },
    "securityDefinitions": {
        "Bearer": {
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    },
	` + tags + `,
	` + definitions + `
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8001",
	BasePath:         "/api/v1",
	Schemes:          []string{"http", "https"},
	Title:            "Restaurant Management API",
	Description:      "A robust REST API built with Go, featuring PostgreSQL integration, structured logging, and Swagger documentation for managing restaurant operations.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
