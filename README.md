<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/assets/brand/banner-dark.svg">
    <img src="docs/assets/brand/banner-light.svg" alt="Task Manager API: Tasks in, status out. The status lifecycle runs pending, in_progress, complete." width="100%">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/kfeuerschvenger/task-manager-api/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/kfeuerschvenger/task-manager-api/ci.yml?branch=main&style=flat-square&label=CI&logo=githubactions&logoColor=white" alt="CI status"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/licence-MIT-00ADD8?style=flat-square" alt="MIT licence"></a>
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.25">
  <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat-square&logo=postgresql&logoColor=white" alt="PostgreSQL 16">
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker">
</p>

<p align="center"><strong>English</strong> · <a href="README_es.md">Español</a></p>

<p align="center">
  <img src="docs/assets/screens/hero.gif" alt="The Swagger UI of the API: signing in, authorizing with the token, creating a task and listing the high priority tasks" width="100%">
</p>

**Task Manager API is a Go REST API for tasks that have an owner, an assignee and a deadline that, with luck, someone meets.**

Register, sign in, and create tasks that move through `pending`, `in_progress` and `complete`. Give a task to a teammate and they can see it, while you keep the keys: only the creator edits or deletes.

It is a small, layered Go service on PostgreSQL, with JWT authentication, a Swagger UI generated from the code and a Docker setup that starts in three steps. The configuration is strict on purpose: a weak secret stops the server before it can make promises it cannot keep.

## Take the tour

<table>
  <tr>
    <td width="50%" valign="top">
      <img src="docs/assets/screens/overview.png" alt="The Swagger UI with the auth, healthcheck and tasks groups open">
      <p><strong>Every endpoint, one page.</strong> The Swagger UI is generated from the code, so what you read is what the server does.</p>
    </td>
    <td width="50%" valign="top">
      <img src="docs/assets/screens/create-task.png" alt="POST /tasks executed from the Swagger UI and answering 201 with the new task">
      <p><strong>Tasks in, 201 out.</strong> Title, description and a due date are all it takes. Priority and status fall back to sensible defaults.</p>
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <img src="docs/assets/screens/filter-high.png" alt="GET /tasks with the priority filter set to high, answering 200 with the high priority tasks">
      <p><strong>Filter what is urgent.</strong> Ask for a status or a priority and get the tasks sorted by due date, soonest first.</p>
    </td>
    <td width="50%" valign="top">
      <img src="docs/assets/screens/forbidden.png" alt="PUT /tasks/{id} from the assignee answering 403 with the message Unauthorized to update task">
      <p><strong>Looking is not editing.</strong> An assignee sees the task, but changing it is the creator's call, and the API says so with a 403.</p>
    </td>
  </tr>
</table>

## Endpoints at a glance

| Method | Path | Auth | Purpose |
|---|---|---|---|
| GET | `/ping` | None | Health check |
| POST | `/auth/register` | None | Create a user and get a token |
| POST | `/auth/login` | None | Sign in and get a token |
| GET | `/documentation/` | None | Swagger UI |
| GET | `/tasks` | Bearer | List your tasks, with `status` and `priority` filters |
| POST | `/tasks` | Bearer | Create a task |
| GET | `/tasks/{id}` | Bearer | Get one task |
| PUT | `/tasks/{id}` | Bearer | Update a task, creator only |
| DELETE | `/tasks/{id}` | Bearer | Delete a task, creator only |

Request fields, status codes and error bodies are in [docs/API.md](docs/API.md).

## What makes it special

- **The creator holds the keys.** Creator and assignee both see a task. Only the creator can update, reassign or delete it, and a user who is neither of them cannot even read it: they get a 404.
- **Assign it, keep it.** A creator can hand a task to a teammate and still own it, and still sees it in their list.
- **Deadlines are data.** Every task has a due date and the list is sorted by it. Past dates are accepted: the API records your optimism without judging it.
- **Strict by default.** `JWT_SECRET` must be at least 32 bytes, request bodies are capped at 1 MiB, passwords are hashed with bcrypt, and an unknown email gets the same answer as a wrong password.
- **Docs that cannot drift.** The Swagger spec is generated from the code, and CI fails when the committed spec is stale.

A task can move between statuses freely, and only its creator can move it: there is no gatekeeping between the three.

```mermaid
%%{init: {"theme": "base", "themeVariables": {"fontFamily": "Segoe UI, Helvetica, Arial, sans-serif", "primaryColor": "#D6F3FA", "primaryBorderColor": "#00ADD8", "primaryTextColor": "#0F172A", "lineColor": "#0092B8", "textColor": "#0F172A", "edgeLabelBackground": "#ffffff"}}}%%
stateDiagram-v2
  direction LR
  [*] --> pending: created, unless you say otherwise
  pending --> in_progress
  in_progress --> complete
  complete --> in_progress
  in_progress --> pending
  pending --> complete
  complete --> pending
  classDef pendingStyle fill:#E2E8F0,stroke:#94A3B8,color:#0F172A
  classDef progressStyle fill:#FEF3C7,stroke:#F59E0B,color:#0F172A
  classDef completeStyle fill:#DCFCE7,stroke:#22C55E,color:#0F172A
  class pending pendingStyle
  class in_progress progressStyle
  class complete completeStyle
```

