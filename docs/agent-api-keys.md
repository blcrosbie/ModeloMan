# Agent API Keys

Per-agent API keys are stored in PostgreSQL table `agent_api_keys`.

- Raw keys are never stored.
- Server validates `SHA-256` hash of the incoming token.
- Accepted headers for write RPCs:
  - `x-modeloman-token: <raw-key>`
  - `authorization: Bearer <raw-key>`

## Bootstrap

Set startup env vars to seed one key automatically:

- `BOOTSTRAP_AGENT_ID` (default `orchestrator`)
- `BOOTSTRAP_AGENT_KEY` (raw secret)

On startup, the server inserts the key if it does not already exist.

## Add Another Key (Manual SQL)

1. Generate a raw key string.
2. Compute `SHA-256` hash (hex).
3. Insert into `agent_api_keys`.

Example SQL:

```sql
INSERT INTO agent_api_keys (agent_id, key_id, key_hash, is_active, created_at)
VALUES (
  'agent-worker-1',
  'ak_agent-worker-1_1739999999000000000',
  '<sha256-hex-of-raw-key>',
  TRUE,
  NOW()
);
```

## Revoke Key

```sql
UPDATE agent_api_keys
SET is_active = FALSE, revoked_at = NOW()
WHERE key_id = 'ak_agent-worker-1_1739999999000000000';
```

## Optional Expiration

```sql
UPDATE agent_api_keys
SET expires_at = NOW() + INTERVAL '30 days'
WHERE key_id = 'ak_agent-worker-1_1739999999000000000';
```
