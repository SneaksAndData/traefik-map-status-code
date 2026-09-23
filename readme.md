# Traefik Map Status Code

This project is a simple Traefik middleware that allows you to map HTTP status codes to custom responses.
It can be useful for handling specific error codes in a more user-friendly way.

## Debug logging

Logging is disabled by default. Set `debug: true` in the middleware configuration
to log the loaded status code mapping once per plugin instance and the original
and resulting status codes on each `WriteHeader` call:

```yaml
http:
  middlewares:
    map-status:
      plugin:
        mapstatus:
          from: "400-499,503"
          to: "200"
          debug: true
```

With Traefik 3.7, plugin stdout is captured at the DEBUG level, so also set
`log.level: DEBUG` in Traefik. Restart Traefik after changing plugin Go source.

## Local integration setup

With Docker Compose v2 and `just` installed, run `just up` from the repository
root (or `docker compose up -d`). Static Traefik configuration
is in `integration-tests/traefik.yaml`, and routers, middleware, and backend
configuration are in `integration-tests/dynamic.yaml`.

The root `.traefik.yml` is the plugin manifest, not the server configuration.
After changing Compose commands or mounts, run `docker compose up -d` to recreate
the affected containers; `docker compose restart` does not apply these changes.

Traefik is available at `http://localhost:8080`, and the backend is directly
accessible at `http://localhost:8081`.

Run `just` to list the available commands:

| Command | Description |
| --- | --- |
| `just up` | Start Traefik and the backend. |
| `just stop` | Remove the test containers and network. |
| `just fresh` | Stop and recreate the test environment. |
| `just restart` | Recreate containers to reload plugin code and configuration. |
| `just logs` | Follow Traefik and backend logs. |
| `just status` | Show container status. |

## Integration tests

With the environment already running, execute:

```bash
go test -tags=integration -count=1 -v ./integration-tests
```

The tests only send HTTP GET requests to `/missing-integration-test`: directly
to the backend on port 8081 (expecting 404), and through Traefik on port 8080
(expecting 200). They do not create files, invoke `just`, or manage containers.
An unavailable endpoint fails the test. Without the `integration` build tag,
these tests are excluded from normal Go test runs.