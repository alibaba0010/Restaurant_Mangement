## Swagger Documentation Structure

This directory contains the Swagger/OpenAPI documentation for the Restaurant Management API.

### Files Organization

- **docs.go** - Main documentation file that generates the complete Swagger specification. Contains the base template with info, host, basePath, schemes, security definitions, and assembles all paths and definitions.

- **system.go** - System endpoints documentation (e.g., healthcheck)

- **auth.docs.go** - Authentication endpoints documentation (e.g., signup, signin)

- **users.docs.go** - User management endpoints documentation (list users, get user by ID)

- **restaurants.docs.go** - Restaurant management endpoints documentation (list, create restaurants)

- **menus.docs.go** - Menu management endpoints documentation (list, create, update, delete menus)

- **categories.docs.go** - Menu category management endpoints documentation

- **orders.docs.go** - Order management endpoints documentation (list, create, update, delete orders)

- **payments.docs.go** - Payment management endpoints documentation

- **definitions.go** - Data models/schemas used across all endpoints (User, SignupInput, Restaurant, Error, etc.)

- **tags.go** - API tags organization for Swagger UI grouping

- **Performance Testing** - Load testing scripts and reports located in the project root (`load_tester.go`, `LOAD_TEST_REPORT.md`)

### Order of Documentation

The endpoints are ordered as follows:

1. **system** - `/healthcheck`
2. **Auth** - `/auth/signup`, `/auth/signin`
3. **Users** - `/user`, `/user/users`, `/user/{id}`
4. **Restaurants** - `/restaurants`
5. **Menus** - `/menus`
6. **Categories** - `/categories`
7. **Orders** - `/orders`
8. **Payments** - `/payments`

### How to Update

When adding or modifying endpoints:

1. Update the relevant file (auth.go, users.go, etc.) with the endpoint definition
2. If adding a new data model, add it to definitions.go
3. The docs.go file will compile all parts into the final Swagger spec
4. Generate Swagger docs with: `swag init`

### View Documentation

Once the server is running, access the Interactive Swagger UI at:

```
http://localhost:8001/swagger/index.html
```

### Performance & Load Testing

We maintain a custom load testing suite to ensure the API's resilience and verify rate-limiting policies.

1. **Run Load Test**: `go run load_tester.go` (or `load_test.go`)
2. **View Report**: [LOAD_TEST_REPORT.md](../LOAD_TEST_REPORT.md) in the project root.

The report includes:

- Throughput (Requests/sec)
- Latency Percentiles (P50, P90, P99)
- Success Rate (verifying rate-limit effectiveness)

### Note

The `docs.go` file is auto-generated and structured for clarity. While it says "DO NOT EDIT" at the top, it's maintained manually to support this modular documentation structure.
