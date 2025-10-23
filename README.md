# 🛡️ Sentinel

Enterprise-grade JWT authentication microservice built with Go — small,
easy to deploy, and suitable as an auth microservice for your stack.

[![Go Version](https://img.shields.io/badge/Go-1.25.3-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## 📋 Overview

Sentinel is a modern, production-ready authentication microservice designed for building secure, scalable applications. It provides JWT-based authentication with best-in-class security features including rate limiting, comprehensive input validation, and secure password hashing.

### ✨ Key Features

- 🔐 **JWT Authentication** - Secure token-based authentication using industry-standard JWT
- 🛡️ **Enterprise Security** - Built-in rate limiting, security headers, and CORS protection
- 📊 **Dual Storage Options** - SQLite for production, in-memory for development
- ✅ **Comprehensive Validation** - Email validation, password strength checks, input sanitization
- 🔒 **Secure Password Hashing** - bcrypt with configurable cost factors
- 📝 **Structured Logging** - Contextual logging with multiple severity levels
- 🚀 **Production Ready** - Graceful shutdown, health checks, and robust error handling
- 🧪 **Well Tested** - Extensive unit test coverage for critical components
- 🔄 **Token Refresh** - Access + refresh tokens with rotation support
- 🆔 **Request IDs** - Every request includes a request ID header (`X-Request-ID`) for tracing
- ⚡ **High Performance** - Lightweight and efficient with minimal dependencies

## 🏗️ Architecture

```
sentinel/
├── cmd/
│   └── server/          # Alternative server entry point
├── internal/
│   ├── auth/           # JWT and password helpers
│   ├── config/         # Configuration management
│   ├── handlers/       # HTTP handlers
│   ├── logger/         # Structured logger
│   ├── middleware/     # Security, CORS, rate limit, request-ID
│   ├── models/         # Domain models
│   ├── server/         # HTTP server wiring
│   ├── store/          # SQLite and in-memory stores
│   └── validation/     # Input validation
```

## 🚀 Quick Start

### Prerequisites

- Go 1.25.3 or higher
- Git

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/mayvqt/Sentinel.git
   cd Sentinel
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up environment variables**
   
   Create a `.env` file in the project root:
   ```env
   PORT=8080
   DATABASE_URL=sqlite://./sentinel.db
   JWT_SECRET=your-super-secure-secret-key-here-min-32-chars
   ```

   ⚠️ **Important**: Generate a secure JWT secret:
   ```bash
   # Linux/Mac
   openssl rand -base64 32
   
   # Windows (PowerShell)
   [Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Minimum 0 -Maximum 256 }))
   ```

4. **Run the service**

   **Option 1: Using Go directly**
   ```bash
   go run .
   ```

   **Option 2: Build and run**
   ```bash
   go build .
   ./Sentinel
   ```

The server will start on `http://localhost:8080` (or your configured port).

## 📡 API Endpoints

### Health Check

**GET** `/health`

Check if the service is running and healthy.

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-10-23T12:00:00Z"
}
```

---

### User Registration

**POST** `/api/auth/register`

Register a new user account.

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com",
    "password": "SecureP@ssw0rd123"
  }'
```

**Response (201 Created):**
```json
{
  "id": "uuid-here",
  "username": "johndoe",
  "email": "john@example.com",
  "created_at": "2025-10-23T12:00:00Z"
}
```

**Validation Rules:**
- Username: 3-30 characters, alphanumeric and underscores only
- Email: Valid email format
- Password: Minimum 8 characters, must contain uppercase, lowercase, number, and special character

---

### User Login

**POST** `/api/auth/login`

Authenticate and receive JWT tokens.

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "password": "SecureP@ssw0rd123"
  }'
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

---

### Token Refresh

**POST** `/api/auth/refresh`

Refresh an expired access token using a valid refresh token.

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

---

### Get User Profile

**GET** `/api/auth/profile`

Get the authenticated user's profile. This endpoint requires an access token.

**Request:**
```bash
curl http://localhost:8080/api/auth/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Request-ID: your-request-id-optional"
```

**Response (200 OK):**
```json
{
  "id": "1",
  "username": "johndoe",
  "email": "john@example.com",
  "created_at": "2025-10-23T12:00:00Z"
}
```

---

### Protected Endpoint Example

**GET** `/api/protected`

A demonstration of a protected endpoint requiring authentication.

**Request:**
```bash
curl http://localhost:8080/api/protected \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

**Response (200 OK):**
```json
{
  "message": "This is a protected resource",
  "user_id": "uuid-here"
}
```

## 🔒 Security Features

### Rate Limiting

Sentinel implements intelligent rate limiting to prevent abuse:

- **Authentication endpoints**: 5 requests per 2 seconds per IP
- **General endpoints**: 10 requests per second per IP
- Automatic cleanup of expired rate limit entries

### Security Headers

All responses include security headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Content-Security-Policy: default-src 'self'`

### CORS Protection

Configurable CORS middleware with support for:
- Allowed origins
- Allowed methods
- Allowed headers
- Credentials support

### Input Validation & Sanitization

- SQL injection prevention
- XSS attack prevention
- Email format validation
- Password strength requirements
- Username format validation

### Password Security

- bcrypt hashing with cost factor 12
- Secure random salt generation
- No plaintext password storage

## 🧪 Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/auth
go test ./internal/handlers
go test ./internal/validation
```

## 🔧 Configuration

Sentinel can be configured through environment variables or `.env` files:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | 8080 | No |
| `DATABASE_URL` | SQLite database path | - | No* |
| `JWT_SECRET` | Secret key for JWT signing | - | **Yes** |

*If `DATABASE_URL` is not provided, Sentinel will use an in-memory store (suitable for development only).

### Example `.env` file

```env
# Server Configuration
PORT=8080

# Database Configuration
DATABASE_URL=sqlite://./data/sentinel.db

# JWT Configuration
JWT_SECRET=YkBm/hZZYyq8VHQ4caGT8m22VJH/F02fPDKvCFuNTuo=
```

## 📊 Logging

Sentinel uses structured logging with multiple severity levels:

- **DEBUG**: Detailed diagnostic information
- **INFO**: General informational messages
- **WARN**: Warning messages for potentially harmful situations
- **ERROR**: Error events that might still allow the application to continue

Example log output:
```
[INFO] 2025-10-23T12:00:00Z Configuration loaded successfully | app=Sentinel version=0.1.0
[INFO] 2025-10-23T12:00:01Z Using SQLite store | database=./sentinel.db
[INFO] 2025-10-23T12:00:01Z Server starting | address=:8080
```

## 🐳 Docker Support (Coming Soon)

Docker support is planned for future releases.

## 🚢 Production Deployment

### Best Practices

1. **Always use HTTPS** in production
2. **Set a strong JWT secret** (minimum 32 characters)
3. **Use SQLite or external database** (not in-memory store)
4. **Configure CORS** with specific allowed origins
5. **Monitor rate limiting** and adjust as needed
6. **Enable structured logging** and centralize logs
7. **Implement health check monitoring**
8. **Use environment variables** for configuration
9. **Regular security updates** of dependencies
10. **Set up database backups**

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👤 Author

**mayvqt**

- GitHub: [@mayvqt](https://github.com/mayvqt)

## 🙏 Acknowledgments

- [golang-jwt](https://github.com/golang-jwt/jwt) for JWT implementation
- [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) for pure Go SQLite driver
- [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) for bcrypt implementation

---

<p align="center">Made with ❤️ and Go</p>
