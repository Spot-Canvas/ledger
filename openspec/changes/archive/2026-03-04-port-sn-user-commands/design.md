## Context

`trader` is a single-purpose ledger CLI that only talks to `trader_url`. The `sn` CLI has always been the "full platform" tool, mixing admin commands (products, tenants, engine, metrics, ml, backfill) with user-facing ones (auth, strategy, trading config, price, backtest, signals). Non-admin users need `sn` installed even though half its commands are off-limits to them.

The non-admin commands in `sn` are self-contained: they each call `api_url` or `ingestion_url` (the same two base URLs) and share a common HTTP client, config, and output layer. The porting work is therefore largely mechanical, with the main design effort in:

1. How to extend `trader`'s client and config without breaking the existing ledger commands
2. How to adapt the `trading` command interface (account as positional arg)
3. Where auth lives now that `trader` has its own `auth login`

The current `trader` config already reads `api_key` from `~/.config/sn/config.yaml` as a fallback and resolves `tenant_id` via `GET /auth/resolve`. The sn `auth login` OAuth flow writes to `~/.config/sn/config.yaml`. After this change, `trader auth login` will write to `~/.config/trader/config.yaml` instead — existing users who already ran `sn auth login` continue to work with no action required.

## Goals / Non-Goals

**Goals:**
- Port all non-admin `sn` commands into `trader` with minimal adaptation
- Single config file (`~/.config/trader/config.yaml`) covers the full platform surface
- `auth login` works standalone — no `sn` install required
- `trading` command uses account ID as a positional argument
- Signals are filtered to the authenticated user's enabled trading configs (same behaviour as `sn`)
- README.md and `skills/trader/SKILL.md` fully updated to reflect new commands

