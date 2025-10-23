# Usage

Development quick flow

1. Copy `.env.example` to `.env` and edit as needed.

```powershell
copy .env.example .env
```

2. Build and run the server (development):

```powershell
go build ./...
go run ./cmd/server
```

3. Health check (placeholder):

```
GET /health
200 OK
```

Authentication endpoints (planned)

- POST /register — create a new user (username, email, password).
- POST /login — authenticate and receive a JWT.
- GET /me — protected; returns current user's profile.

Example: registering a user (curl)

```bash
curl -X POST http://localhost:8080/register \
	-H "Content-Type: application/json" \
	-d '{"username":"alice","email":"alice@example.com","password":"s3cret"}'
```

Notes

- The scaffold provides handlers and interfaces; implement storage, validation, and JWT issuance before using in production.
- For database migrations, add your SQL or use a migration tool (golang-migrate, goose) in the `migrations/` folder.

Configuration notes

- `config.Load()` reads environment variables directly and does not populate defaults. Ensure you set `PORT`, `DATABASE_URL`, and `JWT_SECRET` in your environment or in a local `.env` file copied from `.env.example`.
- `.env` should never be committed; the repository includes `.env.example` as a template only.

