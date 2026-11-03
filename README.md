# web2desktop

Turn any website into a desktop app.

## Quick start

```sh
go run ./cmd/web2desktop https://example.com --name Example
```

The build creates an `.app` and `.dmg` in `./dist/darwin-arm64` or `./dist/darwin-amd64`.

## Examples

Create an app from a domain:

```sh
go run ./cmd/web2desktop example.com --name Example
```

Set an icon, window size, and output directory:

```sh
go run ./cmd/web2desktop https://example.com \
  --name Example \
  --icon ./icon.png \
  --width 1200 \
  --height 800 \
  --output ./build
```

Open the generated app:

```sh
open ./dist/darwin-$(go env GOARCH)/Example.app
```

Check that the local build requirements are available:

```sh
go run ./cmd/web2desktop doctor
```

## Options

```text
--name          application name
--identifier    macOS bundle identifier
--app-version   application version (default: 0.1.0)
--width         initial window width (default: 1200)
--height        initial window height (default: 800)
--icon          square PNG icon
--output        artifact directory (default: ./dist)
--dev           launch in development mode
--verbose        show build output
--keep-workdir  retain the generated project
```

## Development

```sh
go run ./cmd/web2desktop --help
go build -o web2desktop ./cmd/web2desktop
```

## License

MIT
