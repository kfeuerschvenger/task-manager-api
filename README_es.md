<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/assets/brand/banner-dark.svg">
    <img src="docs/assets/brand/banner-light.svg" alt="Task Manager API: tareas de entrada, estado de salida. El ciclo de estados va de pending a in_progress y a complete." width="100%">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/kfeuerschvenger/task-manager-api/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/kfeuerschvenger/task-manager-api/ci.yml?branch=main&style=flat-square&label=CI&logo=githubactions&logoColor=white" alt="Estado de CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/licencia-MIT-00ADD8?style=flat-square" alt="Licencia MIT"></a>
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.25">
  <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat-square&logo=postgresql&logoColor=white" alt="PostgreSQL 16">
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker">
</p>

<p align="center"><a href="README.md">English</a> · <strong>Español</strong></p>

<p align="center">
  <img src="docs/assets/screens/hero.gif" alt="La interfaz Swagger de la API: inicio de sesión, autorización con el token, creación de una tarea y listado de las tareas de prioridad alta" width="100%">
</p>

**Task Manager API es una API REST en Go para tareas que tienen un creador, un asignado y, con suerte, una fecha de vencimiento.**

Regístrate, inicia sesión y crea tareas que avanzan por `pending`, `in_progress` y `complete`. Si le asignas una tarea a un compañero, la puede ver, pero las llaves las conservas tú: solo el creador edita o elimina.

Es un servicio Go pequeño y por capas sobre PostgreSQL, con autenticación JWT, una interfaz Swagger generada desde el código y una configuración Docker que arranca con dos comandos. La configuración es estricta a propósito: un secreto débil detiene el servidor antes de que prometa lo que no puede cumplir.

## Un recorrido

<table>
  <tr>
    <td width="50%" valign="top">
      <img src="docs/assets/screens/overview.png" alt="La interfaz Swagger con los grupos auth, healthcheck y tasks abiertos">
      <p><strong>Todos los endpoints, una sola página.</strong> Swagger se genera desde el código, así que lo que lees es lo que hace el servidor.</p>
    </td>
    <td width="50%" valign="top">
      <img src="docs/assets/screens/create-task.png" alt="POST /tasks ejecutado desde Swagger y respondiendo 201 con la tarea nueva">
      <p><strong>Tareas de entrada, 201 de salida.</strong> Con título, descripción y fecha de vencimiento alcanza. La prioridad y el estado toman valores por defecto razonables.</p>
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <img src="docs/assets/screens/filter-high.png" alt="GET /tasks con el filtro de prioridad en high, respondiendo 200 con las tareas de prioridad alta">
      <p><strong>Filtra lo urgente.</strong> Pide un estado o una prioridad y recibe las tareas ordenadas por vencimiento, las más próximas primero.</p>
    </td>
    <td width="50%" valign="top">
      <img src="docs/assets/screens/forbidden.png" alt="PUT /tasks/{id} del asignado respondiendo 403 con el mensaje Unauthorized to update task">
      <p><strong>Mirar no es editar.</strong> Un asignado ve la tarea, pero modificarla le corresponde al creador, y la API lo dice con un 403.</p>
    </td>
  </tr>
</table>

## Endpoints de un vistazo

| Método | Ruta | Autenticación | Propósito |
|---|---|---|---|
| GET | `/ping` | Ninguna | Chequeo de salud |
| POST | `/auth/register` | Ninguna | Crea un usuario y devuelve un token |
| POST | `/auth/login` | Ninguna | Inicia sesión y devuelve un token |
| GET | `/documentation/` | Ninguna | Interfaz Swagger |
| GET | `/tasks` | Bearer | Lista tus tareas, con filtros `status` y `priority` |
| POST | `/tasks` | Bearer | Crea una tarea |
| GET | `/tasks/{id}` | Bearer | Obtiene una tarea |
| PUT | `/tasks/{id}` | Bearer | Actualiza una tarea, solo el creador |
| DELETE | `/tasks/{id}` | Bearer | Elimina una tarea, solo el creador |

Los campos de cada petición, los códigos de estado y los cuerpos de error están en [docs/API.md](docs/API.md), que está en inglés.

## Qué la hace especial

- **El creador tiene las llaves.** Creador y asignado ven la tarea. Solo el creador puede actualizarla, reasignarla o eliminarla, y quien no es ninguno de los dos ni siquiera puede leerla: recibe un 404.
- **Asígnala y sigue siendo el dueño.** Un creador puede pasarle una tarea a un compañero y seguir siendo su dueño, y la sigue viendo en su lista.
- **Los vencimientos son datos.** Cada tarea tiene fecha de vencimiento y la lista se ordena por ella. Se aceptan fechas pasadas: la API registra tu optimismo sin juzgarlo.
- **Estricta por defecto.** `JWT_SECRET` debe tener al menos 32 bytes, los cuerpos de las peticiones están limitados a 1 MiB, las contraseñas se guardan con bcrypt y un correo desconocido recibe la misma respuesta que una contraseña incorrecta.
- **Documentación que no se desfasa.** La especificación Swagger se genera desde el código, y CI falla cuando la versión guardada está desactualizada.

Una tarea puede pasar de un estado a cualquier otro, y solo su creador puede moverla: no hay controles entre los tres.

