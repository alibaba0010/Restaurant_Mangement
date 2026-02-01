## Swagger Documentation Structure

This directory contains the Swagger/OpenAPI documentation for the Restaurant Management API.

### Files Organization

- **docs.go** - Main documentation file that generates the complete Swagger specification. Contains the base template with info, host, basePath, schemes, security definitions, and assembles all paths and definitions.

- **system.go** - System endpoints documentation (e.g., healthcheck)

- **auth.go** - Authentication endpoints documentation (e.g., signup, signin)

- **users.go** - User management endpoints documentation (list users, get user by ID)

- **restaurants.go** - Restaurant management endpoints documentation (list, create restaurants)

- **menus.go** - Menu management endpoints documentation (list, create, update, delete menus)

- **orders.go** - Order management endpoints documentation (list, create, update, delete orders)

- **payments.go** - Payment management endpoints documentation

- **definitions.go** - Data models/schemas used across all endpoints (User, SignupInput, Restaurant, Error, etc.)

- **tags.go** - API tags organization for Swagger UI grouping

### Order of Documentation

The endpoints are ordered as follows:

1. **system** - `/healthcheck`
2. **Auth** - `/auth/signup`
3. **Users** - `/users`, `/users/{id}`
4. **Restaurants** - `/restaurants`

### How to Update

When adding or modifying endpoints:

1. Update the relevant file (auth.go, users.go, etc.) with the endpoint definition
2. If adding a new data model, add it to definitions.go
3. The docs.go file will compile all parts into the final Swagger spec
4. Generate Swagger docs with: `swag init`

### View Documentation

Once running, access the Swagger UI at:

```
http://localhost:3000/swagger/
```

### Note

The `docs.go` file is auto-generated and structured for clarity. While it says "DO NOT EDIT" at the top, it's maintained manually to support this modular documentation structure.
