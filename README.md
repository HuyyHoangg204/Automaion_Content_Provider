# User Management API

A simple User Management API built with Go and Gin framework, providing JWT-based authentication and user management functionality.

## Project Structure

The project follows a standard Go project layout:

```
user-management-api/
├── cmd/                  # Application entry points
│   └── server/           # Main server application
├── internal/             # Private application code
│   ├── database/         # Database connection and repositories
│   ├── handlers/         # HTTP request handlers
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Data models
│   ├── router/           # HTTP router setup
│   ├── services/         # Business logic services
│   └── utils/            # Utility functions
├── docs/                 # Swagger documentation
├── Dockerfile            # Docker build instructions
├── docker-compose.yml    # Docker Compose configuration
├── go.mod                # Go module definition
└── go.sum                # Go module checksums
```

## Features

- RESTful API for user authentication
- JWT-based authentication with refresh tokens
- Secure password hashing
- User profile management
- Swagger API documentation
- PostgreSQL database integration with GORM
- CORS support
- Graceful server shutdown

## Requirements

- Go 1.21 or higher
- PostgreSQL 13 or higher
- Docker (optional, for containerized deployment)

## Setup and Installation

### Local Development

1. Clone the repository:
   ```
   git clone <repository-url>
   cd user-management-api
   ```

2. Copy the example environment file and modify as needed:
   ```
   cp env.example .env
   ```

3. Install dependencies:
   ```
   go mod tidy
   ```

4. Run the application:
   ```
   go run cmd/server/main.go
   ```

### Using Docker

1. Build and run using Docker Compose:
   ```bash
   docker-compose up -d
   ```

2. Check if services are running:
   ```bash
   docker-compose ps
   ```

3. View logs:
   ```bash
   docker-compose logs -f
   ```

4. Stop services:
   ```bash
   docker-compose down
   ```

This will start both the PostgreSQL database and the backend server.

**📖 For detailed Docker setup guide, see: [docs/DOCKER_SETUP.md](docs/DOCKER_SETUP.md)**

## API Documentation

The API provides the following endpoints:

### Authentication (Public)
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Refresh JWT token

### User Management (Protected)
- `GET /api/v1/auth/profile` - Get current user profile
- `POST /api/v1/auth/change-password` - Change user password
- `POST /api/v1/auth/logout` - User logout
- `GET /api/v1/users/me` - Get current user info

### Admin Management (Admin Only)
- `POST /api/v1/admin/register` - Register new user (admin only)
- `GET /api/v1/admin/users` - Get all users (admin only)
- `PUT /api/v1/admin/users/{id}/status` - Set user active status (admin only)

### Health Check
- `GET /api/v1/health` - Health check endpoint

For detailed API documentation, run the server and visit `/swagger/index.html` in your browser.

## Environment Variables

Create a `.env` file with the following variables:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=green_anti_detect_browser
DB_SSLMODE=disable

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-in-production
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h

# Server Configuration
PORT=8080
LOG_LEVEL=info

# API Configuration
BASE_PATH=/
```

## Default Admin User

The application automatically creates a default admin user on startup:
- **Username**: admin
- **Password**: Helloworld@@123

**Important**: Change the admin password immediately after first login in production!

## JWT Token Usage

1. Login with credentials to get access and refresh tokens
2. Include the access token in the Authorization header: `Bearer <access_token>`
3. Use the refresh token to get new access tokens when they expire

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