```mermaid
%%{init: {"theme": "base", "themeVariables": {"fontFamily": "Segoe UI, Helvetica, Arial, sans-serif", "primaryColor": "#D6F3FA", "primaryBorderColor": "#00ADD8", "primaryTextColor": "#0F172A", "lineColor": "#0092B8", "textColor": "#0F172A", "edgeLabelBackground": "#ffffff"}}}%%
stateDiagram-v2
  direction LR
  [*] --> pending: se crea así, salvo que indiques otro
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

## Bajo el capó

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/diagrams/architecture-dark.svg">
  <img src="docs/assets/diagrams/architecture.svg" alt="Los clientes llegan a un único binario de Go. Una petición pasa por el límite de cuerpo, el router mux, el middleware JWT en las rutas de tareas, los controladores y los servicios, que consultan PostgreSQL mediante GORM. Swagger lo sirve el mismo binario y golang-migrate aplica el esquema al iniciar" width="100%">
</picture>

Un único binario de Go hace todo. Una petición cruza el límite de cuerpo y el router `gorilla/mux`, después el middleware JWT en las rutas `/tasks`, y llega a un controlador que la decodifica y traduce los errores tipados a códigos de estado. Los servicios contienen las reglas, incluida la de quién puede tocar qué, y hablan con PostgreSQL mediante GORM. Las migraciones corren con `golang-migrate` cada vez que arranca el servidor, y la interfaz Swagger la sirve el mismo proceso.

<p align="center">
  <img src="https://skillicons.dev/icons?i=go" alt="Go" title="Go" width="48">&nbsp;
  <img src="https://skillicons.dev/icons?i=postgres" alt="PostgreSQL" title="PostgreSQL" width="48">&nbsp;
  <img src="https://skillicons.dev/icons?i=docker" alt="Docker" title="Docker" width="48">
</p>

## Inicio rápido

Solo necesitas Docker en ejecución.

```bash
cp .env.example .env
docker compose up --build
```

Después abre http://localhost:8080/documentation/ y pruébala.

> [!TIP]
> Reemplaza `JWT_SECRET` en `.env` por al menos 32 bytes, por ejemplo la salida de `openssl rand -base64 32`. El valor de ejemplo se rechaza a propósito, así que el servidor no arranca con él. Todas las variables, cómo correrlo sin Docker y los datos de demostración están en [ENVIRONMENT.md](docs/ENVIRONMENT.md).

## Para profundizar

El README es la presentación rápida. Los documentos de fondo son los planos. Están en inglés.

| Documento | De qué trata |
|---|---|
| [Architecture](docs/ARCHITECTURE.md) | Las capas, la secuencia de arranque, cómo un error se vuelve un código de estado, el modelo de datos y el diseño del JWT |
| [Environment](docs/ENVIRONMENT.md) | Cada variable, la configuración con Docker y local, los datos de demostración y qué revisar cuando el servidor se niega a arrancar |
| [Testing](docs/TESTING.md) | Qué paquetes necesitan base de datos, dos formas de correr las pruebas y el proceso que toma estas capturas |
| [CI](docs/CI.md) | Los tres jobs, la protección de ramas y cómo fluyen las actualizaciones de Dependabot |
| [API](docs/API.md) | Cada endpoint con sus campos, códigos de estado y cuerpos de error, más el flujo de Swagger |
| [Roadmap](docs/ROADMAP.md) | Lo que sigue abierto, agrupado por lo que toca |

---

<p align="center">
  <strong>Hecho en la Patagonia · 50°20′S</strong><br>
  Hecho con cariño por <strong>Kevin Feuerschvenger</strong> · Ingeniero de Software (terminado antes del vencimiento, casi siempre)
</p>

<p align="center">
  <a href="https://github.com/kfeuerschvenger"><img src="https://img.shields.io/badge/GitHub-kfeuerschvenger-181717?style=flat-square&logo=github&logoColor=white" alt="GitHub"></a>
  <a href="https://www.linkedin.com/in/kevin-feuerschvenger/"><img src="https://img.shields.io/badge/LinkedIn-kevin--feuerschvenger-0A66C2?style=flat-square&logo=data%3Aimage%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI%2BPHBhdGggZmlsbD0id2hpdGUiIGQ9Ik0yMC40NSAyMC40NWgtMy41NnYtNS41N2MwLTEuMzMtLjAyLTMuMDQtMS44NS0zLjA0LTEuODUgMC0yLjE0IDEuNDUtMi4xNCAyLjk0djUuNjdIOS4zNVY5aDMuNDF2MS41NmguMDVjLjQ4LS45IDEuNjQtMS44NSAzLjM3LTEuODUgMy42IDAgNC4yNyAyLjM3IDQuMjcgNS40NnY2LjI4ek01LjM0IDcuNDNhMi4wNiAyLjA2IDAgMSAxIDAtNC4xMyAyLjA2IDIuMDYgMCAwIDEgMCA0LjEzek03LjEyIDIwLjQ1SDMuNTZWOWgzLjU2djExLjQ1eiIvPjwvc3ZnPg%3D%3D" alt="LinkedIn"></a>
  <a href="https://feuerschvenger.com/"><img src="https://img.shields.io/badge/Portfolio-feuerschvenger.com-00ADD8?style=flat-square&logo=googlechrome&logoColor=white" alt="Portfolio"></a>
</p>

<p align="center"><sub>Publicado bajo la <a href="LICENSE">licencia MIT</a>.</sub></p>
