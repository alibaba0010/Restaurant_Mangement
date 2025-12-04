# Signup Improvements & Summary

## 5-step signup summary

1. Client POSTs signup data (name, email, password, confirmPassword) to the API. Controller validates input and rejects invalid requests (400).
2. Service checks DB for existing email, hashes the password (argon2id), generates a cryptographically-strong token, and saves a payload to Redis under `verify:<token>` with TTL = 15 minutes.
3. Service sends an HTML verification email containing a verification link (and plain-text fallback). Client receives 201 and user checks their email.
4. User clicks the verification link → GET /auth/verify?token=...; controller/service reads `verify:<token>` from Redis. If missing/expired, returns 400 (invalid/expired); otherwise proceeds.
5. Service creates the user in Postgres using the hashed password stored in the Redis payload, deletes the Redis token, and returns 200 (activation success).

## Summary & checklists (concise)

- Token storage: verification tokens are stored in Redis with TTL (15 minutes). Payload stores hashed password rather than raw password.
- Hashing: argon2id is used for password hashing; encoded string stores params + salt + hash.
- UUIDs: user IDs created with UUIDv7 (time-ordered) via `utils.GenerateUUIDv7()`.
- Email: verification email is sent with `github.com/wneessen/go-mail` and an HTML body function (`VerifyMail`).
- Validation: go-playground/validator is used; custom `password_special` validator and controller/service-level validations return friendly messages.
- Panic handling: RecoverMiddleware logs panic plus stack trace to help debug internal server errors.

## Changes made (what I actually implemented)

- `internal/controllers/auth.controller.go`
  - Added controller-level validation using `validator` + `dto.RegisterValidators` and mapping of validator errors to friendly messages (returns 400 before calling service).
- `internal/services/auth.services.go`
  - Added service-level input validation mapping to `errors.ValidationErrors`.
  - Hash password with argon2id during signup BEFORE storing the payload in Redis (so Redis does not contain raw passwords).
  - Store hashed password in Redis payload and use that hashed password to create the DB user at activation (removed double-hash).
- `internal/errors/handler.go`
  - Enhanced RecoverMiddleware to log the recovered panic value and stack trace (via `runtime/debug`) to aid debugging while still returning a generic 500 to clients.
- `internal/database/redis.go`
  - A background janitor is present to remove orphaned `verify:*` keys that lack TTL (defensive cleanup).

## Improvements added or recommended (technical)

- Security
  - Hashing with argon2id (strong parameters) for new users.
  - No raw password storage in Redis (hashed only). Consider encrypting Redis payloads with AES-GCM using a server key for extra protection.
  - Return friendly validation messages; avoid returning internal errors to clients.
- Reliability
  - Redis token TTL enforcement and janitor to remove orphan keys.
  - Improved panic logging to find and fix server panics quickly.
- Developer experience
  - Centralized validators in `dto` for reuse.
  - Clear, user-facing error envelope `errors.AppError` with messages array.
- Operational
  - Recommend Docker + docker-compose with Postgres, Redis, MailHog for dev; GitHub Actions for CI (build, lint, test).

## Areas to further improve / optimize signup flow

1. Encrypt the Redis payload: use AES-GCM with a key from env (`REDIS_PAYLOAD_KEY`) to prevent payload disclosure even if Redis is compromised.
2. Rate-limit signup attempts by IP and email (sliding window) to prevent abuse.
3. Add email send retries and a dead-letter / resend mechanism (store failed emails and retry via background worker).
4. Token invalidation & rotation on resend: ensure tokens are single-use and replace old tokens on resend requests.
5. Migrate existing bcrypt passwords: implement detection of bcrypt-hashed records and rehash to argon2 on successful login (or force reset if preferred).
6. Add unit and integration tests (use docker-compose to run DB/Redis/MailHog for tests).
7. Observability: Prometheus metrics and OpenTelemetry tracing for signup/activation flow.
8. Add a secure unique index on lower(email) in Postgres to enforce case-insensitive uniqueness and avoid race conditions (also handle unique constraint errors gracefully).
9. Improve email template: responsive inline CSS, clear CTA, bold 15-minute TTL notice, plain-text fallback, and unsubscribe/support links.

## Quick next steps (recommended priority)

1. Add AES-GCM payload encryption (medium effort, high security improvement).
2. Implement email template update (bold 15-minute notice) and plaintext fallback (quick win).
3. Add rate-limiting middleware for signup endpoint (important for abuse prevention).
4. Add tests for signup→activate flow using local dockerized dependencies.

---

File created: `docs/SIGNUP_IMPROVEMENTS.md`
