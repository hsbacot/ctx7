# ctx7 🔍

A beautiful CLI tool for fetching LLM context from [context7.com](https://context7.com) with fuzzy search and interactive selection.

Built with ✨ [Charm](https://charm.sh/) for a delightful terminal experience.

## Features

- **Fuzzy Search**: Leverages context7's LLM-powered search to find libraries
- **Interactive Mode**: Beautiful selection menu for multiple matches (powered by [Huh](https://github.com/charmbracelet/huh))
- **Styled Logging**: Colorful, structured logs with [Charm Log](https://github.com/charmbracelet/log)
- **Pipeable Output**: Raw content to stdout, logs to stderr - perfect for scripting
- **Fast**: < 2 second response time for typical queries

## Installation

```bash
go install github.com/hsbacot/ctx7@latest
```

Or build from source:

```bash
git clone https://github.com/hsbacot/ctx7.git
cd ctx7
go build -o ctx7
```

## Usage

### Basic Usage

Fetch documentation for a library:

```bash
ctx7 react-router
```

The tool will search context7.com and return the first match's `llms.txt` content.

### Interactive Mode

When multiple libraries match your query, use interactive mode to choose:

```bash
ctx7 -i react
```

This displays a beautiful selection menu:

```
INFO Searching context7.com query=react
INFO Found multiple libraries count=5

? Multiple libraries found - choose one:
  > React Router - React Router is a multi-strategy router... ⭐ 54762
    React - A JavaScript library for building user interfaces ⭐ 228000
    React Native - A framework for building native apps...
```

### Verbose Mode

See detailed logs including timestamps and file locations:

```bash
ctx7 -v react-router
```

### Piping Output

Logs go to stderr, raw content to stdout - perfect for piping:

```bash
# Save to file
ctx7 react-router > docs.txt

# Pipe to other tools
ctx7 react-router | grep "API"

# Suppress logs
ctx7 react-router 2>/dev/null | head -n 20
```

## Command Line Options

```
Usage: ctx7 [OPTIONS] <library-name>

Options:
  -i, --interactive    Show selection menu for multiple matches
  -v, --verbose        Show detailed logs

Example:
  ctx7 react-router
  ctx7 -i react
```

## Examples

```bash
# Get Next.js documentation
ctx7 nextjs

# Search for Vue with interactive selection
ctx7 -i vue

# Get Express.js docs and save to file
ctx7 express > express-docs.txt

# Verbose mode for debugging
ctx7 -v typescript
```

## How It Works

1. **Search**: Queries context7.com's `/v2/libs/search` API with your query
2. **Select**: Automatically picks first match, or shows interactive menu with `-i` flag
3. **Fetch**: Downloads the library's `llms.txt` file
4. **Output**: Prints raw markdown content to stdout

## Project Structure

```
ctx7/
├── main.go              # CLI entry point
├── client/
│   └── context7.go      # context7.com API client
└── ui/
    ├── logger.go        # Charm Log setup
    └── selector.go      # Huh interactive selection
```

## Development

### Requirements

- Go 1.21+
- Terminal with color support

### Build

```bash
go build -o ctx7
```

### Test

```bash
# Basic test
./ctx7 react-router

# Interactive mode
./ctx7 -i react

# Verbose mode
./ctx7 -v nextjs
```

## Dependencies

- [Charm Huh](https://github.com/charmbracelet/huh) - Interactive forms
- [Charm Log](https://github.com/charmbracelet/log) - Beautiful logging
- [Charm Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling

## Contributing

Contributions welcome! Feel free to open issues or PRs.

## License

MIT

## Credits

- Built with [Charm](https://charm.sh/) tools
- Documentation from [context7.com](https://context7.com)
- Inspired by the need for better LLM context fetching

---

Made with 💜 and Go
