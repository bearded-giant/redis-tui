# Changelog

Notable changes per release. Newest first. Auto-populated by `./release.sh` from
conventional-commit messages between tags; review/edit in `$EDITOR` before commit.

## [v1.0.33] - 2026-06-07

### Feat
- match count, jump-to-key, jq-path filter, copy to cli command
- copy redis-cli command for current key with Y
- show D duplicate in connections footer
- duplicate connection with D keybind
- latency-doctor dashboard
- live MONITOR stream w/ filter, pause, buffer cap
- bulk-ttl dry-run preview before apply
- blob-decoder in preview pane + toggle (ctrl+p), persisted
- export single key to file from detail view

### Fix
- ssh connection updates existing

### Test
- skip live MONITOR test under -race

### Chore
- add release.sh helper + CHANGELOG stub
- trim Makefile targets, auto-gen help
- rip out unused themes keybind
- add make dev for local docker+seed+run
- connection details modal wider

