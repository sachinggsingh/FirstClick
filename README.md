# 🎬 FirstClick

> Real-time cinema seat booking backend — built with Go, Redis, and WebSockets.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7.x-DC382D?style=flat-square&logo=redis&logoColor=white)
![Gin](https://img.shields.io/badge/Gin-Framework-00BCD4?style=flat-square)
![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)

---

## Overview

FirstClick is a lightweight cinema seat-booking backend that handles **concurrent seat holds** with Redis atomic locks, real-time seat-grid updates over WebSockets, and a clean REST API consumed by a static frontend.

```
Browser  ──REST──▶  Gin API  ──▶  BookingService  ──▶  RedisStore
   ▲                                                          │
   └────────────WebSocket (SEATS_UPDATED broadcast)──────────┘
```

---

## Features

- ⚡ **Atomic seat locking** via Redis `SETNX` — only one user wins per seat
- 🔒 **Service-level mutex** for consistent booking behavior
- ⏱️ **Auto-expiring holds** — seats release automatically via Redis TTL (20s)
- 📡 **WebSocket broadcast** — all clients get instant seat-grid updates
- 🧱 **Clean layered architecture** — service / store / model separation

---

## Project Structure

```
firstclick/
├── cmd/
│   └── firstclick/
│       └── main.go           # Server entrypoint, Gin router, routes
├── internal/
│   ├── service/
│   │   └── service.go        # BookingService — hold / confirm / release logic
│   ├── store/
│   │   └── memory_store.go  # MemoryStore + RedisStore (Redis atomic seat lock)
│   ├── model/
│   │   └── model.go         # Booking struct, SeatStatus enum
│   ├── redis/
│   │   └── redis.go         # Redis client connection helper
│   └── realtime/
│       └── seats_hub.go    # WebSocket hub for SEATS_UPDATED broadcasts
└── static/
    └── index.html            # Single-page frontend (REST + WebSocket client)
```

---

## API Reference

All endpoints are prefixed at `http://localhost:8080`.

### Movies

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/movies` | List all movies |
| `GET` | `/movies/:movieId/seats` | Get live seat statuses for a movie |

### Seat Booking Flow

```
POST /movies/:movieId/seats/:seatId/hold
PUT  /sessions/:sessionId/confirm
DELETE /sessions/:sessionId
```

#### `POST /movies/:movieId/seats/:seatId/hold`

Hold a seat for 20 seconds.

```json
// Request
{ "user_id": "abc123" }

// Response 200
{
  "session_id": "uuid",
  "movie_id":   "movie-1",
  "seat_id":    "B4",
  "expires_at": 1712000000000
}
```

#### `PUT /sessions/:sessionId/confirm`

Confirm a held seat (must be called before TTL expires).

```json
// Request
{ "user_id": "abc123" }

// Response 204 No Content
```

#### `DELETE /sessions/:sessionId`

Release a held seat early.

```json
// Request
{ "user_id": "abc123" }

// Response 204 No Content
```

### Static Files

| Path | Serves |
|------|--------|
| `GET /` | `static/index.html` |
| `GET /index.html` | `static/index.html` |
| `GET /static/*` | Assets from `static/` |

---

## WebSocket

**Endpoint:** `GET /ws/seats`

The server broadcasts a message whenever any seat state changes:

```json
{
  "type":     "SEATS_UPDATED",
  "movie_id": "movie-1",
  "ts":       1712000000000
}
```

The browser listens and re-fetches `/movies/:movieId/seats` immediately when `movie_id` matches the currently selected movie — keeping all open tabs in sync without polling.

---

## Data Model

### `SeatStatus`

| Value | Meaning |
|-------|---------|
| `available` | Free to hold |
| `hold` | Temporarily locked (TTL: 20s) |
| `confirmed` | Permanently booked |
| `booked` | Not used by current flow (kept for future states) |
| `reserved` | Not used by current flow (kept for future states) |

### `Booking`

```go
type Booking struct {
    MovieID   string
    SeatID    string
    UserID    string
    Status    SeatStatus
    ExpiresAt time.Time  // populated on hold, used by UI countdown timer
}
```

---

## Concurrency Model

Two layers protect against double-booking:

```
Request 1: Hold B4  ──▶  SETNX seat:movie-1:B4  ──▶  ✅ wins (key didn't exist)
Request 2: Hold B4  ──▶  SETNX seat:movie-1:B4  ──▶  ❌ key exists → error returned
```

**Layer 1 — Redis `SETNX`** (primary gate)
- Key: `seat:<movie_id>:<seat_id>`
- Value: JSON-encoded `Booking` with `Status = hold`
- TTL: 20 seconds — seat auto-releases if user doesn't confirm

**Layer 2 — `sync.Mutex` in `BookingService`** (consistency guard)
- Ensures the read-check-write sequence inside the service stays coherent
- Redis remains the actual source of truth

> **Note:** If you ever swap the store to an in-memory `map`, the mutex becomes critical for preventing data races. With Redis it's a belt-and-suspenders guard.

---

## Getting Started

### Prerequisites

- Go 1.21+
- Redis 7.x running locally

### Run

```bash
# Start Redis (if not already running)
redis-server

# Run the server
make run

# Open in browser
open http://localhost:8080
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | `localhost:6379` | Redis server address |

```bash
# Custom Redis address
REDIS_ADDR=redis.internal:6379 make run
```

---

## Roadmap

- [ ] **Multi-seat selection** — bulk hold and bulk confirm in a single session
- [ ] **Database persistence** — add PostgreSQL as a durable backing store alongside Redis
- [ ] **Write-aside caching** — use Redis as a read cache with DB as source of truth for confirmed bookings
- [ ] **Payment integration** — hook confirm step into a payment gateway
- [ ] **Admin dashboard** — manage movies, view occupancy, force-release holds

---

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-feature`
3. Commit your changes: `git commit -m 'feat: add bulk seat hold'`
4. Push and open a pull request

---