## Under the hood

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/diagrams/architecture-dark.svg">
  <img src="docs/assets/diagrams/architecture.svg" alt="Callers reach one Go binary. A request passes the body limit, the mux router, the JWT middleware on the tasks routes, the controllers and the services, which query PostgreSQL through GORM. Swagger UI is served by the same binary and golang-migrate applies the schema at startup" width="100%">
</picture>

One Go binary does it all. A request crosses the body limit and the `gorilla/mux` router, then the JWT middleware on the `/tasks` routes, and reaches a controller that decodes it and maps typed errors to status codes. The services hold the rules, including who may touch what, and talk to PostgreSQL through GORM. Migrations run with `golang-migrate` every time the server starts, and the Swagger UI is served by the same process.

<p align="center">
  <img src="https://skillicons.dev/icons?i=go" alt="Go" title="Go" width="48">&nbsp;
  <img src="https://skillicons.dev/icons?i=postgres" alt="PostgreSQL" title="PostgreSQL" width="48">&nbsp;
  <img src="https://skillicons.dev/icons?i=docker" alt="Docker" title="Docker" width="48">
</p>

## Quick start

All you need is Docker, running.

```bash
cp .env.example .env
# edit .env: set JWT_SECRET to at least 32 bytes (openssl rand -base64 32)
docker compose up --build
```

Then open http://localhost:8080/documentation/ and try it out.

> [!TIP]
> Replace `JWT_SECRET` in `.env` with at least 32 bytes, for example the output of `openssl rand -base64 32`. The placeholder is rejected on purpose, so the server will not start with it. Every variable, running without Docker and the demo seed are in [ENVIRONMENT.md](docs/ENVIRONMENT.md).

## Dive deeper

The README is the elevator pitch. The deep docs are the building plans.

| Document | The hook |
|---|---|
| [Architecture](docs/ARCHITECTURE.md) | The layers, the startup sequence, how an error becomes a status code, the data model and the JWT design |
| [Environment](docs/ENVIRONMENT.md) | Every variable, Docker and local setup, the demo seed and what to check when the server refuses to start |
| [Testing](docs/TESTING.md) | Which packages need a database, two ways to run the suite and the pipeline that takes these screenshots |
| [CI](docs/CI.md) | The three jobs, branch protection and how Dependabot updates flow |
| [API](docs/API.md) | Every endpoint with its fields, status codes and error bodies, plus the Swagger workflow |
| [Roadmap](docs/ROADMAP.md) | What is still open, grouped by what it touches |

---

<p align="center">
  <strong>Made in Patagonia · 50°20′S</strong><br>
  Built with care by <strong>Kevin Feuerschvenger</strong> · Software Engineer (built before the due date, mostly)
</p>

<p align="center">
  <a href="https://github.com/kfeuerschvenger"><img src="https://img.shields.io/badge/GitHub-kfeuerschvenger-181717?style=flat-square&logo=github&logoColor=white" alt="GitHub"></a>
  <a href="https://www.linkedin.com/in/kevin-feuerschvenger/"><img src="https://img.shields.io/badge/LinkedIn-kevin--feuerschvenger-0A66C2?style=flat-square&logo=data%3Aimage%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI%2BPHBhdGggZmlsbD0id2hpdGUiIGQ9Ik0yMC40NSAyMC40NWgtMy41NnYtNS41N2MwLTEuMzMtLjAyLTMuMDQtMS44NS0zLjA0LTEuODUgMC0yLjE0IDEuNDUtMi4xNCAyLjk0djUuNjdIOS4zNVY5aDMuNDF2MS41NmguMDVjLjQ4LS45IDEuNjQtMS44NSAzLjM3LTEuODUgMy42IDAgNC4yNyAyLjM3IDQuMjcgNS40NnY2LjI4ek01LjM0IDcuNDNhMi4wNiAyLjA2IDAgMSAxIDAtNC4xMyAyLjA2IDIuMDYgMCAwIDEgMCA0LjEzek03LjEyIDIwLjQ1SDMuNTZWOWgzLjU2djExLjQ1eiIvPjwvc3ZnPg%3D%3D" alt="LinkedIn"></a>
  <a href="https://feuerschvenger.com/"><img src="https://img.shields.io/badge/Portfolio-feuerschvenger.com-00ADD8?style=flat-square&logo=googlechrome&logoColor=white" alt="Portfolio"></a>
</p>

<p align="center"><sub>Released under the <a href="LICENSE">MIT licence</a>.</sub></p>
