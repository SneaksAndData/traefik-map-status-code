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


# Follow Traefik and backend logs.
logs:
    docker compose logs
