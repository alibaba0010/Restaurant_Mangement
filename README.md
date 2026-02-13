# Restaurant Management API with PostgreSQL

A robust REST API built with Go, featuring PostgreSQL integration, structured logging, and Swagger documentation for managing restaurant operations.

## 🚀 Features

- **PostgreSQL Integration**: Efficient database operations using `pgx` driver and `Bun` ORM
- **Structured Logging**: Implemented using `zap` logger
- **API Documentation**: Swagger/OpenAPI integration
- **Environment Configuration**: Flexible configuration management
- **Router**: Using `gorilla/mux` for HTTP routing
- **Error Handling**: Centralized error handling system
- **Middleware Support**: Authentication, Rate Limiting, and CORS
- **High-Performance Pagination**: Cursor-based pagination for scalable data retrieval.
- **Redis Integration**: Caching layer for optimized menu listings and token management.
- **Event-Driven Architecture**: Redpanda-based event streaming for reactive updates (e.g., Order Status updates on Payment).
- **Payment Processing**: Integrated Monnify, Paystack, and Flutterwave with secure webhook verification and unified API.
- **AWS S3 & CloudFront**: Advanced media handling with presigned URLs and CDNs for high-performance content delivery.

## 📋 Prerequisites

- Go 1.24 or higher
- PostgreSQL
- Redis
- Redpanda (Kafka-compatible)

## 🛠 Installation

1. Clone the repository:

```bash
git clone https://github.com/alibaba0010/postgres-api.git
cd postgres-api
```

2. Install dependencies:

```bash
go mod download
```

3. Set up environment variables:
   Create a `.env` file in the root directory with the following variables:

```env
# Server
PORT=8001
FRONTEND_URL=http://localhost:3000

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=postgres
DB_PASSWORD=password
DB_NAME=postgres

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Event Bus (Redpanda/Kafka)
REDPANDA_BROKERS=localhost:9092

# AWS Configuration
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_REGION=us-east-1
AWS_BUCKET_NAME=your_bucket_name
AWS_CLOUDFRONT_DOMAIN=your_domain.cloudfront.net

# Payment Providers
PAYSTACK_SECRET_KEY=sk_test_...
MONNIFY_API_KEY=MK_...
MONNIFY_SECRET_KEY=...
MONNIFY_CONTRACT_CODE=...
FLUTTERWAVE_SECRET_KEY=mx_...
FLUTTERWAVE_PUBLIC_KEY=...
FLUTTERWAVE_HASH=your_webhook_hash
```

## 🏃‍♂️ Running the Application

1. Start the server (with hot-reload):

```bash
air
```

Or normally:

```bash
go run main.go
```

The server will start on the configured port (default: 8080)

## 📁 Project Structure

```
├── cmd/                # Entry points and migration tools
├── internal/
│   ├── auth/           # Authentication module (Users, JWT)
│   ├── common/         # Shared utilities (Logger, Errors, Events, Middlewares)
│   ├── config/         # Configuration management
│   ├── database/       # DB Connections (Postgres, Redis)
│   ├── migration/      # SQL Migrations
│   ├── orders/         # Order Management & Subscribers
│   ├── payment/        # Payment Processors & Webhooks
│   ├── restaurants/    # Restaurant & Menu Management
│   └── utils/          # Helper functions
├── docs/               # Swagger documentation
└── main.go             # Application entry point
```

## 🔄 API Endpoints

The API documentation is available at `/swagger/index.html` when the server is running.

## 🛡 Middleware

- **Authentication**: Token-based authentication and RBAC
- **Logging**: Request/Response logging
- **Error Handling**: Centralized error handling

## 📝 Logging

The application uses Uber's `zap` logger for structured logging with the following features:

- Log levels (DEBUG, INFO, WARN, ERROR, FATAL)
- Structured logging format
- Log file rotation (configured in `logger/logger.go`)

## 🔨 Development

To run the application in development mode with hot reload:

```bash
go install github.com/cosmtrek/air@latest
air
```

## 📚 Dependencies

Major dependencies include:

- `pgx/v5`: PostgreSQL driver and connection pooling
- `gorilla/mux`: HTTP router and URL matcher
- `zap`: Fast, structured logging
- `viper`: Configuration management
- `swag`: Swagger documentation generator

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.
