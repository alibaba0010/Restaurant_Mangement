# Restaurant Management API with PostgreSQL

A robust REST API built with Go, featuring PostgreSQL integration, structured logging, and Swagger documentation for managing restaurant operations.

## 🚀 Features

- **PostgreSQL Integration**: Efficient database operations using `pgx` driver
- **Structured Logging**: Implemented using `zap` logger
- **API Documentation**: Swagger/OpenAPI integration
- **Environment Configuration**: Using `viper` for flexible configuration management
- **Router**: Using `gorilla/mux` for HTTP routing
- **Error Handling**: Centralized error handling system
- **Middleware Support**: Authentication and logging middleware
- **High-Performance Pagination**: Cursor-based pagination for scalable data retrieval.
- **Redis Integration**: Caching layer for optimized menu listings and token management.
- **Event-Driven Architecture**: Event streaming support for real-time updates (e.g., MenuUpdated events).
- **AWS S3 & Multipart Uploads**: Advanced media handling with presigned URLs and multipart support for large video files.
- **CloudFront Content Delivery**: Optimized serving of menu images and videos.

## 📋 Prerequisites

- Go 1.24 or higher
- PostgreSQL
- Redis (optional)

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
DB_HOST=your_host
DB_PORT=your_port
DB_USERNAME=your_username
DB_PASSWORD=your_password
DB_NAME=your_database_name
PORT=your_app_port

# AWS Configuration
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_REGION=your_region
AWS_BUCKET_NAME=your_bucket_name
AWS_CLOUDFRONT_DOMAIN=your_cloudfront_domain.cloudfront.net
```

## 🏃‍♂️ Running the Application

1. Start the server:

```bash
go run main.go
```

The server will start on the configured port (default: 8080)

## 📁 Project Structure

```
├── api/
│   ├── config/         # Configuration management
│   ├── controllers/    # Request handlers
│   ├── database/       # Database connections (PostgreSQL,
│   ├── errors/         # Error handling and types
│   ├── middlewares/    # HTTP middlewares
│   ├── models/         # Data models
│   └── routes/         # API routes
├── docs/              # Swagger documentation
├── logger/            # Logging configuration
└── main.go           # Application entry point
```

## 🔄 API Endpoints

The API documentation is available at `/swagger/index.html` when the server is running.

## 🛡 Middleware

- **Authentication**: Token-based authentication
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
