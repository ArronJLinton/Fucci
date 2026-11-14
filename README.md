# Fucci

A comprehensive football application with mobile app, admin dashboard, and backend services.

## Project Structure

```
fucci/
├── apps/
│   ├── mobile/                 # React Native (Expo) mobile app
│   └── admin/                  # Next.js web admin dashboard
├── services/
│   ├── api/                    # Go REST API (Gin/Fiber)
│   └── workers/                # Go async workers (debate gen, media processing)
├── packages/
│   ├── api-schema/             # OpenAPI specification
│   ├── api-client/             # Generated TypeScript client
│   └── ui/                     # Shared React Native components
├── infra/
│   ├── terraform/              # Infrastructure as Code
│   └── github/                 # GitHub Actions templates
├── tools/
│   └── scripts/                # Development scripts and hooks
└── .github/
    └── workflows/              # CI/CD pipelines
```

## Quick Start

### Prerequisites

- Node.js 18+
- Go 1.22+
- Yarn 1.22+
- Docker (optional)
- Terraform (for infrastructure)

### Development Setup

1. **Clone and install dependencies:**

   ```bash
   git clone <repository-url>
   cd fucci
   make setup
   ```

2. **Start development servers:**

   ```bash
   # Start all services
   make dev

   # Or start individual services
   make mobile    # React Native metro bundler
   make api       # Go API server
   make workers   # Go workers
   make admin     # Next.js admin dashboard
   ```

### Available Commands

Run `make help` to see all available commands:

- `make install` - Install all dependencies
- `make build` - Build all applications
- `make test` - Run all tests
- `make lint` - Run all linters
- `make clean` - Clean build artifacts
- `make infra-plan` - Plan infrastructure changes
- `make infra-apply` - Apply infrastructure changes

## Services

### Mobile App (`apps/mobile/`)

React Native application built with Expo, featuring:

- Team and match information
- Debate system
- News and media
- User profiles

### Admin Dashboard (`apps/admin/`)

Next.js web application for content management:

- Team and league management
- User administration
- Content moderation
- Analytics dashboard

### API Service (`services/api/`)

Go REST API providing:

- Team and match data
- User management
- Debate system
- Media handling
- Authentication

### Workers Service (`services/workers/`)

Go background workers for:

- Debate generation
- Media processing
- Data aggregation
- Notification delivery

## Packages

### API Schema (`packages/api-schema/`)

OpenAPI 3.0 specification defining the API contract.

### API Client (`packages/api-client/`)

Generated TypeScript client for API consumption.

### UI Package (`packages/ui/`)

Shared React Native components and styles.

## Infrastructure

### Terraform (`infra/terraform/`)

Infrastructure as Code for AWS:

- VPC and networking
- RDS PostgreSQL database
- S3 for media storage
- ECS/Lambda for services
- Secrets management

### GitHub Actions (`.github/workflows/`)

CI/CD pipelines for:

- Mobile app testing and building
- Go services testing and linting
- API schema validation
- Infrastructure deployment

## Development Workflow

1. **Code Quality**: Pre-commit hooks ensure code quality
2. **Testing**: Comprehensive test suites for all services
3. **Linting**: ESLint for TypeScript/JavaScript, golangci-lint for Go
4. **Type Safety**: TypeScript throughout the frontend
5. **API-First**: OpenAPI specification drives client generation

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linters
5. Submit a pull request

## License

Copyright (c) 2026 Arron [add your legal name or company entity]. All rights reserved.

This software and associated documentation files (the "Software") are
proprietary and confidential. No part of the Software may be copied,
modified, merged, published, distributed, sublicensed, sold, or used to
create derivative works, in whole or in part, without the prior written
permission of the copyright holder.

Viewing the source code, where made available, does not grant any license
or right to use it.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
COPYRIGHT HOLDER BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY ARISING
FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR ITS USE.
