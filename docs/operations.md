# Operations

## Run The Wiki

```bash
docker compose up -d --build world-harness
```

The service listens on host port `8097` and expects nginx to strip `/world/` before proxying to it.
The checked-in nginx fragment is [deployments/nginx-world-harness.conf](../deployments/nginx-world-harness.conf).

Health check:

```bash
curl -fsS http://127.0.0.1:8097/health
```

After nginx config changes:

```bash
docker exec nginx-proxy nginx -t
docker exec nginx-proxy nginx -s reload
curl -k -fsS https://urrrm.com/world/ | grep "World Packs"
```

## Telegram Authoring

Telegram is disabled unless a bot token is provided.

```bash
cp .env.example .env
# Fill TELEGRAM_BOT_TOKEN and TELEGRAM_ALLOWED_CHAT_ID.
docker compose --profile telegram up -d --build world-harness-telegram
```

Supported commands:

- `/packs`
- `/status [pack]`
- `/search [pack] <query>`
- `/ideas [pack]`
- `/draft [pack] <type> <id> | <title> | <body>`
- `/codex [pack] <request>`

Plain text messages that do not start with `/` are stored under `packs/<pack-id>/ideas/inbox/` and do not run Codex. Use `/codex` only when a message should spend model tokens and create/update drafts.

`/codex` runs `codex exec` inside the container with `CODEX_HOME=/home/node/.codex`. The compose file mounts the host Codex home by default through `${CODEX_HOME:-${HOME}/.codex}`.

## Pack Layout

Migrated packs live under:

```text
packs/<pack-id>/
├── content/
├── drafts/
├── ideas/
├── raw/
├── resources/
├── runs/
├── archive/
└── harness.yaml
```

For the first migration, `/Users/hoonzi/Documents/repo/world-lore/contents` was converted into `packs/lumen-federation/content`. Previous `drafts`, `raw`, `schema`, `prompts`, and `tools` are preserved under `raw/` and `resources/` so they do not become active pending drafts.
