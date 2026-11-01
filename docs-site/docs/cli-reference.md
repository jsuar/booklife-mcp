---
sidebar_position: 8
---

# CLI Reference

BookLife provides CLI subcommands for setup and standalone operations.

## Commands

### `serve`

Start the MCP server (default command).

```bash
booklife serve --config booklife.kdl
booklife --config booklife.kdl  # same thing
```

| Flag | Description |
|------|-------------|
| `--config PATH` | Path to KDL configuration file |

### `libby-connect`

Connect a Libby/OverDrive account using a clone code.

```bash
booklife libby-connect <8-digit-code>
```

**Steps:**
1. Open Libby app → Settings → Copy To Another Device → Sonos Speakers
2. Note the 8-digit code displayed
3. Run the command within ~40 seconds

Identity is saved to:
- Linux: `~/.config/booklife/libby-identity.json`
- macOS: `~/Library/Application Support/booklife/libby-identity.json`

### `sync`

Sync returned Libby books to Hardcover as "read".

```bash
booklife sync --config booklife.kdl [--dry-run] [--limit N]
```

| Flag | Description |
|------|-------------|
| `--config PATH` | Path to KDL configuration file |
| `--dry-run` | Preview sync without making changes |
| `--limit N` | Limit number of entries to sync |

Output includes:
- Book matching results (ISBN and title+author matching)
- Hardcover IDs for successfully matched books
- Error details for failed matches

### `import-timeline`

Import a Libby timeline JSON export to local history.

```bash
booklife import-timeline <json-file> --config booklife.kdl
```

### `version`

Show version information.

```bash
booklife version
```
