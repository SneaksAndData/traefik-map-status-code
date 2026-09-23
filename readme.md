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

Run `docker compose up -d` from the repository root. Static Traefik configuration
is in `integration-tests/traefik.yaml`, and routers, middleware, and backend
configuration are in `integration-tests/dynamic.yaml`.

The root `.traefik.yml` is the plugin manifest, not the server configuration.
After changing Compose commands or mounts, run `docker compose up -d` to recreate
the affected containers; `docker compose restart` does not apply these changes.

Traefik is available at `http://localhost:8080`, and the backend is directly
accessible at `http://localhost:8081`.