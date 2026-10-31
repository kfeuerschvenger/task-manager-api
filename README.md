# Collaborative Task Management API

This project is a RESTful API designed to manage tasks in a collaborative environment. Built with Go and PostgreSQL, it follows clean architecture principles and is fully containerized with Docker.

## Technologies Used

| Backend                                                                                                                                                                                                                                                                                                      | Database                                                                                                                                                               | DevOps                                                                                                                                                     | Tools                                                                                                                                                         |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| <img width="50" src="https://raw.githubusercontent.com/marwin1991/profile-technology-icons/refs/heads/main/icons/go.png" alt="Go" title="Go"/> <img width="50" src="https://raw.githubusercontent.com/marwin1991/profile-technology-icons/refs/heads/main/icons/swagger.png" alt="Swagger" title="Swagger"/> | <img width="50" src="https://raw.githubusercontent.com/marwin1991/profile-technology-icons/refs/heads/main/icons/postgresql.png" alt="PostgreSQL" title="PostgreSQL"/> | <img width="50" src="https://raw.githubusercontent.com/marwin1991/profile-technology-icons/refs/heads/main/icons/docker.png" alt="Docker" title="Docker"/> | <img width="50" src="https://raw.githubusercontent.com/marwin1991/profile-technology-icons/refs/heads/main/icons/postman.png" alt="Postman" title="Postman"/> |

## Features

- User registration and login with JWT authentication.
- Task management with creation, update, and filtering.
- Tasks can be created for oneself or assigned to other users.
- Protected routes requiring authentication.
- PostgreSQL database with migrations.
- Dockerized environment for easy setup.

## Project Structure

```
task-manager-api
├── cmd
│   └── main.go
├── controllers
│   ├── auth_controller.go
│   ├── healthcheck_controller.go
│   └── task_controller.go
├── database
│   ├── db.go
│   ├── migrations
│   │   ├── 000001_enable_uuid_ossp.down.sql
│   │   ├── 000001_enable_uuid_ossp.up.sql
│   │   ├── 000002_create_users_table.down.sql
│   │   ├── 000002_create_users_table.up.sql
│   │   ├── 000003_create_tasks_table.down.sql
│   │   └── 000003_create_tasks_table.up.sql
│   └── migrations.go
├── docs
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── dto
│   ├── auth.go
│   ├── error.go
│   └── task.go
├── errors
│   ├── auth.go
│   ├── errors.go
│   └── validation.go
├── middleware
│   └── auth.go
├── models
│   ├── task.go
│   └── user.go
├── routes
│   └── routes.go
├── services
│   ├── auth_service.go
│   └── task_service.go
├── tests
│   ├── auth_test.go
│   ├── task_test.go
│   └── utils_test.go
├── utils
│   ├── bcrypt.go
│   ├── email.go
│   ├── jwt.go
│   └── response.go
├── validators
│   └── auth.go
├── .env
├── .env.example
├── .gitignore
├── docker-compose-test.yml
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

## Setup Instructions

1. Copy the example environment variables file:

   ```sh
   cp .env.example .env
   ```

   Then edit `.env`: set `DB_PASSWORD` and set `JWT_SECRET` to a random value of at least 32 bytes (for example `openssl rand -base64 32`). The server refuses to start otherwise.

2. Build and run all services using Docker Compose:

   ```sh
   docker-compose up -d --build
   ```

3. The API will be available at `http://localhost:8080` (or the configured port).

4. Database migrations will run automatically on startup.

## Runing Tests

To run the unit and integration tests, use the following command:

```sh
docker compose -f docker-compose-test.yml up --abort-on-container-exit --exit-code-from test
docker compose -f docker-compose-test.yml down -v
```

The test stack is self-contained: it needs no `.env` file and publishes no ports.

## API Endpoints

### Authentication

- **Register:** `POST /auth/register`
- **Login:** `POST /auth/login`

### Task Management (requires authentication)

- **Create Task:** `POST /tasks`
- **List Tasks:** `GET /tasks?status=&priority=`
- **Get one Task:** `GET /tasks/:id`
- **Update Task:** `PUT /tasks/:id`
- **Delete Task:** `DELETE /tasks/:id`

### Documentation

- **Navigate to:** `http://localhost:8080/documentation/index.html`

### Health Check

- **Ping:** `GET /ping`

## Environment Variables

See `.env.example` for the list of environment variables. `JWT_SECRET` must be at least 32 bytes. `DB_SSLMODE` sets the PostgreSQL sslmode and defaults to `disable`; it must be one of `disable`, `require`, `verify-ca`, or `verify-full`, otherwise the server refuses to start (`allow` and `prefer` are not supported because the migration driver rejects them). `DB_SSLROOTCERT` optionally sets the path of the root certificate used by `verify-ca` and `verify-full`. `DATABASE_URL` optionally overrides the connection string used by the migrations only: it is used verbatim, so `DB_SSLMODE` and `DB_SSLROOTCERT` do not apply to it (only an `sslmode` query parameter in it is checked against the same allowlist at startup), while the application itself keeps connecting with the `DB_*` variables. Postgres is published on `127.0.0.1:5432` only.

### TLS to the database

The image ships the system CA bundle (`ca-certificates`), so `verify-ca` and `verify-full` work against servers signed by a public CA with no extra setup. For a private CA, mount its root certificate into the container and point `DB_SSLROOTCERT` at it, for example with this `docker-compose.yml` override on the `api` service:

```yaml
    volumes:
      - ./certs/root.crt:/certs/root.crt:ro
```

and in `.env`:

```
DB_SSLMODE=verify-full
DB_SSLROOTCERT=/certs/root.crt
```

The container healthcheck calls `GET /ping` on `APP_PORT` (default `8080`), and both compose services use `restart: unless-stopped`.

## Notes

- Only the task creator can update a task.
- Passwords must be at least 8 characters (counted as Unicode characters) and at most 72 bytes (a bcrypt limit; multibyte characters use more than one byte). They are securely stored using bcrypt.
- JWT tokens are required for all protected routes.
