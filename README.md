# IIS Logs Parser

A high-performance Go backend service for parsing and managing IIS (Internet Information Services) W3C Extended Log Format files. Features a REST API with JWT authentication, domain management, and optimized batch processing for large log files.

## Features

- Parse IIS W3C Extended Log Format files
- High-performance batch processing using PostgreSQL `COPY` command
- JWT-based authentication with email verification
- Multi-tenant domain management
- Background processing scheduler for uploaded log files
- RESTful API built with Gin

## Requirements

- Go 1.23+
- PostgreSQL 12+
- SMTP server (for email verification)

## Installation

```bash
cd iis-logs-parser
go mod download
go build -o iis-logs-parser main.go
```

## Configuration

Copy `.env.local` and configure your environment:

```bash
# Environment
GO_ENV=development              # development or production

# Server
SERVER_PORT=8090

# JWT
JWT_SECRET="ReplaceWithStrongSecret"

# Database
DB_USER=postgres
DB_PASS=password
DB_HOST=localhost
DB_PORT=5432
DB_NAME=postgres-dev

# Email (SMTP)
FROM_EMAIL="your-email@example.com"
FROM_EMAIL_PASSWORD=""
FROM_EMAIL_SMTP="smtp.gmail.com"
FROM_EMAIL_PORT=587
```

## Usage

### Development

```bash
source .env.local
./iis-logs-parser "$FROM_EMAIL_PASSWORD"
```

### Production

```bash
GO_ENV=production ./iis-logs-parser
```

The email password is read from the `FROM_EMAIL_PASSWORD` environment variable in production mode.

## API Reference

All endpoints return JSON. Protected endpoints require `Authorization: Bearer <token>` header.

### Authentication

| Method | Endpoint                             | Description             |
| ------ | ------------------------------------ | ----------------------- |
| POST   | `/api/v1/users/register`             | Register new user       |
| POST   | `/api/v1/users/login`                | Login and get JWT token |
| GET    | `/api/v1/users/verify?token=<token>` | Verify email address    |

**Register Request:**

```json
{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "first_name": "John",
  "last_name": "Doe"
}
```

Password requirements: 8+ characters, uppercase, lowercase, number, special character.

**Login Response:**

```json
{
  "message": "Logged in",
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### Domains (Protected)

| Method | Endpoint              | Description         |
| ------ | --------------------- | ------------------- |
| GET    | `/api/v1/domains/`    | List user's domains |
| POST   | `/api/v1/domains/`    | Create domain       |
| PUT    | `/api/v1/domains/:id` | Update domain       |
| DELETE | `/api/v1/domains/:id` | Delete domain       |

**Create Domain Request:**

```json
{
  "name": "example.com",
  "description": "Production server"
}
```

### Log Files (Protected)

| Method | Endpoint                  | Description                 |
| ------ | ------------------------- | --------------------------- |
| GET    | `/api/v1/logs/`           | List all user's log files   |
| GET    | `/api/v1/logs/domain/:id` | List log files for a domain |
| POST   | `/api/v1/logs/upload`     | Upload log files            |
| DELETE | `/api/v1/logs/:id`        | Delete log file             |

**Upload Request:**

```bash
curl -X POST http://localhost:8090/api/v1/logs/upload \
  -H "Authorization: Bearer <token>" \
  -F "domain=1" \
  -F "logfiles=@/path/to/logfile.log"
```

## Log File Processing

Uploaded log files are processed asynchronously by a background scheduler running every 20 seconds. The processing pipeline:

1. File status changes from `pending` to `processing`
2. Log file is parsed line-by-line (IIS W3C format)
3. Entries are batch-inserted to PostgreSQL using `COPY` command
4. Status changes to `completed` (or `failed` on error)

### Supported Log Format

IIS W3C Extended Log Format with fields:

```
date time s-ip cs-method cs-uri-stem cs-uri-query s-port cs-username c-ip cs(User-Agent) sc-status sc-substatus sc-win32-status time-taken
```

## Running Tests

```bash
# Run all tests
go test ./tests -v

# Run specific test
go test ./tests -v -run TestParseLogLine

# Run benchmarks
go test ./tests -bench=. -benchmem
```

## Performance

The processor uses `pgx` with PostgreSQL `COPY` command instead of GORM for batch inserts, achieving:

- ~60% faster processing time
- ~85% memory reduction

Benchmark processing a 1.7GB log file: ~37 seconds.

See `task2-brainstorming.md` for detailed benchmark analysis.

## Project Structure

```
.
├── main.go              # Entry point, background scheduler
├── config/              # Application constants
├── database/            # PostgreSQL connection (GORM + pgx)
├── middleware/          # JWT authentication middleware
├── models/              # Data models (User, Domain, LogFile, LogEntry)
├── parser/              # IIS log line parser
├── processor/           # Batch processing with pgx COPY
├── routes/              # API route handlers
├── utils/               # JWT, password hashing, email, validation
└── tests/               # Unit tests and benchmarks
```
