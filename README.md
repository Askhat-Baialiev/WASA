# WASAText

![Go](https://img.shields.io/badge/Go-1.21-00ADD8?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-003B57?logo=sqlite&logoColor=white)
![OpenAPI](https://img.shields.io/badge/API-OpenAPI%203.0-6BA539?logo=openapiinitiative&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-backend%20%2B%20frontend-2496ED?logo=docker&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-black)

A WhatsApp-like messaging web app: direct and group chats, text and image
messages, replies, forwards, emoji reactions and read receipts. Built as the
Web and Software Architecture (WASA) project at Sapienza University of Rome.

The API is designed spec-first — the full contract lives in
[`doc/api.yaml`](doc/api.yaml) (OpenAPI 3.0) and the Go handlers implement it.

## Features

- **Login by name** — enter a username; a new one is registered on the spot. The
  returned identifier is the Bearer token for every authenticated request.
- **Profile** — change your display name (unique across users) and your photo.
- **Find people** — search users by a substring of their name.
- **Conversations** — one-to-one chats and groups, each showing its latest message.
- **Messages** — send text or an image, reply to a specific message, forward a
  message to another chat, and delete your own.
- **Delivery state** — every message tracks `sent → delivered → read`.
- **Reactions** — react to a message with an emoji and remove it again.
- **Groups** — create a group, add members, rename it, set a group photo, and leave.

## Tech stack

| Layer | Choice |
| --- | --- |
| Backend | Go 1.21, `net/http` with `julienschmidt/httprouter` |
| Storage | SQLite (`mattn/go-sqlite3`) |
| API | OpenAPI 3.0 contract in `doc/api.yaml`, Bearer-token auth |
| Frontend | Vue 3, Vite, Axios, Bootstrap |
| Packaging | Separate Docker images for backend and frontend |

## Architecture

```
webui/  (Vue 3)  ──HTTP/JSON──▶  cmd/webapi  ──▶  service/api  ──▶  service/database  ──▶  SQLite
                                 (server)         (handlers)        (data access)
```

- `cmd/webapi` — server entry point: config, logging, graceful shutdown.
- `service/api` — one handler per endpoint, request validation, auth middleware.
- `service/database` — users, conversations and messages, each in its own file.
- `doc/api.yaml` — the source of truth for the HTTP contract.

## Run in development

Backend:

```shell
go run ./cmd/webapi/
# API on http://localhost:3000
```

Frontend (in a second terminal):

```shell
./open-node.sh        # starts a throwaway node:20 container
yarn run dev          # Vue dev server; it calls the API at :3000
```

## Run with Docker

```shell
docker build -t wasatext-backend  -f Dockerfile.backend  .
docker build -t wasatext-frontend -f Dockerfile.frontend .

docker run -p 3000:3000 wasatext-backend
docker run -p 8080:80   wasatext-frontend
```

## Build for production

```shell
./open-node.sh
yarn run build-embed          # bundles the Vue app
exit
go build -tags webui ./cmd/webapi/   # single binary with the UI embedded
```

## License

[MIT](LICENSE).
