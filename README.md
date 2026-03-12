# deck-pb

A CLI tool that adds progress bars to Google Slides presentations created with [k1LoW/deck](https://github.com/k1LoW/deck)  

[日本語](README.ja.md)

## Installation

```bash
go install github.com/yashikota/deck-pb@latest
```

## Usage

```bash
# Add progress bars
deck-pb apply slides.md

# Remove progress bars
deck-pb delete slides.md
```

## Configuration

Configure the progress bar appearance in `deck-pb.yml`.

```yaml
progress:
  position: "bottom"   # "top" or "bottom" (default: "bottom")
  height: 10           # pixels (default: 10)
  color: "#4285F4"   # hex color (default: "#4285F4")
  startPage: 1         # first page to show (default: 1)
  endPage: 0           # last page to show, 0 = last slide (default: 0)
```

To use a different config file, specify it with the `--config` flag.

```bash
deck-pb apply slides.md --config custom.yml
```

## Authentication

Uses the same credentials as deck (`~/.local/share/deck/credentials.json`). If deck is already set up, it works out of the box.

Environment variable authentication is also supported.

| Variable | Description |
|---|---|
| `DECK_SERVICE_ACCOUNT_KEY` | Service account key JSON |
| `DECK_ENABLE_ADC` | Use Application Default Credentials |
| `DECK_ACCESS_TOKEN` | OAuth2 access token |

If you use deck's profile feature, specify it with the `--profile` flag.

```bash
deck-pb apply slides.md --profile work
```
