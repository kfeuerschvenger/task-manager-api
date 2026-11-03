<picture>
  <source media="(prefers-color-scheme: dark)" srcset="assets/brand/header-strip-dark.svg">
  <img src="assets/brand/header-strip-light.svg" alt="Task Manager API" width="100%">
</picture>

# API

Nine endpoints, one JSON error shape and one rule that matters: the creator owns a task, the assignee only sees it.

This page is the reference. The interactive version is the Swagger UI at http://localhost:8080/documentation/, and the raw spec is served at `/documentation/doc.json`.

## Quick path

1. Create an account with `POST /auth/register`, or sign in with `POST /auth/login`. Both return a token.
2. Send it as `Authorization: Bearer <token>` on every `/tasks` request.
3. In the Swagger UI, click Authorize and paste the full value `Bearer <token>`: the security definition is an API key in the `Authorization` header, so the prefix is part of what you paste.

```bash
curl -s -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"maya.rivera@example.com","password":"<password>"}'
```

## Conventions

- Base URL `http://localhost:8080`, with the port taken from `APP_PORT`. Every body is JSON except `/ping`, which answers plain text.
- Authentication is `Authorization: Bearer <token>`. A token lives 72 hours. There is no refresh.
- Every failure has the same body, a `dto.ErrorResponse`:

```json
{"code": 404, "message": "Task not found"}
```

- The body of any request is capped at 1 MiB. A larger one answers 413 `Request body too large`.
- Malformed JSON answers 400, with `Invalid JSON`, `Invalid input` or `Invalid request body` depending on the endpoint. Unknown JSON fields are ignored.
- Ids are UUID strings. A malformed task id answers 400 `Invalid task ID`.
- Timestamps are RFC 3339, for example `2025-06-01T15:04:05Z`.
- `status` is one of `pending`, `in_progress`, `complete`. `priority` is one of `low`, `medium`, `high`.

## Endpoints

| Method | Path | Auth | Purpose |
|---|---|---|---|
| GET | `/ping` | None | Health check, plain text `Pong` |
| POST | `/auth/register` | None | Create a user and return a token |
| POST | `/auth/login` | None | Return a token |
| GET | `/documentation/` | None | Swagger UI, redirects to `/documentation/index.html` |
| GET | `/tasks` | Bearer | List the tasks where the caller is creator or assignee |
| POST | `/tasks` | Bearer | Create a task |
| GET | `/tasks/{id}` | Bearer | Get one task, as creator or assignee |
| PUT | `/tasks/{id}` | Bearer | Partial update, creator only |
| DELETE | `/tasks/{id}` | Bearer | Delete, creator only |

`/documentation` without the trailing slash answers 404, because the route is the prefix `/documentation/`.

## Authentication

### POST /auth/register

