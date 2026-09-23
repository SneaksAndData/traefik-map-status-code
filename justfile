default:
    @just --list

# Recreate the test environment from scratch.
fresh: stop up

# Start Traefik and the test backend.
up:
    docker compose up -d --wait --wait-timeout 60

# Remove the test containers and network.
stop:
    docker compose down

# Reload plugin source and apply configuration changes.
restart:
    docker compose up -d --force-recreate --wait --wait-timeout 60

# Follow Traefik and backend logs.
logs:
    docker compose logs -f

# Show the test environment status.
status:
    docker compose ps -a
