# Mini-Redis

A very simple Redis server built for the November 2026 No-AI Challenge. My Go
is *VERY* rusty.

Only PING, SET, GET, and DEL are implemented.

## Build

Make sure you have Golang installed.

```bash
make
```

The provided Makefile will build `mini-redis` in the project's root.

## Run

```bash
./mini-redis
```
## Connecting

Use `redis-cli` to connect to the server.

```bash
redis-cli
```

