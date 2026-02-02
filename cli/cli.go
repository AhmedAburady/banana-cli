package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/briandowns/spinner"

	"github.com/AhmedAburady/banana-cli/api"
	"github.com/AhmedAburady/banana-cli/config"
	"golang.org/x/term"
)

// version is set at build time via ldflags
var version = "dev"

// GetVersion returns the version from ldflags or go build info
func GetVersion() string {
	if version != "" && version != "dev" {
		return version
	}

	// Get version from go install build info
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return "dev"
}

// Options holds CLI configuration
type Options struct {
	Prompt           string
	Output           string
	NumImages        int
	AspectRatio      string
	ImageSize        string
	Grounding        bool
	RefInput         string // -i flag, triggers edit mode if set
	PreserveFilename bool   // -r flag, preserve input filename for output (replace)
	UseVertex        bool   // -vertex flag, use Vertex AI instead of Gemini API
	Help             bool
	Version          bool
}

// Valid aspect ratios and image sizes
var (
	validAspectRatios = map[string]bool{
		"":     true, // Auto (not included in request)
		"1:1":  true,
		"16:9": true,
		"9:16": true,
		"4:3":  true,
		"3:4":  true,
		"2:3":  true,
		"3:2":  true,
		"5:4":  true,
		"4:5":  true,
		"21:9": true,
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
	flag.StringVar(&opts.AspectRatio, "ar", "", "Aspect ratio (default: Auto)")
	flag.StringVar(&opts.ImageSize, "s", "1K", "Image size: 1K, 2K, 4K")
	flag.BoolVar(&opts.Grounding, "g", false, "Enable grounding with Google Search")
	flag.StringVar(&opts.RefInput, "i", "", "Reference image/folder (enables edit mode)")
	flag.BoolVar(&opts.PreserveFilename, "r", false, "Replace: use input filename for output (single file only)")
	flag.BoolVar(&opts.UseVertex, "vertex", false, "Use Vertex AI instead of Gemini API (requires gcloud auth)")
	flag.BoolVar(&opts.Help, "help", false, "Show help message")
	flag.BoolVar(&opts.Version, "version", false, "Show version")
	flag.BoolVar(&opts.Version, "v", false, "Show version")

	flag.Parse()

	// CLI mode is active if prompt is provided
	cliMode := opts.Prompt != ""

	return opts, cliMode
}

// PrintVersion prints the version
func PrintVersion() {
	fmt.Printf("banana version %s\n", GetVersion())
}

// Validate validates the CLI options
func (opts *Options) Validate() error {
	// Prompt is required for CLI mode
	if opts.Prompt == "" {
		return fmt.Errorf("prompt is required (-p flag)")
	}

	// Check if prompt is a file path - if file exists, read prompt from it
	promptPath := api.ExpandTilde(opts.Prompt)
	if info, err := os.Stat(promptPath); err == nil && !info.IsDir() {
		data, err := os.ReadFile(promptPath)
		if err != nil {
			return fmt.Errorf("failed to read prompt file: %v", err)
		}
		opts.Prompt = strings.TrimSpace(string(data))
		if opts.Prompt == "" {
			return fmt.Errorf("prompt file is empty: %s", promptPath)
		}
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
		return fmt.Errorf("invalid aspect ratio: %s", opts.AspectRatio)
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
			// -r flag is only for single files, not folders
			if opts.PreserveFilename {
				return fmt.Errorf("-r flag only works with a single input file, not a folder")
			}
			count, _ := api.FindImagesInDir(opts.RefInput)
			if count == 0 {
				return fmt.Errorf("no images found in reference directory: %s", opts.RefInput)
			}
		} else if !api.IsSupportedImage(opts.RefInput) {
			return fmt.Errorf("unsupported image format: %s", opts.RefInput)
		}
	}

	// -r requires -i to be set with a single file
	if opts.PreserveFilename && opts.RefInput == "" {
		return fmt.Errorf("-r flag requires -i with an input image file")
	}

	return nil
}

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
	cfg := &api.Config{
		OutputFolder:     opts.Output,
		NumImages:        opts.NumImages,
		Prompt:           opts.Prompt,
		APIKey:           apiKey,
		AspectRatio:      opts.AspectRatio,
		ImageSize:        opts.ImageSize,
		Grounding:        opts.Grounding,
		RefImages:        refImages,
		RefInputPath:     opts.RefInput,
		PreserveFilename: opts.PreserveFilename,
		UseVertex:        opts.UseVertex,
	}

	// Ensure output folder exists
	if err := os.MkdirAll(cfg.OutputFolder, 0755); err != nil {
		fmt.Printf("\033[31mError:\033[0m Failed to create output folder: %v\n", err)
		os.Exit(1)
	}

	// Start spinner (CharSet 14 = braille dots)
	modeText := "Generating"
	if opts.RefInput != "" {
		modeText = "Editing"
	}
	if opts.UseVertex {
		modeText += " (Vertex AI)"
	}

	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond)
	s.Suffix = fmt.Sprintf(" %s %d image(s)...", modeText, opts.NumImages)
	s.Color("magenta")
	s.Start()

	// Run generation
	output := api.RunGeneration(cfg)

	// Stop spinner
	s.Stop()

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
	outputPath := cfg.OutputFolder
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
  banana                        Open interactive TUI
  banana [flags]                Generate/edit images from command line
  banana describe [flags]       Describe/analyze image style using AI
  banana config <command>       Manage configuration

