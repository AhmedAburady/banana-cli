package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AhmedAburady/banana-cli/api"
)

// Options holds CLI configuration
type Options struct {
	Prompt      string
	Output      string
	NumImages   int
	AspectRatio string
	ImageSize   string
	Grounding   bool
	RefInput    string // -i flag, triggers edit mode if set
	Help        bool
}

// Valid aspect ratios and image sizes
var (
	validAspectRatios = map[string]bool{
		"1:1": true, "16:9": true, "9:16": true, "4:3": true, "3:4": true,
	}
	validImageSizes = map[string]bool{
		"1K": true, "2K": true, "4K": true,
	}
)

// ParseFlags parses CLI flags and returns options and whether CLI mode is active
func ParseFlags() (*Options, bool) {
	opts := &Options{}

	flag.StringVar(&opts.Prompt, "p", "", "Prompt text (required for CLI mode)")
	flag.StringVar(&opts.Output, "o", ".", "Output folder")
	flag.IntVar(&opts.NumImages, "n", 1, "Number of images to generate (1-20)")
	flag.StringVar(&opts.AspectRatio, "ar", "1:1", "Aspect ratio: 1:1, 16:9, 9:16, 4:3, 3:4")
	flag.StringVar(&opts.ImageSize, "s", "1K", "Image size: 1K, 2K, 4K")
	flag.BoolVar(&opts.Grounding, "g", false, "Enable grounding with Google Search")
	flag.StringVar(&opts.RefInput, "i", "", "Reference image/folder (enables edit mode)")
	flag.BoolVar(&opts.Help, "help", false, "Show help message")

	flag.Parse()

	// CLI mode is active if prompt is provided
	cliMode := opts.Prompt != ""

	return opts, cliMode
}

// Validate validates the CLI options
func (opts *Options) Validate() error {
	// Prompt is required for CLI mode
	if opts.Prompt == "" {
		return fmt.Errorf("prompt is required (-p flag)")
	}

	// Expand tilde in paths
	opts.Output = api.ExpandTilde(opts.Output)
	opts.RefInput = api.ExpandTilde(opts.RefInput)

	// Validate number of images
	if opts.NumImages < 1 || opts.NumImages > 20 {
		return fmt.Errorf("number of images must be between 1 and 20")
	}

	// Validate aspect ratio
	if !validAspectRatios[opts.AspectRatio] {
		return fmt.Errorf("invalid aspect ratio: %s (valid: 1:1, 16:9, 9:16, 4:3, 3:4)", opts.AspectRatio)
	}

	// Validate image size
	if !validImageSizes[opts.ImageSize] {
		return fmt.Errorf("invalid image size: %s (valid: 1K, 2K, 4K)", opts.ImageSize)
	}

	// Validate reference input if provided (edit mode)
	if opts.RefInput != "" {
		info, err := os.Stat(opts.RefInput)
		if os.IsNotExist(err) {
			return fmt.Errorf("reference path does not exist: %s", opts.RefInput)
		}
		if err != nil {
			return fmt.Errorf("cannot access reference path: %v", err)
		}

		if info.IsDir() {
			count, _ := api.FindImagesInDir(opts.RefInput)
			if count == 0 {
				return fmt.Errorf("no images found in reference directory: %s", opts.RefInput)
			}
		} else if !api.IsSupportedImage(opts.RefInput) {
			return fmt.Errorf("unsupported image format: %s", opts.RefInput)
		}
	}

	return nil
}

// Spinner frames for terminal spinner
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Run executes CLI mode with terminal spinner
func Run(opts *Options, apiKey string) {
	// Validate options
	if err := opts.Validate(); err != nil {
		fmt.Printf("\033[31mError:\033[0m %v\n", err)
		os.Exit(1)
	}

	// Load reference images if in edit mode
	var refImages []api.Part
	if opts.RefInput != "" {
		var err error
		refImages, err = api.LoadReferences(opts.RefInput)
		if err != nil {
			fmt.Printf("\033[31mError:\033[0m Failed to load references: %v\n", err)
			os.Exit(1)
		}
	}

	// Create config
	config := &api.Config{
		OutputFolder: opts.Output,
		NumImages:    opts.NumImages,
		Prompt:       opts.Prompt,
		APIKey:       apiKey,
		AspectRatio:  opts.AspectRatio,
		ImageSize:    opts.ImageSize,
		Grounding:    opts.Grounding,
		RefImages:    refImages,
	}

	// Ensure output folder exists
	if err := os.MkdirAll(config.OutputFolder, 0755); err != nil {
		fmt.Printf("\033[31mError:\033[0m Failed to create output folder: %v\n", err)
		os.Exit(1)
	}

	// Start spinner
	stopSpinner := make(chan bool)
	spinnerDone := make(chan bool)

	modeText := "Generating"
	if opts.RefInput != "" {
		modeText = "Editing"
	}
	spinnerMsg := fmt.Sprintf("%s %d image(s)...", modeText, opts.NumImages)

	go func() {
		frameIdx := 0
		for {
			select {
			case <-stopSpinner:
				// Clear spinner line
				fmt.Print("\r\033[K")
				spinnerDone <- true
				return
			default:
				fmt.Printf("\r\033[35m%s\033[0m %s", spinnerFrames[frameIdx], spinnerMsg)
				frameIdx = (frameIdx + 1) % len(spinnerFrames)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()

	// Run generation
	output := api.RunGeneration(config)

	// Stop spinner
	stopSpinner <- true
	<-spinnerDone

	// Print results
	fmt.Println()
	successCount := 0
	errorCount := 0

	for _, r := range output.Results {
		if r.Error != nil {
			fmt.Printf("\033[31m✗\033[0m Image %d: %v\n", r.Index+1, r.Error)
			errorCount++
		} else {
			fmt.Printf("\033[32m✓\033[0m %s\n", r.Filename)
			successCount++
		}
	}

	fmt.Println()
	fmt.Printf("Done: %d success, %d failed (%.1fs)\n", successCount, errorCount, output.Elapsed.Seconds())

	// Show output path as absolute if it was relative
	outputPath := config.OutputFolder
	if !filepath.IsAbs(outputPath) {
		if abs, err := filepath.Abs(outputPath); err == nil {
			outputPath = abs
		}
	}
	fmt.Printf("Output: %s\n", outputPath)

	if errorCount > 0 {
		os.Exit(1)
	}
}

// PrintHelp prints the usage help message
func PrintHelp() {
	help := `
BANANA CLI - Gemini AI Image Generator

Usage:
  banana                     Open interactive TUI
  banana [flags]             Generate/edit images from command line

Flags:
  -p string    Prompt (required for CLI mode)
  -o string    Output folder (default ".")
  -n int       Number of images (default 1)
  -ar string   Aspect ratio: 1:1, 16:9, 9:16, 4:3, 3:4 (default "1:1")
  -s string    Image size: 1K, 2K, 4K (default "1K")
  -g           Enable grounding with Google Search
  -i string    Reference image/folder (enables edit mode)
  --help       Show this help message

Examples:
  banana -p "a sunset over mountains" -n 3
  banana -i ./photo.png -p "make it cartoon style"
  banana -p "a futuristic city" -g -ar 16:9 -s 2K
  banana -i ./images/ -p "add rain effect" -n 2 -o ./output
`
	fmt.Print(help)
}
