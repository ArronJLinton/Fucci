# internal/futbol

Backend-only domain package for futbol data orchestration.

Scope:

- provider abstraction for external futbol APIs
- cache key + TTL policy
- canonical futbol domain DTOs and typed errors
- reusable service methods for handlers and internal aggregators

This package intentionally excludes HTTP route wiring and response writing, which remain in `services/api/internal/api`.