Generate/Edit Flags:
  -p string    Prompt text or path to prompt file (required for CLI mode)
  -o string    Output folder (default ".")
  -n int       Number of images (default 1)
  -ar string   Aspect ratio (default: Auto)
  -s string    Image size: 1K, 2K, 4K (default "1K")
  -g           Enable grounding with Google Search
  -i string    Reference image/folder (enables edit mode)
  -r           Replace: use input filename for output (single file only)
  -vertex      Use Vertex AI instead of Gemini API (requires gcloud auth)
  --version    Show version
  --help       Show this help message

Describe Flags:
  -i string    Input image or folder (required)
  -o string    Output file path (default: stdout)
  -p string    Custom prompt (overrides default instruction)
  -a string    Additional instructions (prepended to default)
  -json        Output as structured JSON format

Config Commands:
  banana config set-key <KEY>   Save your Gemini API key
  banana config show            Show current configuration
  banana config path            Show config file location

Examples:
  banana -p "a sunset over mountains" -n 3
  banana -p prompt.txt -n 3                      # load prompt from file
  banana -i ./photo.png -p "make it cartoon style"
  banana -i ./photo.png -p "make it cartoon" -r  # output keeps name: photo.png
  banana -p "a futuristic city" -g -ar 16:9 -s 2K
  banana -i ./images/ -p "add rain effect" -n 2 -o ./output
  banana describe -i photo.jpg                   # analyze image style
  banana describe -i ./styles/ -o style.json    # analyze folder of images
`
	fmt.Print(help)
}

// HandleConfigCommand handles the config subcommand
func HandleConfigCommand(args []string) bool {
	if len(args) < 2 || args[0] != "config" {
		return false
	}

	switch args[1] {
	case "set-key":
		if len(args) < 3 {
			fmt.Println("Usage: banana config set-key <API_KEY>")
			os.Exit(1)
		}
		if err := config.SaveAPIKey(args[2]); err != nil {
			fmt.Printf("\033[31mError:\033[0m Failed to save API key: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\033[32m✓\033[0m API key saved successfully")
		fmt.Printf("  Location: %s\n", config.DefaultConfigPath())

	case "show":
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("\033[31mError:\033[0m Failed to load config: %v\n", err)
			os.Exit(1)
		}
		if cfg.APIKey == "" {
			fmt.Println("No API key configured")
			fmt.Println("Set one with: banana config set-key <YOUR_API_KEY>")
		} else {
			// Mask the API key, showing only first 8 and last 4 chars
			key := cfg.APIKey
			masked := key
			if len(key) > 12 {
				masked = key[:8] + "..." + key[len(key)-4:]
			}
			fmt.Printf("API Key: %s\n", masked)
		}

	case "path":
		fmt.Println(config.DefaultConfigPath())

	default:
		fmt.Printf("Unknown config command: %s\n", args[1])
		fmt.Println("Available commands: set-key, show, path")
		os.Exit(1)
	}

	return true
}

// PromptForAPIKey prompts the user to enter their API key
func PromptForAPIKey() string {
	fmt.Println("\033[33mNo API key found.\033[0m")
	fmt.Println()
	fmt.Println("Get your free API key from: https://aistudio.google.com/app/apikey")
	fmt.Println()
	fmt.Print("Enter your Gemini API key: ")

	// Read password without echoing to terminal
	keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // Add newline after hidden input
	if err != nil {
		fmt.Printf("\033[31mError:\033[0m Failed to read input: %v\n", err)
		os.Exit(1)
	}

	key := strings.TrimSpace(string(keyBytes))
	if key == "" {
		fmt.Println("\033[31mError:\033[0m API key cannot be empty")
		os.Exit(1)
	}

	// Save the key
	if err := config.SaveAPIKey(key); err != nil {
		fmt.Printf("\033[31mError:\033[0m Failed to save API key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\033[32m✓\033[0m API key saved successfully")
	fmt.Println()

	return key
}
