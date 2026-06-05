# transmission-telegram

Telegram bot to control [Transmission](https://transmissionbt.com/) and search [RuTracker](https://rutracker.org). Fork of [pyed/transmission-telegram](https://github.com/pyed/transmission-telegram) with a native Go RuTracker client.

## Features

### Transmission (unchanged)

All original text commands still work: `list`, `add`, `search`, `stop`, `stats`, and the rest. Send `/help` in Telegram for the full list.

- Magnet links and `.torrent` files are added to Transmission
- `search <query>` searches torrents already in Transmission

### RuTracker (new)

| You send | Bot does |
|---|---|
| Plain text, e.g. `metallica 2008` | Searches RuTracker, shows 10 results per page with inline buttons |
| RuTracker topic URL | Fetches magnet and adds to Transmission |
| Magnet / other URL | Adds to Transmission (as before) |
| `.torrent` file | Adds to Transmission (as before) |
| `search ubuntu` | Searches local Transmission torrents (not RuTracker) |

**Search UI**

- Buttons `1`–`10` to pick a result
- `◀ Prev` / `Next ▶` to paginate
- On selection: title, size, seeds/leeches, description preview
- `Download` adds magnet to Transmission; `Back` returns to the result list

## Build

```bash
go build -o transmission-telegram .
```

## Run

```bash
./transmission-telegram \
  -token=YOUR_BOT_TOKEN \
  -master=your_telegram_username \
  -url=http://localhost:9091/transmission/rpc \
  -username=admin \
  -password=your_password \
  -rutracker-user=your_rutracker_login \
  -rutracker-pass=your_rutracker_password
```

`-master` is your Telegram **username** without `@`. Can be repeated for multiple users.

### Flags

| Flag | Env | Description |
|---|---|---|
| `-token` | `TT_BOTT` | Telegram bot token |
| `-master` | — | Allowed Telegram username (repeatable) |
| `-url` | — | Transmission RPC URL (default `http://localhost:9091/transmission/rpc`) |
| `-username` | `TR_AUTH` (user:pass) | Transmission RPC username |
| `-password` | `TR_AUTH` | Transmission RPC password |
| `-rutracker-user` | `RT_USER` | RuTracker login |
| `-rutracker-pass` | `RT_PASS` | RuTracker password |
| `-no-transmission` | — | Skip Transmission at startup; RuTracker search still works. Transmission connects lazily on Download |
| `-no-live` | — | Disable live message editing |
| `-logfile` | — | Write logs to file |
| `-transmission-logfile` | — | Tail Transmission log for completion notifications |

### RuTracker-only test mode

To test search without Transmission running at startup:

```bash
./transmission-telegram \
  -token=YOUR_BOT_TOKEN \
  -master=your_username \
  -no-transmission \
  -rutracker-user=your_login \
  -rutracker-pass=your_password
```

Transmission commands are disabled; free-text messages trigger RuTracker search. Download still connects to Transmission if it becomes available.

### RuTracker login note

RuTracker often shows a **captcha** on programmatic login. If you see `incorrect username or password` in logs but credentials work in a browser, the site is blocking automated login — not rejecting your password.

## Docker

```bash
docker build .
```

```bash
docker run -d --name transmission-telegram \
  --network host \
  xxut/transmission-telegram:0.0.1 \
  -token=YOUR_BOT_TOKEN \
  -master=your_username \
  -url=http://localhost:9091/transmission/rpc \
  -username=admin \
  -password=your_password \
  -rutracker-user=your_login \
  -rutracker-pass=your_password
```


## Development

```bash
# unit tests (HTML parser mocks)
go test ./rutracker/...

# live RuTracker test (needs credentials)
RT_USER=login RT_PASS=pass go test -tags=integration ./rutracker/ -run TestLive
```

## Credits

- [pyed/transmission-telegram](https://github.com/pyed/transmission-telegram) — original bot
- [nikityy/rutracker-api](https://github.com/nikityy/rutracker-api) — Node API used as reference, reimplemented in Go under `rutracker/`
