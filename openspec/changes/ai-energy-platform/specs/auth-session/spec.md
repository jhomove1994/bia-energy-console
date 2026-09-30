# Auth Session Specification

## Purpose

A single demo user with a stateless signed token that protects the API. Out of scope: multiple users, roles, password reset, OAuth.

## Requirements

### Requirement: Login

`POST /api/auth/login` with JSON `{username, password}` MUST return `200` with `{token, expires_at}` when they match the demo credentials from configuration. Otherwise it MUST return `401` with no token.

#### Scenario: Valid credentials

- GIVEN the configured demo user
- WHEN the client posts matching credentials
- THEN the response is `200` with a token and expiry

#### Scenario: Invalid credentials

- GIVEN wrong password
- WHEN the client posts to login
- THEN the response is `401` and contains no token

#### Scenario: Malformed body

- GIVEN a body that is not valid JSON
- WHEN the client posts to login
- THEN the response is `400`

### Requirement: Stateless signed token

The token MUST be signed with a secret from configuration (HMAC) and carry an expiry. Verification MUST need no server-side session store.

#### Scenario: Restart keeps sessions valid

- GIVEN a token issued before an API restart with the same secret
- WHEN the client uses it after restart before expiry
- THEN the request is authorized

### Requirement: Protected API

Every `/api/*` route except login MUST require `Authorization: Bearer <token>` and return `401` for a missing, malformed, tampered or expired token.

#### Scenario: Missing token

- GIVEN no Authorization header
- WHEN the client calls `GET /api/meters`
- THEN the response is `401`

#### Scenario: Tampered token

- GIVEN a valid token with its payload altered
- WHEN the client calls `GET /api/meters`
- THEN the response is `401`

#### Scenario: Expired token

- GIVEN a token past its expiry
- WHEN the client calls `GET /api/meters`
- THEN the response is `401`

#### Scenario: Valid token

- GIVEN a valid token
- WHEN the client calls `GET /api/meters`
- THEN the response is `200`

### Requirement: Zero-config credentials

The demo credentials, token secret and token lifetime MUST be configurable by environment and MUST have documented defaults so the stack runs without configuration. Passwords and secrets MUST NOT be logged or returned.

#### Scenario: Defaults

- GIVEN no auth environment variables
- WHEN the API starts
- THEN login works with the documented default demo credentials
- AND startup logs do not print the password or secret
