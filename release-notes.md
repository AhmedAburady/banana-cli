# BANANA CLI v1.1.1

## What's New

### Replace Flag (`-r`)

New `-r` flag preserves the input filename for the output when editing images.

```bash
# Without -r: generates "generated_1_20260201_143052.png"
banana -i photo.png -p "make it cartoon"

# With -r: outputs "photo.png" (replaces original)
banana -i photo.png -p "make it cartoon" -r

# With -r and multiple images: outputs "photo_1.png", "photo_2.png", etc.
banana -i photo.png -p "make it cartoon" -r -n 3
```

**Notes:**
- Only works with single input files (not folders)
- When `-n > 1`, adds index suffix to preserve the original

---

# BANANA CLI v1.0.9

## What's New

### New `describe` Command

Analyze images and extract style descriptions using AI. Perfect for creating consistent style prompts.

```bash
# Plain text style description
banana describe -i photo.jpg

# Analyze folder of style references (unified description)
banana describe -i ./reference_images/

# Add style context to guide analysis
banana describe -i image.png -a "2D flat vector art"

# Structured JSON output
banana describe -i photo.jpg -json -o style.json
```

**Features:**
- Single image or folder analysis
- Multiple images = unified style description
- `-p` flag for custom prompts (overrides default)
- `-a` flag for additional context (prepended to default)
- `-json` flag for comprehensive structured output
- Output to file (`-o`) or stdout

### Prompt File Support

- `-p` flag now accepts a file path in addition to text
- Useful for complex JSON prompts that are hard to escape in shell
- Supports any text file: `.json`, `.md`, `.txt`, etc.

```bash
# Text prompt (as before)
banana -p "a sunset over mountains"

# Load prompt from file
banana -p prompt.json -n 3
banana -p ~/prompts/calligraphy.txt -ar 1:1
```

### Auto Version Detection

- Version now auto-detected from Go build info when installed via `go install`
- No more "dev" version when installing from module

---

# BANANA CLI v1.0.8

(Broken release - use v1.0.9 instead)

---

# BANANA CLI v1.0.7

(Broken release - use v1.0.9 instead)

---

# BANANA CLI v1.0.6

## What's New

### Performance

- HTTP connection pooling for faster concurrent requests
- Parallel loading and base64 encoding of reference images
- Request timeout handling (120s)

---

# BANANA CLI v1.0.5

## What's New

### Security

- API key input is now hidden in both TUI and CLI
- TUI shows `•••••` as you type
- CLI uses standard hidden input (like sudo/ssh)

---

# BANANA CLI v1.0.4

## What's New

### Expanded Aspect Ratios

- Added "Auto" as the default aspect ratio - lets Gemini choose the best ratio for your prompt
- Now supports 11 aspect ratios: Auto, 1:1, 16:9, 9:16, 4:3, 3:4, 2:3, 3:2, 5:4, 4:5, 21:9
- Added ultra-wide 21:9 for cinematic shots

### Improved TUI

- Horizontal scrolling for aspect ratio selector with ◀ ▶ indicators

---

# BANANA CLI v1.0.3

## What's New

### API Key Configuration System

No more exporting environment variables! BANANA CLI now saves your API key securely in your `~/.config/banana` folder.

**New config commands:**
```bash
banana config set-key YOUR_API_KEY   # Save your key
banana config show                    # View config (key is masked)
banana config path                    # Show config file location
```

**Auto-prompt for API key:**
- **CLI**: If no API key is found, you'll be prompted to enter one
- **TUI**: Shows a dedicated API key input screen on first launch

**Priority-based lookup:**
1. `GEMINI_API_KEY` environment variable
2. `GOOGLE_API_KEY` environment variable
3. Config file (`~/.config/banana/config.json`)

This lets you override the saved key with env vars when needed.

### Version Flag

```bash
banana --version
banana -v
```

### Code Quality

- Refactored TUI architecture for better separation of concerns
- API key view extracted to dedicated module

## Upgrade

```bash
go install github.com/AhmedAburady/banana-cli/cmd/banana@latest
```

Or download the binary for your platform from the releases page.

## Quick Start

```bash
# Save your API key once
banana config set-key YOUR_GEMINI_API_KEY

# Generate images
banana -p "a cyberpunk city at night" -n 3

# Or use the interactive TUI
banana
```

---

# BANANA CLI v1.0.0 - v1.0.2

AI-powered image generation and editing using Google's Gemini API.

## Features

### Dual Interface
- **Interactive TUI** - Beautiful terminal UI with gradient banner and intuitive navigation
- **CLI Mode** - Scriptable command-line interface for automation

```bash
banana -p "A beautiful sunset"
```

### Image Generation
- Generate images from text prompts
- Edit existing images with AI
- Support for reference images (single file or folder)

### Performance
- **Parallel Processing** - Generate up to 20 images simultaneously
- Optimized API calls with minimal memory footprint

### Customization
- **Aspect Ratios**: 1:1, 16:9, 9:16, 4:3, 3:4
- **Image Sizes**: 1K, 2K, 4K
- **Google Search Grounding** - Enhance prompts with real-time web context
