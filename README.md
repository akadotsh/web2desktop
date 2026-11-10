# web2desktop

Turn any website into a desktop app.

## Install

Requires Go. Install the CLI with:

```sh
go install github.com/akadotsh/web2desktop/cmd/web2desktop@latest
```

Make sure your Go `bin` directory is on your `PATH` to run `web2desktop`.

## Quick start

```sh
web2desktop https://example.com --name Example
```

The build writes its files to `./dist/<platform>`.

For a repeatable project, create a configuration file instead:

```sh
web2desktop init example.com --name Example
web2desktop build web2desktop.json
```

`init` fills in defaults and writes `web2desktop.json`. Edit that file when you want to change the website, icon, window size, version, or platforms. Use the one-line command when you do not need a file.

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

Open the generated app on macOS:

```sh
open ./dist/darwin-$(go env GOARCH)/Example.app
```

Build for Windows and Linux:

```sh
go run ./cmd/web2desktop https://example.com \
  --name Example \
  --platform windows/amd64,linux/amd64
```

This creates a Windows executable in `./dist/windows-amd64`, and a Linux executable, installer archive, and Debian package in `./dist/linux-amd64`. Install NSIS on the build machine to also create the Windows installer.

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
--platform      comma-separated targets, for example windows/amd64,linux/amd64
--config         configuration file path when using init
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