**Non-Goals:**
- Admin commands (`products`, `engine`, `metrics`, `tenants`, `ml`, `backfill`) are not ported
- No changes to the ledger REST API or server
- No changes to `sn` — it continues to exist as-is
- No unified config file between `sn` and `trader` — they remain separate files with a one-way read fallback (trader reads sn's `api_key` if its own is missing)
- Preserving backwards compatibility with existing `trader` command signatures — consistency takes priority

## Decisions

### 1. Consistency over backwards compatibility

**Decision:** Existing `trader` command signatures may change wherever they are inconsistent with the patterns established by the ported commands or the broader CLI design. Backwards compatibility is not a constraint.

**Rationale:** This change consolidates two CLIs into one. A patchwork of inconsistent conventions would undermine the goal. Concretely, the existing commands already follow a clean account-as-positional-arg pattern that the ported commands must match.

**What this means in practice:** The existing ledger commands (`trades`, `positions`, `orders`, `portfolio`, `accounts`) are well-structured already. The ported commands will follow the same patterns. If inconsistencies are discovered during implementation (e.g. flag naming, output formatting), fix them.

### 2. Two HTTP clients, not one

**Decision:** Keep the existing ledger `Client` (single `trader_url`) and introduce a separate `PlatformClient` that holds `api_url` + `ingestion_url`, mirroring `sn`'s client exactly.

**Rationale:** The ledger client has distinct behaviour (tenant-scoped paths, `auth/resolve` bootstrapping, raw SSE streaming). Merging them into one struct would add complexity without benefit. The new commands never touch `trader_url`; the existing commands never touch `api_url`. Clean separation.

**Alternative considered:** A single unified client with three base URL fields. Rejected — the ledger client's `newClient()` exits on credential failure, which is correct for ledger commands but wrong for `auth status` (which should just print "not logged in").

### 3. Config extension — additive only

**Decision:** Add `api_url`, `web_url`, `ingestion_url`, `nats_url`, `nats_creds_file` to `validConfigKeys` and `configDefaults`. Existing `trader_url`, `api_key`, `tenant_id` keys are unchanged.

Default values copied from `sn`:
- `api_url`: `https://signalngn-api-potbdcvufa-ew.a.run.app`
- `ingestion_url`: `https://signalngn-ingestion-potbdcvufa-ew.a.run.app`
- `nats_url`: `tls://connect.ngs.global`

**Env var prefix:** `TRADER_` (consistent with existing convention). `TRADER_API_URL`, `TRADER_INGESTION_URL`, etc.

**Rationale:** Most users will never need to set these — the defaults point at production. Power users (local dev, staging) can override via env var or `trader config set`.

### 4. `PlatformClient` does not call `auth/resolve`

**Decision:** `PlatformClient.newPlatformClient()` reads `api_key` using the existing `resolveAPIKey()` helper but does **not** call `auth/resolve` or require `tenant_id`. The platform API endpoints are tenant-scoped via the API key itself (server-side), not via a tenant ID in the URL path.

**Rationale:** sn's client never needed `tenant_id` — it authenticates purely with `api_key` as a Bearer token. The trader ledger client needed `tenant_id` because the ledger URL paths include it (`/api/v1/accounts/...`). The platform API does not.

**Exception:** `signals.go` needs trading configs to build the allowlist — it calls `GET /config/trading` via `PlatformClient`, which works with just `api_key`.

### 5. Auth writes to `~/.config/trader/config.yaml`

**Decision:** `trader auth login` writes `api_key` to `~/.config/trader/config.yaml`, not to `~/.config/sn/config.yaml`.

**Rationale:** `trader` owns its config file. Writing to sn's config would be surprising and creates a cross-tool dependency in the wrong direction. Users who previously logged in via `sn auth login` continue to work because `resolveAPIKey()` already falls back to reading sn's config.

**Migration:** Zero action required for existing users.

### 6. `trading` command — account as positional arg

**Decision:** All `trading` subcommands that require an account take it as the **first positional argument**, not a `--account` flag. `trading reload` is not ported (admin-only, stays in `sn`):

```
trader trading list [account]
trader trading get <account> <exchange> <product>
trader trading set <account> <exchange> <product> [flags]
trader trading delete <account> <exchange> <product>
```

`trader trading list` without an account lists all configs (for discoverability); with an account filters to that account.

**Rationale:** Consistent with every other trader command (`trades list live`, `positions live`, etc.) where the account is always the first positional argument. The `--account` flag approach in `sn` was designed for an audience that may not know their account ID upfront; trader users always know (live/paper/default).

**Difference vs sn:** In `sn`, `trading set` requires `--account`. In `trader`, the account is positional and required by cobra's `Args: cobra.MinimumNArgs(3)`.

### 7. Signals NATS credentials — embed sn's read-only creds

**Decision:** Copy the embedded read-only NGS credentials from `sn/cmd/sn/signals.go` verbatim into `cmd/trader/cmd_signals.go`. Users can override with `nats_creds_file` in config.

**Rationale:** The credentials are already public (subscribe-only, publish-denied on all subjects — safe to embed). Duplicating them avoids any runtime dependency on `sn`. If the creds rotate, both CLIs will need updating anyway.

### 8. Merge `strategies` and `strategy` into a single `strategies` command

**Decision:** Drop the separate `strategy` top-level command. Everything lives under `strategies`:

```
trader strategies list                          # all built-in + user strategies (TYPE column distinguishes)
trader strategies get <id>                      # get a user strategy
trader strategies validate --name X --file X    # validate a source file
trader strategies create --name X --file X      # create a user strategy
trader strategies update <id> --file X          # update a user strategy
trader strategies activate <id>
trader strategies deactivate <id>
trader strategies delete <id>
trader strategies backtest <id> --exchange ...  # backtest a user strategy
```

`strategies list` calls `GET /strategies` (which returns both built-in and user strategies) and renders a single table with a `TYPE` column (`builtin` / `user`). User strategies also show their `ACTIVE` status; built-in strategies show `-` in that column.

**Rationale:** Having two top-level commands (`strategy` / `strategies`) for what is conceptually one domain is confusing. The split in `sn` was incidental — `strategies list` was a convenience shortcut added later. A single `strategies` command with clear subcommands is more discoverable and consistent with the rest of the CLI (e.g. `trader accounts list`, `trader trades list`).

## Risks / Trade-offs

**[Risk] Config key collision** — `api_key` and `tenant_id` already exist in the trader config with the same semantics as sn. Adding `api_url`/`ingestion_url` could confuse users who think they need to set them.
→ Mitigation: Defaults point at production. `trader config show` will display `[default]` source, making it clear they're optional.

**[Risk] Stale allowlist in `signals`** — the signal allowlist is built once at startup from `GET /config/trading`. Long-running `trader signals` sessions won't pick up config changes.
→ Mitigation: Same behaviour as sn — acceptable for a CLI tool. Document that restarting the command refreshes the allowlist.

**[Risk] NATS credentials rotation** — if the embedded NGS credentials expire, `trader signals` breaks silently until the binary is updated.
→ Mitigation: `nats_creds_file` override allows out-of-band credential updates without a release.

## Migration Plan

1. Implement changes (no server-side changes needed)
2. Build and test locally against staging
3. Release a new `trader` binary version
4. Update README.md and `skills/trader/SKILL.md`
5. No rollback complexity — the ledger commands are untouched; new commands are additive

## Open Questions

- During specs/implementation: are there any inconsistencies in the existing ledger commands worth fixing in this same pass? (flag naming, output column naming, etc.) Flag for discussion if discovered.
