# Docker Compose Examples

## Standalone

Start a single Redis instance on port 6379:

```sh
docker compose -f examples/standalone/docker-compose.yml up -d
```

Connect with redis-tui:

```sh
redis-tui
```

## Cluster

Start a 6-node cluster (3 masters + 3 replicas) on ports 6380-6385:

```sh
docker compose -f examples/cluster/docker-compose.yml up -d
```

Connect with redis-tui:

```sh
redis-tui -c localhost:6380
```

## Cleanup

```sh
docker compose -f examples/standalone/docker-compose.yml down
docker compose -f examples/cluster/docker-compose.yml down
```
