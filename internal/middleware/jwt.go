// Package middleware contains HTTP middleware utilities such as JWT
// validation and request context population.
// Package middleware contains HTTP middleware helpers.
//
// The JWT middleware's responsibility is small and focused:
//   - Read the Authorization header (Bearer <token>)
//   - Parse and validate the JWT using the Auth service
//   - If valid, attach a lightweight user identity (ID, role) to the
//     request context for downstream handlers to use
//   - If invalid, return 401 Unauthorized
//
// Keep middleware logic minimal and delegate parsing/validation to
// internal/auth.Auth so it's easy to unit test.
package middleware
