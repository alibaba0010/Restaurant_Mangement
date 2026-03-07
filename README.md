# GourmetHub — Backend API

A production-grade REST API powering the GourmetHub restaurant management platform. Built with Go 1.24, PostgreSQL, Redis, and Redpanda event streaming.

---

## Table of Contents

- [Architecture Overview](#-architecture-overview)
- [Key Features](#-key-features)
- [Tech Stack](#-tech-stack)
- [Prerequisites](#-prerequisites)
- [Quick Start](#-quick-start)
- [Environment Variables](#-environment-variables)
- [Database Migrations](#-database-migrations)
- [Project Structure](#-project-structure)
- [API Reference](#-api-reference)
- [Middleware Pipeline](#-middleware-pipeline)
- [Event-Driven Architecture](#-event-driven-architecture)
- [Payment Processing](#-payment-processing)
- [Concurrency & Data Integrity](#-concurrency--data-integrity)
- [Logging & Observability](#-logging--observability)
- [Development](#-development)
- [Contributing](#-contributing)
- [License](#-license)

---

## 🏗 Architecture Overview

```
┌─────────────┐     ┌──────────────────────────────────────────────────┐
│   Client    │────▶│  Go HTTP Server (gorilla/mux)                   │
│  (Next.js)  │◀────│                                                  │
└─────────────┘     │  ┌──────────┐  ┌───────────┐  ┌──────────────┐  │
                    │  │   Auth   │  │Restaurant │  │   Orders     │  │
                    │  │  Module  │  │ & Menus   │  │   Module     │  │
                    │  └────┬─────┘  └─────┬─────┘  └──────┬───────┘  │
                    │       │              │               │          │
                    │  ┌────▼──────────────▼───────────────▼───────┐  │
                    │  │         PostgreSQL (Bun ORM)              │  │
                    │  └──────────────────────────────────────────┘  │
                    │  ┌────────────┐  ┌───────────────────────────┐  │
                    │  │   Redis    │  │  Redpanda (Event Bus)     │  │
                    │  │  (Cache)   │  │  Kafka-compatible         │  │
                    │  └────────────┘  └───────────────────────────┘  │
                    │  ┌────────────┐  ┌───────────────────────────┐  │
                    │  │  AWS S3    │  │ Payment Providers         │  │
                    │  │ CloudFront │  │ Paystack/Monnify/Flutter  │  │
                    │  └────────────┘  └───────────────────────────┘  │
                    └──────────────────────────────────────────────────┘
```

---

## ✨ Key Features

### Authentication & Authorization

- **JWT Token Pairs** — Access + Refresh tokens with rotation on refresh
- **Role-Based Access Control (RBAC)** — Three tiers: `user`, `management`, `admin`
- **OAuth 2.0** — Google sign-in with state validation and CSRF-protected cookies
- **Email Verification** — Redis-backed verification flow with token expiry
- **Password Reset** — Secure reset links via email (SMTP with `go-mail`)
- **Cloudflare Turnstile** — Bot protection on authentication endpoints
- **Argon2id Password Hashing** — Industry-standard password security

### Restaurant & Menu Management

- **Restaurant CRUD** — Full lifecycle with address geocoding (`geo-golang`)
- **Menu Items** — Rich menu items with images, video, dietary flags, allergens, and stock tracking
- **Menu Categories** — AI-assisted category suggestions; up to 5 categories per menu item
- **Media Uploads** — Direct S3 presigned URL uploads and server-proxied multipart uploads for large files
- **CloudFront CDN** — Served via CloudFront distribution for global content delivery
- **Redis Cache** — SHA-256 hashed cache keys for menu listings with automatic invalidation on mutations

### Order Management

- **Order Lifecycle** — Full state machine: `pending` → `confirmed` → `preparing` → `ready` → `completed`
- **Service Charge** — Automatic calculation: **10% on orders under 100, 5% on orders 100+**
- **Stock Management** — Pessimistic locking (`SELECT ... FOR UPDATE`) prevents overselling during concurrent orders
- **Batch Menu Validation** — Single query fetches all menu items, preventing N+1 during order creation
- **Cursor-Based Pagination** — Timestamp-based cursors for efficient order history traversal
- **Order Cancellation** — Users can cancel pending orders; management can update any status

### Payment Processing

- **Multi-Provider Support** — Paystack, Monnify, and Flutterwave via a unified provider interface
- **Payment Lifecycle** — `pending` → `processing` → `success` / `failed` with event-driven status updates
- **Settlement Tracking** — Automatic commission calculation (platform fee) and restaurant share on successful payments
- **Webhook Verification** — Provider-specific signature validation for secure webhook processing
- **Webhook Audit Logs** — Deduplication via `provider_event_id` with full payload logging
- **Refund Support** — Data model for refund tracking with provider refund IDs

### Infrastructure

- **Event Streaming** — Redpanda (Kafka-compatible) for `order.created`, `order.status_updated`, `payment_successful`, `payment_failed` events
- **Graceful Shutdown** — Signal handling with context cancellation for clean resource teardown
- **Rate Limiting** — Per-IP token bucket rate limiter with automatic bucket cleanup
- **CORS** — Configurable origin allowlist with localhost dev convenience
- **Structured Logging** — Uber `zap` with request/response logging middleware
- **Swagger Docs** — Auto-generated OpenAPI documentation at `/swagger/index.html`

---

## 🧰 Tech Stack

| Category       | Technology                  |
| -------------- | --------------------------- |
| Language       | Go 1.24                     |
| HTTP Router    | gorilla/mux                 |
| ORM            | Bun (uptrace/bun)           |
| Database       | PostgreSQL (pgx/v5 driver)  |
| Cache          | Redis (go-redis/v9)         |
| Event Bus      | Redpanda via franz-go       |
| Object Storage | AWS S3 + CloudFront         |
| Auth           | golang-jwt/jwt/v5, Argon2id |
| Validation     | go-playground/validator/v10 |
| Decimal Math   | shopspring/decimal          |
| Logging        | uber-go/zap                 |
| Email          | wneessen/go-mail            |
| Geocoding      | codingsince1985/geo-golang  |
| API Docs       | swaggo/swag + http-swagger  |
| Hot Reload     | cosmtrek/air                |

---

## 📋 Prerequisites

| Dependency | Version | Purpose                          |
| ---------- | ------- | -------------------------------- |
| Go         | 1.24+   | Runtime                          |
| PostgreSQL | 14+     | Primary datastore                |
| Redis      | 7+      | Caching, token storage           |
| Redpanda   | Latest  | Event streaming (Kafka protocol) |
| Docker     | 20+     | Container runtime (for Redpanda) |

---

## 🚀 Quick Start

### 1. Clone and install dependencies

```bash
git clone https://github.com/alibaba0010/postgres-api.git
cd postgres-api
go mod download
```

### 2. Start infrastructure services

```bash
docker compose up -d
```

This launches Redpanda (port `29092`) and Redpanda Console (port `8081`).

### 3. Configure environment variables

Copy the template below into a `.env` file at the project root. See the [Environment Variables](#-environment-variables) section for details.

### 4. Run database migrations

```bash
./migrate.sh init    # First-time setup: create migration tracking table
./migrate.sh up      # Apply all pending migrations
```

### 5. Start the server

**With hot reload (recommended for development):**

```bash
go install github.com/air-verse/air@latest
air
```

**Without hot reload:**

```bash
go run main.go
```

The API will be available at `http://localhost:8001/api/v1`.  
Swagger UI: `http://localhost:8001/swagger/index.html`.

---

## 🔧 Environment Variables

Create a `.env` file in the project root:

```env
# ── Server ──────────────────────────────────────────
PORT=8001
FRONTEND_URL=http://localhost:8000

# ── PostgreSQL ──────────────────────────────────────
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=postgres
DB_PASSWORD=password
DB_NAME=gourmethub

# ── Redis ───────────────────────────────────────────
REDIS_HOST=localhost
REDIS_PORT=6379

# ── Redpanda / Kafka ────────────────────────────────
REDPANDA_BROKERS=localhost:29092

# ── JWT ─────────────────────────────────────────────
JWT_SECRET=your_jwt_secret_here

# ── SMTP (Email) ────────────────────────────────────
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your_email@gmail.com
SMTP_PASS=your_app_password

# ── OAuth (Google) ──────────────────────────────────
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_REDIRECT_URL=http://localhost:8000/auth/callback

# ── Cloudflare Turnstile ────────────────────────────
TURNSTILE_SECRET_KEY=your_turnstile_secret

# ── AWS S3 & CloudFront ─────────────────────────────
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_REGION=us-east-1
AWS_BUCKET_NAME=your_bucket_name
AWS_CLOUDFRONT_DOMAIN=your_domain.cloudfront.net

# ── Payment Providers ───────────────────────────────
PAYSTACK_SECRET_KEY=sk_test_...
MONNIFY_API_KEY=MK_...
MONNIFY_SECRET_KEY=...
MONNIFY_CONTRACT_CODE=...
FLUTTERWAVE_SECRET_KEY=mx_...
FLUTTERWAVE_PUBLIC_KEY=...
FLUTTERWAVE_HASH=your_webhook_hash
```

---

## 🗃 Database Migrations

Migrations are managed via Bun's migration tool, wrapped in `migrate.sh`:

| Command                      | Description                                           |
| ---------------------------- | ----------------------------------------------------- |
| `./migrate.sh init`          | Create the migration tracking table (first-time only) |
| `./migrate.sh create <name>` | Generate a new up/down migration pair                 |
| `./migrate.sh up`            | Apply all pending migrations                          |
| `./migrate.sh down`          | Rollback the last applied migration                   |
| `./migrate.sh status`        | Show current migration state                          |

Migration files live in `internal/migration/migrations/` and use the `.tx.up.sql` / `.tx.down.sql` naming convention to run within database transactions.

---

## 📁 Project Structure

```
server/
├── main.go                          # Application entry point & graceful shutdown
├── docker-compose.yaml              # Redpanda + Console containers
├── migrate.sh                       # Migration CLI wrapper
├── .air.toml                        # Hot-reload configuration
├── .env                             # Environment variables (not committed)
│
├── cmd/
│   └── migrate/                     # Migration CLI entry point
│
├── docs/                            # Auto-generated Swagger documentation
│
└── internal/
    ├── config/                      # Environment configuration loader
    ├── database/                    # PostgreSQL and Redis connection managers
    │
    ├── auth/
    │   ├── controllers/             # HTTP handlers (signup, signin, OAuth, etc.)
    │   ├── dto/                     # Request/response data transfer objects
    │   ├── models/                  # User, RefreshToken ORM models
    │   ├── routes/                  # Auth & User route registration
    │   └── services/                # Business logic (JWT, Argon2, OAuth, email)
    │
    ├── restaurants/
    │   ├── controllers/             # Restaurant, Menu, Category handlers
    │   ├── dto/                     # Input/output DTOs with validation tags
    │   ├── models/                  # Restaurant, Menu, Category ORM models
    │   ├── repositories/            # Database queries with locking support
    │   ├── routes/                  # Route registration per entity
    │   ├── services/                # Business logic, S3 uploads, cache
    │   └── subscribers/             # Event handlers for menu-related events
    │
    ├── orders/
    │   ├── controllers/             # Order CRUD handlers
    │   ├── dto/                     # Order DTOs with service charge fields
    │   ├── models/                  # Order, OrderItem ORM models
    │   ├── repositories/            # Order persistence with FOR UPDATE locking
    │   ├── routes/                  # Order route registration
    │   ├── services/                # Order lifecycle, service charge calculation
    │   └── subscribers/             # Payment event handlers (confirm/cancel)
    │
    ├── payments/
    │   ├── controllers/             # Payment initiation, verification, webhooks
    │   ├── dto/                     # Payment request/response DTOs
    │   ├── events/                  # Payment event payloads
    │   ├── models/                  # Payment, Refund, WebhookLog, Settlement models
    │   ├── providers/               # Provider interface + Paystack/Monnify/Flutter
    │   ├── repositories/            # Payment & settlement persistence
    │   ├── routes/                  # Payment route registration
    │   └── services/                # Payment orchestration & settlement
    │
    ├── common/
    │   ├── address/                 # Address validation & geocoding service
    │   ├── dto/                     # Shared DTOs (MessageResponse, pagination)
    │   ├── errors/                  # Centralized error types & HTTP error handler
    │   ├── events/                  # Producer, Consumer, Event interfaces (Redpanda)
    │   ├── guards/                  # Auth middleware, RBAC, Turnstile, ownership checks
    │   ├── logger/                  # Zap logger initialization & HTTP logger middleware
    │   ├── middlewares/             # CORS, rate limiting, recovery middleware
    │   ├── s3/                      # AWS S3 service (upload, presign, multipart)
    │   └── types/                   # Shared enums (roles, statuses, order/payment types)
    │
    ├── migration/
    │   └── migrations/              # SQL migration files (.tx.up.sql / .tx.down.sql)
    │
    └── utils/                       # Helpers (validation, cookies, JSON, IP extraction)
```

---

## 📡 API Reference

All endpoints are prefixed with `/api/v1`. Full interactive documentation is available at `/swagger/index.html`.

### Authentication

| Method | Endpoint                  | Auth      | Description                         |
| ------ | ------------------------- | --------- | ----------------------------------- |
| `POST` | `/auth/signup`            | Turnstile | Register a new user                 |
| `POST` | `/auth/signin`            | Turnstile | Authenticate and receive token pair |
| `GET`  | `/auth/verify?token=`     | —         | Activate account via email link     |
| `POST` | `/auth/refresh`           | Cookie    | Rotate access + refresh tokens      |
| `POST` | `/auth/resend`            | Turnstile | Resend verification email           |
| `POST` | `/auth/forgot-password`   | Turnstile | Request password reset link         |
| `POST` | `/auth/reset-password`    | —         | Reset password with token           |
| `GET`  | `/auth/{provider}/login`  | —         | Initiate OAuth flow (Google)        |
| `POST` | `/auth/{provider}/verify` | —         | Complete OAuth callback             |

### Users

| Method  | Endpoint          | Auth   | Roles             | Description                     |
| ------- | ----------------- | ------ | ----------------- | ------------------------------- |
| `GET`   | `/user`           | Bearer | Any               | Get current user profile        |
| `PATCH` | `/user`           | Bearer | Any               | Update profile (address, phone) |
| `POST`  | `/user/logout`    | Bearer | Any               | Revoke all refresh tokens       |
| `GET`   | `/user/users`     | Bearer | Admin, Management | List all users (paginated)      |
| `GET`   | `/user/{id}`      | Bearer | Admin, Management | Get user by ID                  |
| `PATCH` | `/user/{id}/role` | Bearer | Admin, Management | Update user role/status         |

### Restaurants

| Method  | Endpoint            | Auth   | Roles        | Description                         |
| ------- | ------------------- | ------ | ------------ | ----------------------------------- |
| `POST`  | `/restaurants`      | Bearer | Management   | Create a restaurant                 |
| `GET`   | `/restaurants`      | Bearer | Any          | List restaurants (cursor-paginated) |
| `GET`   | `/restaurants/{id}` | Bearer | Any          | Get restaurant details              |
| `PATCH` | `/restaurants/{id}` | Bearer | Owner, Admin | Update restaurant                   |

### Menus

| Method   | Endpoint                       | Auth   | Roles      | Description                                     |
| -------- | ------------------------------ | ------ | ---------- | ----------------------------------------------- |
| `POST`   | `/menus`                       | Bearer | Management | Create a menu item                              |
| `GET`    | `/menus`                       | —      | Public     | List menus (filtered, cached, cursor-paginated) |
| `GET`    | `/menus/{id}`                  | —      | Public     | Get menu item by ID                             |
| `PATCH`  | `/menus/{id}`                  | Bearer | Management | Update a menu item                              |
| `DELETE` | `/menus/{id}`                  | Bearer | Management | Delete a menu item                              |
| `POST`   | `/menus/upload`                | Bearer | Management | Direct server-side media upload                 |
| `GET`    | `/menus/upload-url`            | Bearer | Management | Get S3 presigned upload URL                     |
| `POST`   | `/menus/multipart/initiate`    | Bearer | Management | Start multipart upload                          |
| `GET`    | `/menus/multipart/part-url`    | Bearer | Management | Get presigned URL for a part                    |
| `POST`   | `/menus/multipart/upload-part` | Bearer | Management | Upload a chunk via server proxy                 |
| `POST`   | `/menus/multipart/complete`    | Bearer | Management | Finalize multipart upload                       |
| `POST`   | `/menus/multipart/abort`       | Bearer | Management | Abort multipart upload                          |

### Categories

| Method   | Endpoint                     | Auth   | Roles      | Description                      |
| -------- | ---------------------------- | ------ | ---------- | -------------------------------- |
| `GET`    | `/categories?restaurant_id=` | —      | Public     | List categories for a restaurant |
| `POST`   | `/categories`                | Bearer | Management | Create a category                |
| `PUT`    | `/categories/{id}`           | Bearer | Management | Update a category                |
| `DELETE` | `/categories/{id}`           | Bearer | Management | Delete a category                |

### Orders

| Method  | Endpoint              | Auth   | Roles             | Description                             |
| ------- | --------------------- | ------ | ----------------- | --------------------------------------- |
| `POST`  | `/orders`             | Bearer | Any               | Place a new order (auto service charge) |
| `GET`   | `/orders`             | Bearer | Any               | List user's orders (cursor-paginated)   |
| `GET`   | `/orders/{id}`        | Bearer | Owner, Admin      | Get order details                       |
| `PATCH` | `/orders/{id}/status` | Bearer | Admin, Management | Update order status                     |

### Payments

| Method | Endpoint                       | Auth   | Description                                    |
| ------ | ------------------------------ | ------ | ---------------------------------------------- |
| `POST` | `/payments/initiate`           | Bearer | Initialize payment (returns authorization URL) |
| `GET`  | `/payments/verify?reference=`  | Bearer | Verify payment status                          |
| `POST` | `/payments/webhook/{provider}` | —      | Provider webhook endpoint                      |

### Health

| Method | Endpoint       | Description                                  |
| ------ | -------------- | -------------------------------------------- |
| `GET`  | `/healthcheck` | Server health with PostgreSQL & Redis status |

---

## 🛡 Middleware Pipeline

Requests pass through the following middleware in order:

```
Request → Recovery → Logger → Rate Limiter → CORS → [Auth] → [RBAC] → [Turnstile] → Handler
```

| Middleware            | Scope                         | Description                                             |
| --------------------- | ----------------------------- | ------------------------------------------------------- |
| **Recovery**          | Global                        | Catches panics and returns 500 with structured error    |
| **Request Logger**    | Global                        | Logs method, path, status, duration for every request   |
| **Rate Limiter**      | Global (100 req/s, burst 200) | Per-IP token bucket with automatic bucket cleanup       |
| **CORS**              | Global                        | Configurable origin, credentials, dev localhost support |
| **Auth**              | Protected routes              | JWT Bearer token validation, extracts user into context |
| **RBAC**              | Restricted routes             | `RequireRole()` checks user role against allowed roles  |
| **Turnstile**         | Auth endpoints                | Cloudflare bot protection token verification            |
| **Upload Rate Limit** | Upload routes                 | Stricter limit (1 req/s, burst 3) for media uploads     |

---

## 📨 Event-Driven Architecture

The system uses Redpanda (Kafka-compatible) for asynchronous communication between domains.

### Published Events

| Event Topic            | Publisher       | Payload        |
| ---------------------- | --------------- | -------------- |
| `order.created`        | Order Service   | `{ order_id }` |
| `order.status_updated` | Order Service   | `{ order_id }` |
| `menu.created`         | Menu Service    | `{ menu_id }`  |
| `menu.updated`         | Menu Service    | `{ menu_id }`  |
| `menu.deleted`         | Menu Service    | `{ menu_id }`  |
| `payment_initiated`    | Payment Service | Payment object |
| `payment_successful`   | Payment Service | Payment object |
| `payment_failed`       | Payment Service | Payment object |

### Subscribed Events

| Consumer         | Subscribes To        | Action                           |
| ---------------- | -------------------- | -------------------------------- |
| Order Subscriber | `payment_successful` | Sets order status to `confirmed` |
| Order Subscriber | `payment_failed`     | Sets order status to `cancelled` |

### Infrastructure

```bash
# Start Redpanda and Console
docker compose up -d

# View events in browser
open http://localhost:8081
```

---

## 💳 Payment Processing

### Flow

```
1. Client → POST /payments/initiate → Server creates Payment record
2. Server → Provider.InitializePayment() → Returns authorization_url
3. Client → Redirects to provider's hosted payment page
4. User completes payment on provider page
5. Provider → POST /payments/webhook/{provider} → Server validates & updates status
6. Provider → Redirects to callback_url → Client calls GET /payments/verify
7. Server → Provider.VerifyPayment() → Confirms status & creates Settlement
8. Server → Publishes payment_successful → Order Subscriber confirms order
```

### Settlement Calculation

On successful payment:

- **Platform Fee**: 10% commission on order total
- **Restaurant Share**: 90% (order total minus platform fee)
- Settlement records track payouts per restaurant

---

## 🔒 Concurrency & Data Integrity

### Pessimistic Locking

Critical operations use PostgreSQL `SELECT ... FOR UPDATE` to prevent race conditions:

- **Order creation**: Menu items are locked during stock deduction to prevent overselling
- **Order status updates**: Order row is locked to prevent conflicting status transitions
- **Menu updates**: Menu row is locked to prevent concurrent mutation conflicts

### Transaction Boundaries

All multi-step operations (order + items + stock deduction) execute within a single database transaction, ensuring atomicity.

---

## 📊 Logging & Observability

- **Structured JSON logging** via Uber's `zap` with per-request fields
- **Log levels**: DEBUG, INFO, WARN, ERROR, FATAL
- **Request logging middleware**: Method, path, status code, response time
- **Error tracking**: Centralized `AppError` type with titles, messages, and HTTP status codes

---

## 🔨 Development

### Hot Reload

```bash
go install github.com/air-verse/air@latest
air
```

### Create a New Migration

```bash
./migrate.sh create add_service_charge_to_orders
```

### Run Tests

```bash
go test ./...
```

### Generate Swagger Docs

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init
```

---

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 📄 License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
