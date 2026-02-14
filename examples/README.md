# Docker Compose Examples

## Standalone

Start a single Redis instance on port `6379`:

```sh
docker compose -f examples/standalone/docker-compose.yml up -d
```

Connect with redis-tui (defaults to `localhost:6379`):

```sh
redis-tui
```

## Cluster

Start a 6-node cluster (3 masters + 3 replicas) on ports `6380`-`6385`:

```sh
docker compose -f examples/cluster/docker-compose.yml up -d
```

Connect with redis-tui using any node port (`6380`-`6385`):

```sh
redis-tui -c localhost:6380
```

## Seed Data

Populate an instance with sample data covering every data type:

```sh
# Standalone (localhost:6379)
go run ./examples/seed

# Cluster
go run ./examples/seed -addr localhost:6380 -cluster

# Flush existing data before seeding
go run ./examples/seed -flush
```

## Cleanup

```sh
docker compose -f examples/standalone/docker-compose.yml down
docker compose -f examples/cluster/docker-compose.yml down
```
