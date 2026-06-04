# api

Simple API testing TUI built with Go.

## Run

```sh
go run .
```

## Build

```sh
go build -buildvcs=false -o api
./api
```

## Install

Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/ESHAYAT102/api-client-tui/refs/heads/main/scripts/install-linux.sh | sh
```

macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/ESHAYAT102/api-client-tui/refs/heads/main/scripts/install-macos.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/ESHAYAT102/api-client-tui/refs/heads/main/scripts/install-windows.ps1 | iex
```

## Uninstall

Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/ESHAYAT102/api-client-tui/refs/heads/main/scripts/uninstall-linux.sh | sh
```

macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/ESHAYAT102/api-client-tui/refs/heads/main/scripts/uninstall-macos.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/ESHAYAT102/api-client-tui/refs/heads/main/scripts/uninstall-windows.ps1 | iex
```

## Controls

- `tab` / `shift+tab`: move focus
- arrow keys: move method table, switch Body/Header/Bearer tabs when tab selector is focused
- `enter` or `ctrl+s`: send request
- `ctrl+c`: quit

Bare `localhost:` URLs default to `http://`; other bare URLs default to `https://`.

Previous request state is saved in `~/.cache/api/config.json`.
