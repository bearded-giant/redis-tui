# Releasing redis-tui

## Cut a release

```bash
git push origin main          # push code first
./release.sh 1.0.34           # opens $EDITOR for CHANGELOG review; --yes to skip
```

`release.sh` does:
1. Validates working tree clean + on main + tag doesn't exist
2. Builds CHANGELOG section from conventional commits since previous tag
3. Opens `$EDITOR` for review (skip with `--yes`)
4. Commits CHANGELOG.md, tags `v<version>`, pushes both

## What happens after push

```
v1.0.34 tag pushed
  → release.yml (goreleaser) builds 7 platforms + .deb/.rpm/.apk
  → publishes GitHub release w/ artifacts
  → goreleaser pushes Cask update to bearded-giant/homebrew-tap inline
  → brew upgrade --cask redis-tui works
```

No separate `update-homebrew.yml` — goreleaser handles tap commit inline (different from gitlab-monitor/mdlive/gproxy).

## Watch

```bash
gh run watch -R bearded-giant/redis-tui
gh release view v1.0.34 -R bearded-giant/redis-tui --web
```

## Verify install

```bash
brew update
brew upgrade --cask redis-tui
redis-tui --version
```

## Secrets

| Secret | Required for |
|---|---|
| `GITHUB_TOKEN` | auto, release upload |
| `HOMEBREW_TAP_TOKEN` | goreleaser push to `bearded-giant/homebrew-tap` |

If `HOMEBREW_TAP_TOKEN` missing, build still works — Cask update step fails. Set via:
```bash
gh secret set HOMEBREW_TAP_TOKEN -R bearded-giant/redis-tui --body "$HOMEBREW_TAP_TOKEN"
```

## Failure recovery

| Symptom | Fix |
|---|---|
| release.yml red | fix on main, `git tag -d v1.0.34 && git push origin :v1.0.34 && ./release.sh 1.0.34` |
| Tap not updated | re-trigger release: re-tag (above). Or hand-edit `Casks/redis-tui.rb` in tap (last resort) |

See also: tap-wide `~/dev/homebrew-tap/RELEASING.md`.
