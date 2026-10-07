# Repository Planning Graph (RPG) PRD: Role-Based Access Control (RBAC) & Initial Admin Setup

**Feature Slug**: `auth-rbac-and-setup`  
**Status**: `Ready for Review`  
**Target Release**: `v1.3.0`  
**Tracking Issue**: [Map #88](https://github.com/ishafiul/cubit/issues/88)  

---

<overview>
## Problem Statement
The Cubit platform currently lacks an authentication and authorization layer. In its current state:
1. **Unprotected API & Control Plane**: Any client with network access to `cubitd` (`:8000`) can invoke `/api/v1/*` endpoints to deploy, modify, or delete worker applications, query or drop D1 SQLite databases, manipulate KV stores, or alter cluster nodes.
2. **Missing First-Run Bootstrapping**: There is no onboarding flow to detect an uninitialized cluster or provision root administrative credentials securely.
3. **Absence of Access Control**: There is no distinction between system operators, application developers, and read-only stakeholders or auditors.
4. **No Machine Credential Strategy**: CI/CD pipelines, GitHub Actions, and Cloudflare Wrangler integrations lack scoped, revokable API tokens.

## Target Users & Personas
- **Cluster Administrator**: Infrastructure and platform owner. Manages bare-metal server nodes, configures cluster settings, invites users, and defines/assigns RBAC roles.
- **Application Developer / Operator**: Builds and maintains edge workers. Deploys worker bundles, inspects live logs, configures bindings (KV, D1, R2, Queues, Cron, Workflows), but cannot alter cluster nodes or manage platform users.
- **Read-Only Viewer / Auditor**: Observes deployment status, metrics, and logs without permission to mutate applications or trigger deployments.
- **CI/CD & Automation (Machine Client)**: Automated deployment pipelines executing `wrangler deploy` or API calls using scoped, revokable API tokens.

## Success Metrics
- **Zero Open Routes**: 100% of mutating `/api/v1/*` routes require authenticated caller identity and verified RBAC permissions (except `/health` and initial `/api/v1/auth/*` entrypoints).
- **Setup Security & Single-Use Lock**: Initial setup (`POST /api/v1/auth/setup`) succeeds exactly once on an uninitialized cluster and is permanently locked (`409 Conflict`) thereafter.
- **Fine-Grained RBAC Parity**: Requests lacking required permissions return `403 Forbidden` with structured error details.
- **Automation Support**: CI/CD pipelines and external tools can authenticate via scoped `cbt_...` bearer tokens.
- **Test Integrity & No Regressions**: 100% test pass rate across all Go test suites (`go test -race ./...`) and frontend Vitest suites.

## Source Requirements & Decisions
- **Requirements Map**: [#88](https://github.com/ishafiul/cubit/issues/88)
- **Resolved Grilling Decisions**:
  - [Closed #89](https://github.com/ishafiul/cubit/issues/89): **Authentication Token & Credential Storage**: Stateless short-lived JWT access tokens (15–60 min) in `Authorization: Bearer` header, SQLite-backed rotating refresh tokens for session revocation, and bcrypt (cost factor 12) password hashing.
  - [Closed #90](https://github.com/ishafiul/cubit/issues/90): **RBAC Role & Permission Model**: Dynamic custom roles with discrete permission strings (`apps:read`, `apps:write`, `apps:deploy`, `services:kv:*`, `nodes:manage`, etc.), plus pre-seeded system roles: `Admin` (`*`), `Developer` (`apps:*`, `services:*`, `deployments:*`, `domains:*`, `nodes:read`), and `Viewer` (`*:read`).
  - [Closed #91](https://github.com/ishafiul/cubit/issues/91): **Initial Setup & Admin Bootstrapping**: Hybrid Web Setup Wizard (`/setup`) auto-presented when 0 users exist in the database, with permanent lock once initialized; supports headless auto-provisioning via `CUBIT_ADMIN_EMAIL` and `CUBIT_ADMIN_PASSWORD` environment variables on startup.
  - [Closed #92](https://github.com/ishafiul/cubit/issues/92): **API Tokens for Automation**: Scoped personal access tokens prefixed with `cbt_`, stored as SHA-256 hashes in SQLite, individually revokable, validated through unified bearer authentication middleware.
</overview>

---

<functional-decomposition>
## Capability Tree (The WHAT)

```text
Authentication, RBAC & Initial Setup
├── Capability 1: Authentication Core & Credential Management
│   ├── Feature 1.1: Bcrypt Password Hashing & Verification
│   ├── Feature 1.2: JWT Access Token Generation & Validation
│   ├── Feature 1.3: Refresh Token Lifecycle & Session Revocation
│   └── Feature 1.4: Login, Logout & Token Refresh Endpoints
├── Capability 2: Initial Setup & Admin Bootstrapping
│   ├── Feature 2.1: Cluster Initialization Status Probe
│   ├── Feature 2.2: Interactive Web Admin Registration & Single-Use Lock
│   └── Feature 2.3: Headless Environment Variable Auto-Provisioning
├── Capability 3: Role-Based Access Control (RBAC) & Custom Roles
│   ├── Feature 3.1: Permissions Catalog & Wildcard Matcher
│   ├── Feature 3.2: Built-in System Roles Seeding (Admin, Developer, Viewer)
│   ├── Feature 3.3: Role & Permission CRUD Management API
│   └── Feature 3.4: Gin RBAC Authorization Middleware Enforcement
├── Capability 4: Scoped Personal API Tokens for Automation
│   ├── Feature 4.1: High-Entropy API Token Generation (cbt_ prefix)
│   ├── Feature 4.2: SHA-256 Hash Storage & Secret Masking
│   ├── Feature 4.3: Token Scoping & Revocation Management
│   └── Feature 4.4: Unified Bearer Authentication Handling
└── Capability 5: Frontend Web Authentication & Operator Experience
    ├── Feature 5.1: Initial Setup Onboarding Wizard (/setup)
    ├── Feature 5.2: User Login & Session Persistence (/login)
    ├── Feature 5.3: Protected Route Guards & Automatic Unauthenticated Redirect
    ├── Feature 5.4: Axios / Fetch Request Interceptor (Bearer Token & Refresh)
    └── Feature 5.5: User, Role & API Token Management Settings Views
```

### Capability 1: Authentication Core & Credential Management
Provides core cryptography, credential storage, and session token issuance.

#### Feature 1.1: Bcrypt Password Hashing & Verification
- **Inputs**: Plaintext password string.
- **Outputs**: Salted bcrypt hash string (cost 12), or boolean validation result.
- **Behavior**: Securely hashes passwords before storing in SQLite; validates incoming login credentials with constant-time comparison.

#### Feature 1.2: JWT Access Token Generation & Validation
- **Inputs**: User ID, email, role ID, permissions array, cluster signing secret.
- **Outputs**: Signed HMAC-SHA256 JWT string with 15–60 minute expiration.
- **Behavior**: Generates compact bearer token containing user identity and permission claims; parses and validates signature and expiration on incoming requests.

#### Feature 1.3: Refresh Token Lifecycle & Session Revocation
- **Inputs**: User ID, client metadata.
- **Outputs**: Cryptographically secure 256-bit random refresh token stored in SQLite `refresh_tokens`.
- **Behavior**: Rotates refresh tokens on use; revokes tokens on explicit logout or administrative session termination.

#### Feature 1.4: Login, Logout & Token Refresh Endpoints
- **Endpoints**:
  - `POST /api/v1/auth/login`: Accepts email and password, returns access token + refresh token + user profile.
  - `POST /api/v1/auth/refresh`: Accepts refresh token, returns fresh access token and rotated refresh token.
  - `POST /api/v1/auth/logout`: Revokes the provided refresh token in SQLite.

---

### Capability 2: Initial Setup & Admin Bootstrapping
Guarantees a secure, streamlined bootstrapping experience for both interactive and automated deployments.

#### Feature 2.1: Cluster Initialization Status Probe
- **Endpoint**: `GET /api/v1/auth/status`.
- **Outputs**: `{"initialized": false, "version": "1.3.0"}` if 0 users exist; `{"initialized": true, "version": "1.3.0"}` once an admin exists.
- **Behavior**: Publicly accessible without credentials; used by frontend router to detect first-run setup state.

#### Feature 2.2: Interactive Web Admin Registration & Single-Use Lock
- **Endpoint**: `POST /api/v1/auth/setup`.
- **Inputs**: `{ name, email, password }`.
- **Behavior**: If database has 0 users, creates root `Admin` account, assigns built-in `Admin` role (`*`), issues active session, and marks cluster initialized. If any user already exists, returns `409 Conflict` immediately.

#### Feature 2.3: Headless Environment Variable Auto-Provisioning
- **Inputs**: `CUBIT_ADMIN_EMAIL` and `CUBIT_ADMIN_PASSWORD` (or `--admin-email` and `--admin-password`).
- **Behavior**: On daemon startup, if user count is 0 and environment variables are present, automatically provisions the root admin account in SQLite without manual intervention.

---

### Capability 3: Role-Based Access Control (RBAC) & Custom Roles
Enforces fine-grained permission checks across all API routes and enables custom role creation.

#### Feature 3.1: Permissions Catalog & Wildcard Matcher
- **Vocabulary**:
  - `*`: Superuser wildcard (all operations).
  - `apps:*`, `apps:read`, `apps:create`, `apps:update`, `apps:delete`, `apps:deploy`.
  - `services:*`, `services:kv:*`, `services:d1:*`, `services:r2:*`, `services:cron:*`, `services:queues:*`, `services:workflows:*`, `services:do:*`.
  - `nodes:*`, `nodes:read`, `nodes:write`.
  - `domains:*`, `domains:read`, `domains:write`.
  - `users:*`, `users:read`, `users:manage`.
  - `roles:*`, `roles:read`, `roles:manage`.
  - `tokens:*`, `tokens:read`, `tokens:manage`.
- **Behavior**: Wildcard evaluation engine (`apps:*` satisfies `apps:deploy`; `*` satisfies any permission).

#### Feature 3.2: Built-in System Roles Seeding
- **Roles**:
  - `admin` (System role, permissions: `["*"]`).
  - `developer` (System role, permissions: `["apps:*", "services:*", "deployments:*", "domains:*", "nodes:read"]`).
  - `viewer` (System role, permissions: `["*:read", "apps:read", "services:read", "nodes:read", "domains:read"]`).
- **Behavior**: Seeded automatically during database migration if absent; system roles cannot be deleted.

#### Feature 3.3: Role & Permission CRUD Management API
- **Endpoints**:
  - `GET /api/v1/roles`: List all system and custom roles (requires `roles:read`).
  - `POST /api/v1/roles`: Create custom role with specific permissions (requires `roles:manage`).
  - `PUT /api/v1/roles/:id`: Update role permissions or description (requires `roles:manage`).
  - `DELETE /api/v1/roles/:id`: Delete custom role (system roles rejected with 400).
  - `GET /api/v1/permissions`: Returns standard list of all assignable system permissions.

#### Feature 3.4: Gin RBAC Authorization Middleware Enforcement
- **Middleware Signature**: `middleware.RequirePermission(perm string)` and `middleware.RequireAny(perms ...string)`.
- **Behavior**: Inspects caller context attached by auth middleware; checks user permissions against required permission; returns `403 Forbidden` (`{"error": "forbidden", "required_permission": "..."}`) if missing.

---

### Capability 4: Scoped Personal API Tokens for Automation
Allows non-interactive tooling to perform actions securely.

#### Feature 4.1: High-Entropy API Token Generation
- **Inputs**: Token name, role ID (or permission list), expiration duration.
- **Outputs**: Plaintext token string (e.g. `cbt_live_4f9a8b...`) returned only once to caller.

#### Feature 4.2: SHA-256 Hash Storage & Metadata
- **Storage**: Hash of token stored in SQLite `api_tokens` table. Plaintext is never saved.
- **Metadata**: Tracks `name`, `token_hash`, `user_id`, `role_id`, `expires_at`, `last_used_at`, `created_at`.

#### Feature 4.3: Token Management API
- **Endpoints**:
  - `GET /api/v1/tokens`: List caller's tokens (or all tokens if admin).
  - `POST /api/v1/tokens`: Generate new token.
  - `DELETE /api/v1/tokens/:id`: Immediately revoke token.

#### Feature 4.4: Unified Bearer Authentication Handling
- **Behavior**: Gin auth middleware inspects `Authorization: Bearer <token>`:
  - If token begins with `cbt_`: Look up SHA-256 hash in `api_tokens`, verify not expired, update `last_used_at`, load associated role permissions into Gin context.
  - Otherwise: Parse and validate as JWT access token.

---

### Capability 5: Frontend Web Authentication & Operator Experience
Delivers a complete, accessible user experience for authentication and administration.

#### Feature 5.1: Initial Setup Onboarding Wizard (`/setup`)
- Detects `initialized: false` and redirects to `/setup`.
- Clean card UI prompting initial admin name, email, and strong password.
- Auto-logs in upon successful setup and navigates to the main applications dashboard.

#### Feature 5.2: User Login & Session Persistence (`/login`)
- Clean login form with email, password, error feedback, and remember session toggle.
- Stores JWT access token in memory/state and refresh token in `localStorage`.

#### Feature 5.3: Protected Route Guards
- React router redirects unauthenticated users to `/login`.
- Redirects uninitialized instances to `/setup`.
- Role-gated buttons (e.g., disable "Deploy Worker" or "Delete App" for `viewer` role).

#### Feature 5.4: Axios / Custom API Interceptor
- Injects `Authorization: Bearer <token>` automatically on all API calls.
- Intercepts `401 Unauthorized` responses to attempt a silent token refresh via `POST /api/v1/auth/refresh`; redirects to `/login` if refresh fails.

#### Feature 5.5: User, Role & API Token Management Settings
- Admin navigation tab in dashboard:
  - **Users View**: List users, invite/create new users, assign roles, toggle active status.
  - **Roles View**: View permission matrix, create/edit custom roles.
  - **API Tokens View**: Generate automation tokens, display token with one-click copy, revoke active tokens.
</functional-decomposition>

---

<structural-decomposition>
## Repository Structure & Modules (The HOW)

```text
cubit/
├── internal/
│   ├── domain/
│   │   └── auth.go                       # User, Role, Permission, APIToken, RefreshToken entities
│   ├── infrastructure/
│   │   └── db/
│   │       └── schema.sql                # Added: users, roles, permissions, api_tokens, refresh_tokens
│   ├── core/
│   │   └── middleware/
│   │       ├── auth.go                   # Authenticate() middleware (JWT + cbt_ Bearer token)
│   │       └── rbac.go                   # RequirePermission(), RequireAny() middleware
│   └── modules/
│       └── auth/                         # New Auth & RBAC Module
│           ├── repository.go             # SQLite queries for users, roles, tokens
│           ├── service.go                # Business logic, bcrypt, JWT, setup logic
│           ├── handler.go                # Gin HTTP handlers
│           ├── routes.go                 # Group router mounting
│           ├── service_test.go           # Unit & logic tests
│           └── handler_test.go           # HTTP endpoint tests
├── cmd/
│   └── cubitd/
│       └── main.go                       # Wire auth module & mount middlewares to API groups
└── web/
    └── src/
        ├── shared/
        │   └── stores/
        │       └── useAuthStore.ts       # Auth state, login/logout, tokens, current user & role
        ├── api/
        │   └── custom-instance.ts        # Request interceptor injecting Bearer token & refresh handling
        ├── components/
        │   ├── auth/
        │   │   ├── SetupPage.tsx         # First-run setup onboarding wizard
        │   │   ├── LoginPage.tsx         # Standard user login screen
        │   │   ├── UsersView.tsx         # User administration panel
        │   │   ├── RolesView.tsx         # Role & permission editor panel
        │   │   └── ApiTokensView.tsx     # Personal access tokens management panel
        │   └── ProtectedRoute.tsx        # React router route guard
        └── router.tsx                    # Route definitions for /setup, /login, and protected routes
```

### Module Definitions

#### Module: `internal/domain/auth.go`
- **Responsibility**: Pure domain models and validation rules for authentication and access control.
- **Exports**:
  - `type User struct`: `ID, Email, Name, PasswordHash, RoleID, IsActive, CreatedAt, UpdatedAt`.
  - `type Role struct`: `ID, Name, Description, IsSystem, Permissions []string, CreatedAt, UpdatedAt`.
  - `type APIToken struct`: `ID, UserID, Name, TokenHash, RoleID, ExpiresAt, LastUsedAt, CreatedAt`.
  - `type RefreshToken struct`: `ID, UserID, TokenHash, ExpiresAt, RevokedAt, CreatedAt`.
  - `HasPermission(userPerms []string, required string) bool`.

#### Module: `internal/modules/auth`
- **Responsibility**: End-to-end user management, authentication flows, and token lifecycle.
- **Exports**:
  - `type AuthService interface`: `Login, Refresh, Logout, Setup, GetStatus, CreateUser, UpdateUser, ListUsers, DeleteUser, CreateRole, ListRoles, CreateAPIToken, ListAPITokens, RevokeAPIToken, ValidateToken`.
  - `NewService(repo Repository, jwtSecret string) *AuthService`.
  - `NewHandler(service AuthService) *AuthHandler`.
  - `RegisterRoutes(rg *gin.RouterGroup, handler *AuthHandler, authMiddleware gin.HandlerFunc, rbacMiddleware func(string) gin.HandlerFunc)`.

#### Module: `internal/core/middleware/auth.go` & `rbac.go`
- **Responsibility**: Intercept incoming requests, extract caller identity from JWT or API token, and enforce permission checks.
- **Exports**:
  - `Authenticate(authService auth.AuthService) gin.HandlerFunc`: Injects `*domain.User` and permissions into `gin.Context`.
  - `RequirePermission(permission string) gin.HandlerFunc`: Aborts with `403` if caller lacks permission.
  - `RequireAny(permissions ...string) gin.HandlerFunc`: Aborts with `403` if caller lacks at least one permission.
</structural-decomposition>

---

<dependency-graph>
## Explicit Topological Dependency Chain

```text
Phase 0: Foundation
[domain/auth.go] ──► [db/schema.sql (Migrations)] ──► [Crypto: bcrypt & JWT helpers]
                                    │
                                    ▼
Phase 1: Core Domain
                     [modules/auth/repository.go]
                                    │
                                    ▼
                     [modules/auth/service.go]
                                    │
                                    ▼
Phase 2: Middleware & Handlers
   [middleware/auth.go & rbac.go] ──► [modules/auth/handler.go & routes.go] ──► [cmd/cubitd/main.go]
                                                                                       │
                                                                                       ▼
Phase 3: Frontend Presentation                                         [web/src/shared/stores/useAuthStore]
                                                                                       │
                                                                                       ▼
                                                           [web/src/components/auth/* & router.tsx]
```

### Layer Rules
1. **Foundation (Phase 0)**: Pure Go contracts, SQLite tables, cryptographic hashers. Zero external service dependencies.
2. **Core Domain (Phase 1)**: Database repository queries and business service orchestrations.
3. **Middleware & Handlers (Phase 2)**: Gin authentication/RBAC middlewares and API routes mounted into `cubitd`.
4. **Frontend Presentation (Phase 3)**: React SPA login, setup wizard, admin settings, and token management.
</dependency-graph>

---

<implementation-roadmap>
## Phased Development Roadmap

### Phase 0: Foundation Primitives & Database Schema
- **Goal**: Define domain structs, SQLite tables, and cryptographic utilities.
- **Tasks**:
  - [ ] Add auth tables to `internal/infrastructure/db/schema.sql` (`users`, `roles`, `api_tokens`, `refresh_tokens`).
  - [ ] Implement `internal/domain/auth.go` with domain entities, validation, and permission matcher.
  - [ ] Add bcrypt and HMAC-SHA256 JWT helper utilities.
- **Exit Criteria**: `go test -v ./internal/domain/...` passes with 100% coverage of permission logic.

### Phase 1: Core Auth & RBAC Domain Engine
- **Goal**: Implement repository and domain service for authentication, setup, roles, and tokens.
- **Tasks**:
  - [ ] Implement `internal/modules/auth/repository.go` with SQLite queries for all auth tables.
  - [ ] Implement `internal/modules/auth/service.go` (Setup, Login, Refresh, Token creation, Role/User CRUD, Seed default roles).
  - [ ] Add unit tests in `service_test.go` covering setup locking, bcrypt verification, and token rotation.
- **Exit Criteria**: Service tests demonstrate clean user lifecycle, permission resolution, and one-time setup enforcement.

### Phase 2: Gin Middlewares, HTTP Handlers & Wireup
- **Goal**: Expose auth endpoints and protect existing `/api/v1/*` routes with RBAC.
- **Tasks**:
  - [ ] Implement `internal/core/middleware/auth.go` (JWT & `cbt_` token verification).
  - [ ] Implement `internal/core/middleware/rbac.go` (`RequirePermission`).
  - [ ] Implement `internal/modules/auth/handler.go` and `routes.go`.
  - [ ] Wire auth module, headless bootstrap env check, and RBAC middlewares into `cmd/cubitd/main.go`.
  - [ ] Protect existing modules (`application`, `deployment`, `domain`, `node`, `runtime`, `service`, `github`).
- **Exit Criteria**: Integration tests verify that unauthenticated requests receive 401, unauthorized requests receive 403, and setup locks after first call.

### Phase 3: Frontend Setup, Login & User Administration
- **Goal**: Deliver the Web UI onboarding wizard, login page, session interceptor, and admin management screens.
- **Tasks**:
  - [ ] Implement `web/src/shared/stores/useAuthStore.ts` and update `custom-instance.ts` with Bearer header and refresh interceptor.
  - [ ] Implement `SetupPage.tsx` (first-run onboarding) and `LoginPage.tsx`.
  - [ ] Add route guards in `router.tsx` checking initialization status and authentication.
  - [ ] Implement `UsersView.tsx`, `RolesView.tsx`, and `ApiTokensView.tsx` in the dashboard settings.
- **Exit Criteria**: Vitest tests pass; end-to-end setup and login flow is verified in the browser.
</implementation-roadmap>

---

<test-strategy>
## Test Strategy & Critical Scenarios

### Test Pyramid
```text
        /\
       /E2E\        ← 10% (End-to-end setup, login, deployment via API token)
      /------\
     /Integration\  ← 30% (Gin RBAC middleware, SQLite repository, token rotation)
    /------------\
   /  Unit Tests  \ ← 60% (Password hashing, JWT parsing, permission matching, store reducers)
  /----------------\
```

### Critical Test Scenarios
1. **Initial Setup Lockout**:
   - `GET /api/v1/auth/status` returns `initialized: false`.
   - `POST /api/v1/auth/setup` with valid credentials creates root admin and returns `201 Created` + tokens.
   - Immediate second `POST /api/v1/auth/setup` returns `409 Conflict`.
   - Subsequent `GET /api/v1/auth/status` returns `initialized: true`.
2. **Authentication Verification**:
   - Calling protected endpoint without token returns `401 Unauthorized`.
   - Calling protected endpoint with invalid or expired token returns `401 Unauthorized`.
   - Calling protected endpoint with valid JWT returns `200 OK`.
3. **RBAC Permission Enforcement**:
   - User with `Viewer` role attempting `POST /api/v1/applications/deploy` returns `403 Forbidden`.
   - User with `Developer` role attempting `POST /api/v1/applications/deploy` returns `200 OK`.
   - User with `Developer` role attempting `POST /api/v1/nodes` returns `403 Forbidden` (`nodes:write` missing).
   - User with `Admin` role can invoke any endpoint.
4. **API Token Authentication**:
   - API token starting with `cbt_` with `apps:deploy` permission can successfully deploy worker.
   - Revoking the API token causes subsequent requests with that token to return `401 Unauthorized`.
5. **Session Rotation & Revocation**:
   - Using refresh token returns new access token and new rotated refresh token.
   - Reusing an old/consumed refresh token fails with `401 Unauthorized`.
</test-strategy>

---

<architecture>
## System Architecture & ADRs

### Database Schema Additions
```sql
CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_system INTEGER NOT NULL DEFAULT 0,
    permissions TEXT NOT NULL, -- JSON array of strings, e.g. ["apps:*", "nodes:read"]
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role_id TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY(role_id) REFERENCES roles(id)
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token_hash TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS api_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    token_hash TEXT UNIQUE NOT NULL,
    role_id TEXT NOT NULL,
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(role_id) REFERENCES roles(id)
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_api_tokens_hash ON api_tokens(token_hash);
```

### Architectural Decisions Records (ADRs)
- **ADR-01: Authentication Protocol**: Short-lived stateless HMAC-SHA256 JWTs for API efficiency, coupled with SQLite-stored refresh tokens for instant revocation and session rotation.
- **ADR-02: RBAC Model**: Granular string permissions (`domain:action`) with hierarchical wildcard support (`*` and `domain:*`), allowing built-in system roles as well as user-created dynamic roles.
- **ADR-03: Initial Bootstrapping**: Hybrid approach featuring a web setup wizard (`/setup`) that locks after first use, plus optional headless environment variable auto-provisioning (`CUBIT_ADMIN_EMAIL`, `CUBIT_ADMIN_PASSWORD`) for zero-touch cloud deployments.
- **ADR-04: Machine Credentials**: Scoped personal access tokens prefixed with `cbt_`, stored as SHA-256 hashes, validated in unified Gin Bearer authentication middleware.
</architecture>

---

<risks>
## Risk Analysis & Mitigations
- **Existing Deployments Migration**:
  - *Risk*: Existing unauthenticated Cubit clusters upgrading to v1.3.0 will suddenly lock all endpoints if no admin exists.
  - *Mitigation*: The daemon checks initialization state on startup. If 0 users exist, it leaves setup open and redirects to `/setup`, ensuring the existing operator can immediately claim root admin rights without data loss.
- **JWT Secret Security**:
  - *Risk*: Missing secret configuration defaults to an insecure hardcoded key.
  - *Mitigation*: If `CUBIT_JWT_SECRET` is unset, `cubitd` automatically generates a secure random 32-byte secret and persists it in SQLite cluster settings or logs a prominent warning.
- **Performance Impact of Middleware**:
  - *Risk*: JWT parsing and permission checking adding latency to high-frequency worker deployments or telemetry endpoints.
  - *Mitigation*: JWT verification is stateless HMAC-SHA256 in memory (<0.1ms). API token hashes are indexed in SQLite with prepared statements.
</risks>
