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
