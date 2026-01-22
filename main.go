package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

const (
	GeminiModel = "gemini-3-pro-image-preview"
	GeminiURL   = "https://generativelanguage.googleapis.com/v1beta/models/gemini-3-pro-image-preview:generateContent"
)

// Gemini API structures
type InlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type Part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *InlineData `json:"inline_data,omitempty"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type ImageConfig struct {
	AspectRatio string `json:"aspectRatio"`
	ImageSize   string `json:"imageSize"`
}

type GenerationConfig struct {
	ResponseModalities []string    `json:"responseModalities"`
	ImageConfig        ImageConfig `json:"imageConfig"`
}

type GoogleSearch struct{}

type Tool struct {
	GoogleSearch *GoogleSearch `json:"googleSearch,omitempty"`
}

type GeminiRequest struct {
	Contents         []Content        `json:"contents"`
	GenerationConfig GenerationConfig `json:"generationConfig"`
	Tools            []Tool           `json:"tools,omitempty"`
}

// Response structures
type ResponseInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type ResponsePart struct {
	Text       string              `json:"text,omitempty"`
	InlineData *ResponseInlineData `json:"inlineData,omitempty"`
}

type ResponseContent struct {
	Parts []ResponsePart `json:"parts"`
	Role  string         `json:"role"`
}

type Candidate struct {
	Content ResponseContent `json:"content"`
}

type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
}

// Config holds application configuration
type Config struct {
	FolderPath  string
	NumImages   int
	Prompt      string
	APIKey      string
	AspectRatio string
	ImageSize   string
	Grounding   bool
}

func main() {
	printBanner()

	// Get API key from environment
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if apiKey == "" {
		fmt.Println("❌ Error: GEMINI_API_KEY environment variable not set")
		fmt.Println("   Set it with: export GEMINI_API_KEY=your_key")
		os.Exit(1)
	}

	config, err := runForm(apiKey)
	if err != nil {
		if err == huh.ErrUserAborted {
			os.Exit(0)
		}
		fmt.Printf("\n❌ Error: %v\n", err)
		os.Exit(1)
	}

	// Validate folder structure
	refsPath := filepath.Join(config.FolderPath, "refs")
	outputPath := filepath.Join(config.FolderPath, "output")

	if err := validateAndCreateDirs(refsPath, outputPath); err != nil {
		fmt.Printf("\n❌ Error: %v\n", err)
		os.Exit(1)
	}

	// Load reference images
	refImages, err := loadReferenceImages(refsPath)
	if err != nil {
		fmt.Printf("\n❌ Error loading reference images: %v\n", err)
		os.Exit(1)
	}

	if len(refImages) == 0 {
		fmt.Printf("\n❌ No reference images found in %s\n", refsPath)
		fmt.Println("   Supported formats: .jpg, .jpeg, .png, .gif, .webp")
		os.Exit(1)
	}

	fmt.Printf("\n📁 Found %d reference image(s)\n", len(refImages))

	// Generate images in parallel with spinner
	var wg sync.WaitGroup
	results := make(chan GenerationResult, config.NumImages)
	var generationResults []GenerationResult

	startTime := time.Now()

	err = spinner.New().
		Title(fmt.Sprintf("Generating %d image(s) in parallel...", config.NumImages)).
		Action(func() {
			for i := 0; i < config.NumImages; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					result := generateImage(config, refImages, index)
					results <- result
				}(i)
			}

			// Close results channel when all goroutines complete
			go func() {
				wg.Wait()
				close(results)
			}()

			// Collect results
			for result := range results {
				generationResults = append(generationResults, result)
			}
		}).
		Run()

	if err != nil {
		fmt.Printf("\n❌ Error: %v\n", err)
		os.Exit(1)
	}

	// Process and save results
	successCount := 0
	errorCount := 0

	fmt.Println()
	for _, result := range generationResults {
		if result.Error != nil {
			fmt.Printf("❌ Image %d failed: %v\n", result.Index+1, result.Error)
			errorCount++
			continue
		}

		// Save the image
		filename := fmt.Sprintf("generated_%d_%s.png", result.Index+1, time.Now().Format("20060102_150405"))
		outputFile := filepath.Join(outputPath, filename)

		if err := os.WriteFile(outputFile, result.ImageData, 0644); err != nil {
			fmt.Printf("❌ Failed to save image %d: %v\n", result.Index+1, err)
			errorCount++
			continue
		}

		fmt.Printf("✅ Image %d saved: %s\n", result.Index+1, filename)
		successCount++
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📊 Results: %d success, %d failed\n", successCount, errorCount)
	fmt.Printf("⏱️  Time: %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("📂 Output: %s\n", outputPath)
}

type GenerationResult struct {
	Index     int
	ImageData []byte
	Error     error
}

func customTheme() *huh.Theme {
	t := huh.ThemeDracula()
	yellow := lipgloss.Color("#FFFF00")

	t.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(yellow)
	t.Blurred.TextInput.Text = lipgloss.NewStyle().Foreground(yellow)

	return t
}

func printBanner() {
	banner := `
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃     🍌 Nano Banana Pro - Image Generator     ┃
┃        Gemini AI Pattern Generator           ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
`
	fmt.Println(banner)
}

func runForm(apiKey string) (*Config, error) {
	// Default prompt - pre-filled for editing
	defaultPrompt := "A 2D vector art pattern in the style of the reference image/s not a copy of it not an immitation not an edit of it rather, a pattern inspired by the shapes in reference image/s you can get creative with colors and avoid extremely bold outlines"

	var (
		folderPath  string
		numImages   string
		prompt      = defaultPrompt
		aspectRatio string
		imageSize   string
		grounding   = true
	)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Folder Path").
				Description("Parent folder containing refs/ and output/ directories").
				Placeholder("./input/01").
				Value(&folderPath).
				Validate(func(s string) error {
					path := s
					if path == "" {
						path = "./input/01"
					}
					refsPath := filepath.Join(path, "refs")
					if _, err := os.Stat(refsPath); os.IsNotExist(err) {
						return fmt.Errorf("refs/ folder not found in %s", path)
					}
					return nil
				}),

			huh.NewInput().
				Title("Number of Images").
				Description("How many images to generate (in parallel)").
				Placeholder("5").
				Value(&numImages).
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					n, err := strconv.Atoi(s)
					if err != nil || n < 1 || n > 20 {
						return fmt.Errorf("please enter a number between 1 and 20")
					}
					return nil
				}),

			huh.NewText().
				Title("Prompt").
				Description("The generation prompt (Ctrl+U to clear)").
				Value(&prompt).
				Lines(3),

			huh.NewSelect[string]().
				Title("Aspect Ratio").
				Options(
					huh.NewOption("1:1 (Square)", "1:1"),
					huh.NewOption("16:9 (Landscape)", "16:9"),
					huh.NewOption("9:16 (Portrait)", "9:16"),
					huh.NewOption("4:3", "4:3"),
					huh.NewOption("3:4", "3:4"),
				).
				Value(&aspectRatio),

			huh.NewSelect[string]().
				Title("Image Size").
				Options(
					huh.NewOption("1K", "1K"),
					huh.NewOption("2K", "2K"),
					huh.NewOption("4K", "4K"),
				).
				Value(&imageSize),

			huh.NewConfirm().
				Title("Grounding").
				Description("Enable Google Search grounding").
				Affirmative("Enabled").
				Negative("Disabled").
				Value(&grounding),
		),
	).WithTheme(customTheme())

	err := form.Run()
	if err != nil {
		return nil, err
	}

	// Apply defaults
	if folderPath == "" {
		folderPath = "./input/01"
	}
	if numImages == "" {
		numImages = "5"
	}
	if prompt == "" {
		prompt = defaultPrompt
	}
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	if imageSize == "" {
		imageSize = "2K"
	}

	n, _ := strconv.Atoi(numImages)

	return &Config{
		FolderPath:  folderPath,
		NumImages:   n,
		Prompt:      prompt,
		APIKey:      apiKey,
		AspectRatio: aspectRatio,
		ImageSize:   imageSize,
		Grounding:   grounding,
	}, nil
}

func validateAndCreateDirs(refsPath, outputPath string) error {
	// Check refs directory exists
	if _, err := os.Stat(refsPath); os.IsNotExist(err) {
		return fmt.Errorf("refs directory does not exist: %s", refsPath)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	return nil
}

func loadReferenceImages(refsPath string) ([]Part, error) {
	var parts []Part

	supportedExts := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
	}

	entries, err := os.ReadDir(refsPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		mimeType, ok := supportedExts[ext]
		if !ok {
			continue
		}

		filePath := filepath.Join(refsPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %v", entry.Name(), err)
		}

		encoded := base64.StdEncoding.EncodeToString(data)
		parts = append(parts, Part{
			InlineData: &InlineData{
				MimeType: mimeType,
				Data:     encoded,
			},
		})

		fmt.Printf("  📷 Loaded: %s\n", entry.Name())
	}

	return parts, nil
}

func generateImage(config *Config, refImages []Part, index int) GenerationResult {
	// Build request parts: prompt + all reference images
	parts := []Part{{Text: config.Prompt}}
	parts = append(parts, refImages...)

	request := GeminiRequest{
		Contents: []Content{{Parts: parts}},
		GenerationConfig: GenerationConfig{
			ResponseModalities: []string{"TEXT", "IMAGE"},
			ImageConfig: ImageConfig{
				AspectRatio: config.AspectRatio,
				ImageSize:   config.ImageSize,
			},
		},
	}

	if config.Grounding {
		request.Tools = []Tool{{GoogleSearch: &GoogleSearch{}}}
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return GenerationResult{Index: index, Error: fmt.Errorf("failed to marshal request: %v", err)}
	}

	// Build URL with API key
	url := fmt.Sprintf("%s?key=%s", GeminiURL, config.APIKey)

	// Make request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return GenerationResult{Index: index, Error: fmt.Errorf("request failed: %v", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GenerationResult{Index: index, Error: fmt.Errorf("failed to read response: %v", err)}
	}

	if resp.StatusCode != http.StatusOK {
		return GenerationResult{Index: index, Error: fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))}
	}

	// Parse response
	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return GenerationResult{Index: index, Error: fmt.Errorf("failed to parse response: %v", err)}
	}

	// Extract image from response
	for _, candidate := range geminiResp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil && strings.HasPrefix(part.InlineData.MimeType, "image/") {
				imageData, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
				if err != nil {
					return GenerationResult{Index: index, Error: fmt.Errorf("failed to decode image: %v", err)}
				}
				return GenerationResult{Index: index, ImageData: imageData}
			}
		}
	}

	// No image found - check if there's text explaining why
	var textResponse string
	for _, candidate := range geminiResp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				textResponse = part.Text
			}
		}
	}
	if textResponse != "" {
		return GenerationResult{Index: index, Error: fmt.Errorf("no image in response. API said: %s", textResponse)}
	}

	return GenerationResult{Index: index, Error: fmt.Errorf("no image in response (raw: %s)", string(body))}
}
