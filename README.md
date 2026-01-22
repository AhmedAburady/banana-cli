# Nano Banana Pro

A Go CLI tool that generates images using Google's Gemini 3 Pro Image model based on reference images and custom prompts.

## Features

- Generate images from reference images with custom prompts
- Parallel image generation
- Interactive TUI for configuration
- Supports multiple reference images per request

## Requirements

- Go 1.21+
- Gemini API key

## Installation

```bash
go build -o nano_banana_pro .
```

## Usage

1. Set your API key:
```bash
export GEMINI_API_KEY=your_api_key
```

2. Create folder structure:
```
your_folder/
├── refs/      # Put reference images here
└── output/    # Generated images will be saved here
```

**How folders work:**
- You provide the **parent folder path** (e.g., `./input/01` or `/Users/me/projects/batch1`)
- The tool expects two subfolders inside:
  - `refs/` - Place your reference images here (1 or more)
  - `output/` - Generated images are saved here (created automatically if missing)
- All images in `refs/` are sent as references in each API request
- Path can be **relative** (to your current directory) or **absolute**

3. Run the tool:
```bash
./nano_banana_pro
```

4. Fill in the form:
   - **Folder Path**: Path to your folder (relative or absolute)
   - **Number of Images**: How many to generate (1-20, runs in parallel)
   - **Prompt**: Your generation prompt
   - **Aspect Ratio**: 1:1, 16:9, 9:16, 4:3, 3:4
   - **Image Size**: 1K, 2K, 4K

## Supported Image Formats

- `.jpg` / `.jpeg`
- `.png`
- `.gif`
- `.webp`

## Navigation

- `Tab` / `Enter` - Next field
- `Shift+Tab` - Previous field
- `Arrow keys` - Select options
