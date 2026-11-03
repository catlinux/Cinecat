# Plugins (design)

[Versión en español](../es/complementos.md)

> **Status: proposal, not implemented.** This document sets out what a plugin could extend and within which limits. Nothing will be implemented until the author approves it.

## Goal

Let an administrator add features that should not live in the core (other metadata sources, notifications, post-scan tasks) **without loading third-party code into the server** and without opening connections the administrator has not authorised.

## What it is not

- It is not a store or a plugin repository: Cinecat downloads nothing on its own (project rule: no outgoing connections other than the configured metadata sources).
- They are not Go plugins (`plugin.Open`): those require building with the exact same version, do not work on every platform and share memory and permissions with the server.
- They are not scripts inside the web app or the apps: clients never run plugin code.

## Model: an external process with a JSON protocol

A plugin is **an executable** that the administrator copies into `/data/plugins/<name>/` along with a `plugin.json` manifest. The server starts it as a child process and talks to it over **standard input and output**, one JSON request per line and one JSON response per line (like a language server, but minimal).

```json
{
  "name": "avisos-telegram",
  "version": "1.0.0",
  "protocol": 1,
  "command": ["./avisos-telegram"],
  "hooks": ["scan.finished", "item.added"],
  "network": ["api.telegram.org:443"],
  "settings": [{ "key": "token", "label": "Token del bot", "secret": true }]
}
```

- `hooks`: events it subscribes to. The server only sends it those.
- `network`: destinations the plugin declares it needs. The administrator sees them before enabling it. On Linux the server can enforce them with the process network namespace (see «Isolation»); on other systems the declaration is informational and the interface says so.
- `settings`: settings edited in Administration → Plugins. Those marked `secret` are stored encrypted like the other secrets and are never returned to the web app.

### Lifecycle

1. The administrator copies the folder and clicks «Find plugins». The server reads the manifests; **none is enabled on its own**.
2. When enabling one, the web app shows which events it receives, which network destinations it uses and which data it will see. The SHA-256 fingerprint of the executable is stored: if the file changes, the plugin is disabled until it is approved again.
3. The server starts the process at startup (or when it is enabled), sends it `init` with its settings and restarts it with increasing back-off if it crashes. After several failures in a row it disables it and records it in the activity log.

### Events (protocol version 1)

| Event | Data | Response |
|---|---|---|
| `init` | settings, Cinecat version | `ok` or error |
| `scan.finished` | library, new, changed and removed items (id, type and title only) | none |
| `item.added` | id, type, title, year | none |
| `playback.started` / `playback.stopped` | user (name only), item, device | none |
| `metadata.lookup` | title, year, type, known external ids | proposed metadata fields |

- Events are **asynchronous and time-limited** (5 s by default, 30 s for `metadata.lookup`). A slow or crashed plugin never blocks a scan or playback.
- `metadata.lookup` only **proposes** data. It is applied with today's precedence (TMDB > OMDb) as one more source the administrator orders, and it never overwrites a manual correction.
- Disk paths, tokens, passwords and IP addresses are **never** sent to a plugin.

## Isolation

- The process runs as the same user as the server but without inheriting environment variables (only `PATH`, `HOME` pointing to its folder and `LANG`), with its own folder as the only intended write location and CPU and memory limits (`setrlimit` on Linux).
- In the official Docker container, plugins run in the same container; anyone wanting more isolation can run them in another container and use «socket» mode (protocol version 2, not part of the first delivery).
- Accepted risk, stated in the interface: **a plugin is code the administrator chooses to run on their server**. Cinecat limits what it sends and what it can bring down, but it does not turn it into trusted code.

## API

- `GET /api/v1/plugins` (admin): detected plugins with status, fingerprint, events, network and last error.
- `POST /api/v1/plugins/{name}/enable` and `/disable` (admin).
- `PUT /api/v1/plugins/{name}/settings` (admin): secrets are sent but never returned.
- In version 1, plugins do **not** add their own API routes or web pages.

## Proposed delivery

1. Server: manifest discovery, enabling with fingerprint, child process with protocol v1, `init`, `scan.finished` and `item.added` events, activity log. An example plugin in `examples/plugins/registro/` that writes events to a file.
2. Web: Administration → Plugins (list, enable or disable, settings, last error).
3. `metadata.lookup` as an orderable metadata source.
4. Playback events.

## Questions for the author before implementing

- Is there a specific case you want covered first (notifications, another metadata source, something else)? That decides the delivery order.
- Is it fine that plugins are only installed by copying the folder by hand, with no installer in the web app?
