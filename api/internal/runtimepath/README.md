# `runtime/`

Everything the panel remembers lives in this folder: your accounts, your servers, your FiveM builds. If you're poking around in here, read this first.

## What's inside

- **`database.db`** — all accounts, roles, invites, and server registrations.
- **`.runfive-keys`** — the secret keys that protect your login sessions and any Cfx.re API tokens stored for users.
- **`servers/<name>/`** — one folder per server the panel manages. Your server's config and files live here.
- **`artifacts/`** — FiveM server builds that have been downloaded. Shared across every server, so the same build is only downloaded once.

## ⚠ Don't do this

- **Don't delete `.runfive-keys`.** It cannot be recreated. If it's gone, every user has to log in again, and any Cfx.re API tokens stored in the database become permanent garbage. Nobody wants that.
- **Don't delete `database.db`** unless you want to start over from scratch. You'll lose all accounts, roles, and invites, and the panel will walk you through first-time setup again on the next start.
- **Don't rename a folder under `servers/`.** The panel will think the server is gone. Rename from inside the panel if you need to.
- **Don't manually delete a build from `artifacts/`** while a server is using it. That server will fail to start. The panel prevents this through its own delete button — `rm` does not.
- **Don't hand-edit `server.toml`.** The panel rewrites it on config changes and may overwrite anything it doesn't recognize.

## Backups

The whole `runtime/` folder is your backup. Stop the panel, copy the folder somewhere safe, done. To restore, replace the folder and start the panel back up. `database.db` and `.runfive-keys` must always travel together — one without the other is useless.

## Moving it somewhere else

By default this folder sits at the project root. In production, set the `RUNFIVE_ROOT` environment variable to put it wherever you want — e.g. `RUNFIVE_ROOT=/opt/runfive`. The panel will never write anywhere else.
