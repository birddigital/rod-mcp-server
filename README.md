# Rod MCP Server

MCP (Model Context Protocol) server for browser automation using [Rod](https://go-rod.github.io/).

Perfect for testing HTMX-R applications and automating browser interactions.

## Features

- **Browser Control**: Navigate, click, fill forms
- **HTMX-R Testing**: Inspect `data-state-*` attributes
- **Screenshots**: Capture full page or viewport
- **DOM Inspection**: Read text, attributes, evaluate JavaScript
- **Wait Conditions**: Wait for elements to appear

## Installation

```bash
go install github.com/birddigital/rod-mcp-server@latest
```

Or build from source:

```bash
git clone https://github.com/birddigital/rod-mcp-server
cd rod-mcp-server
go build -o rod-mcp main.go
```

## Configuration

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "rod": {
      "command": "/path/to/rod-mcp"
    }
  }
}
```

## Available Tools

### `rod_navigate`
Navigate to a URL.

**Arguments:**
- `url` (string, required): URL to navigate to

### `rod_click`
Click an element by CSS selector.

**Arguments:**
- `selector` (string, required): CSS selector

### `rod_screenshot`
Take a screenshot.

**Arguments:**
- `filename` (string, optional): Filename (default: timestamp)
- `fullPage` (boolean, optional): Capture full page (default: false)

Screenshots saved to: `/tmp/rod-screenshots/`

### `rod_get_attribute`
Get an HTML attribute value (perfect for HTMX-R state).

**Arguments:**
- `selector` (string, required): CSS selector
- `attribute` (string, required): Attribute name

### `rod_get_text`
Get text content of an element.

**Arguments:**
- `selector` (string, required): CSS selector

### `rod_wait_for`
Wait for an element to appear.

**Arguments:**
- `selector` (string, required): CSS selector
- `timeout` (number, optional): Timeout in seconds (default: 30)

### `rod_eval`
Execute JavaScript in the page context.

**Arguments:**
- `script` (string, required): JavaScript code

### `rod_fill`
Fill an input field.

**Arguments:**
- `selector` (string, required): CSS selector for input
- `text` (string, required): Text to fill

## Usage Examples

### Testing HTMX-R State Changes

```
Use rod_navigate to go to http://localhost:8080
Use rod_get_attribute on "[data-state-demo]" for attribute "data-state-demo"
Result: "hidden"

Use rod_click on "button[hx-state-toggle='demo']"
Use rod_get_attribute on "[data-state-demo]" for attribute "data-state-demo"
Result: "visible"
```

### Taking Screenshots

```
Use rod_navigate to go to http://localhost:8080
Use rod_screenshot with fullPage: true, filename: "landing-page.png"
Result: Screenshot saved to /tmp/rod-screenshots/landing-page.png
```

## Requirements

- Go 1.21+
- Chrome/Chromium browser
- macOS/Linux/Windows

## License

MIT
