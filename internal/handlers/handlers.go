// Package handlers provides HTTP handlers and route wiring.
//
// Handlers should be thin: validate input, call services/store, and write
// responses. Keep business logic out of handlers to make testing easier.
package handlers

// Typical responsibilities:
// - Register endpoints (POST /register, POST /login, GET /me)
// - Marshal/unmarshal JSON payloads
// - Return appropriate HTTP status codes
