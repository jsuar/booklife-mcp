---
sidebar_position: 19
---

# history_import_timeline

Import Libby reading history from a timeline export URL.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | Yes | Libby timeline export URL. Only `https://share.libbyapp.com/data/...` links are accepted |

## Examples

```json
{"url": "https://share.libbyapp.com/data/{uuid}/libbytimeline-all-loans.json"}
```

To import a copy from anywhere else, save it and run `booklife import-timeline <file>`.

## How to Get the URL

1. Open Libby app
2. Go to **Settings** → **Reading History**
3. Tap **Export Timeline**
4. Copy the JSON link

## Notes

- One-time import — subsequent syncs are incremental
- Imports all historical checkouts with dates, formats, and library info
- After import, use `sync` to sync to Hardcover