| Field | Type | Rule |
|---|---|---|
| `first_name` | string | At least 2 characters after trimming |
| `last_name` | string | At least 2 characters after trimming |
| `email` | string | A valid address, stored trimmed and lowercased |
| `password` | string | At least 8 characters and at most 72 bytes, see [Password policy](#password-policy) |

| Status | When |
|---|---|
| 201 | `{"token": "<jwt>"}` |
| 400 | `Invalid input` for malformed JSON, or a validator message: `invalid email format`, `first name must be at least 2 characters`, `last name must be at least 2 characters`, `password must be at least 8 characters`, `password must be at most 72 bytes` |
| 409 | `Email already registered` |
| 413 | `Request body too large` |
| 500 | `Registration failed` |

### POST /auth/login

Request fields: `email` and `password`, both required.

| Status | When |
|---|---|
| 200 | `{"token": "<jwt>"}` |
| 400 | `Invalid input` for malformed JSON, or a validator message: `invalid email format` or `password cannot be empty` |
| 401 | `Invalid credentials`, identical for an unknown email and a wrong password |
| 413 | `Request body too large` |
| 500 | `Login failed` |

### Password policy

- At least 8 characters, counted as Unicode characters.
- At most 72 bytes, which is the input limit of bcrypt. Multibyte characters use more than one byte, so a long emoji password can be over the limit while under 72 characters.
- Passwords are stored as bcrypt hashes.
- Login only requires a non-empty password.

## Tasks

Every `/tasks` route requires a Bearer token. A missing or invalid token answers 401 with `Missing or invalid token` or `Invalid token`.

### GET /tasks

Optional query parameters, both exact match: `status` and `priority`. The response is an array of [`TaskResponse`](#taskresponse), ordered by `due_date` ascending, for the tasks whose creator or assignee is the caller.

| Status | When |
|---|---|
| 200 | An array of `TaskResponse` |
| 401 | Missing or invalid token |
| 500 | `Failed to retrieve tasks` |

> [!NOTE]
> When nothing matches, the body is `null`, not `[]`. An unknown `status` or `priority` value is not rejected: it returns the empty result.

### POST /tasks

| Field | Type | Rule |
|---|---|---|
| `title` | string | Required, not empty |
| `description` | string | Required, not empty |
| `due_date` | RFC 3339 string | Required |
| `priority` | string | Optional, default `medium` |
| `status` | string | Optional, default `pending` |
| `assignee_id` | UUID | Optional, default is the caller. It must be an existing user |

| Status | When |
|---|---|
| 201 | The created `TaskResponse` |
| 400 | `Invalid JSON` (including an unparsable `due_date`), `Missing required fields`, `Invalid priority`, `Invalid status`, `Invalid assignee ID`, `Assignee does not exist` |
| 401 | Missing or invalid token |
| 413 | `Request body too large` |
| 500 | `Failed to create task` |

### GET /tasks/{id}

| Status | When |
|---|---|
| 200 | The `TaskResponse` |
| 400 | `Invalid task ID` |
| 401 | Missing or invalid token |
| 404 | `Task not found`, also for a task that exists but belongs to other users |
| 500 | `Failed to retrieve task` |

### PUT /tasks/{id}

A partial update: every field is optional, and an empty or absent field leaves the value unchanged. The accepted fields are `status`, `priority`, `due_date` (RFC 3339), `description` and `assignee_id`. The title cannot be changed.

| Status | When |
|---|---|
| 200 | The updated `TaskResponse` |
| 400 | `Invalid request body`, `Invalid task ID`, `Invalid priority`, `Invalid status`, `Invalid due_date`, `Invalid assignee ID`, `Assignee does not exist` |
| 401 | Missing or invalid token |
| 403 | `Unauthorized to update task`, the caller is not the creator |
| 404 | `Task not found` |
| 413 | `Request body too large` |
| 500 | `Task update failed` |

### DELETE /tasks/{id}

| Status | When |
|---|---|
| 204 | Deleted, with an empty body |
| 400 | `Invalid task ID` |
| 401 | Missing or invalid token |
| 403 | `Unauthorized to delete task`, the caller is not the creator |
| 404 | `Task not found` |
| 500 | `Error deleting task` |

### TaskResponse

| Field | Type | Example |
|---|---|---|
| `id` | UUID string | `550e8400-e29b-41d4-a716-446655440000` |
| `title` | string | `Complete project documentation` |
| `description` | string | |
| `due_date` | RFC 3339 string | `2025-06-01T15:04:05Z` |
| `status` | string | `pending` |
| `priority` | string | `high` |

The response does not include the creator, the assignee or the creation and update timestamps, so a client cannot tell from it who owns a task.

## Creator versus assignee

| Action | Creator | Assignee | Another user |
|---|---|---|---|
| List (`GET /tasks`) | Sees it | Sees it | Does not see it |
| Read (`GET /tasks/{id}`) | 200 | 200 | 404 |
| Update | 200 | 403 | 403 if the id exists, 404 if not |
| Delete | 204 | 403 | 403 if the id exists, 404 if not |
| Reassign (`assignee_id` in an update) | Allowed | 403 | 403 |

- When the creator and the assignee are the same user, which is the default, that user has every right.
- The creator can assign a task to someone else and still owns it. After the reassignment the creator keeps seeing it, because the list matches creator or assignee.
- The assignee cannot change anything, the status included.
- Status changes are free: a task can move from any status to any other, and only the creator can move it.

## Swagger

- The UI is served at `/documentation/` with the models collapsed and the operations closed.
- The spec is Swagger 2.0 with host `localhost:8080`, base path `/` and scheme `http`. The security definition `BearerAuth` is an API key in the `Authorization` header.
- The generated files are committed in `docs/`: `docs.go`, `swagger.json` and `swagger.yaml`. Do not edit them by hand. They come from the godoc annotations on the controllers, the header of `cmd/main.go` and the struct tags in `dto/`.

### Regenerating the spec

From the repository root:

```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/main.go -o docs --instanceName swagger
```

- Pin `@v1.16.6`, the version `go.mod` requires, so the generated `docs.go` matches the library the runtime registers with. CI runs the same command and fails when `docs/` differs from what is committed.
- Use the `cmd/swag` path with no `/v2`. `http-swagger/v2` is the UI package for swag v1 output: its `v2` is the version of the UI package, not of swag. The swag v2 generator lives at a different module path and its output is not what this code registers.
- After regenerating, run `go test ./routes/...`: `TestSwaggerDocJSONIsServed` and `TestSwaggerUIIsServed` check that the spec and the UI are served.

---

[Back to the README](../README.md) · Previous: [CI](CI.md) · Next: [Roadmap](ROADMAP.md)
