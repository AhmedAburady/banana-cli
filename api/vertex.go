package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"google.golang.org/genai"
)

const (
	// VertexModel is the model name for Vertex AI image generation
	// Same model as direct Gemini API (gemini-3-pro-image-preview / nano banana)
	VertexModel = "gemini-3-pro-image-preview"
)

// getVertexConfig returns project and location from environment variables
func getVertexConfig() (project, location string, err error) {
	project = os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		project = os.Getenv("GCLOUD_PROJECT")
	}
	if project == "" {
		return "", "", fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable is required for Vertex AI")
	}

	location = os.Getenv("GOOGLE_CLOUD_LOCATION")
	if location == "" {
		location = "global" // Default location for Gemini 3 models
	}

	return project, location, nil
}

// GenerateImageVertex performs a single image generation request using Vertex AI
func GenerateImageVertex(config *Config, index int) GenerationResult {
	ctx := context.Background()

	project, location, err := getVertexConfig()
	if err != nil {
		return GenerationResult{Index: index, Error: err}
	}

	// Create Vertex AI client using Application Default Credentials
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return GenerationResult{Index: index, Error: fmt.Errorf("failed to create Vertex AI client: %w", err)}
	}

	// Build content parts
	var parts []*genai.Part
	parts = append(parts, genai.NewPartFromText(config.Prompt))

	// Add reference images if in edit mode
	for _, refImg := range config.RefImages {
		if refImg.InlineData != nil {
			// Decode base64 image data
			imageData, err := base64.StdEncoding.DecodeString(refImg.InlineData.Data)
			if err != nil {
				return GenerationResult{Index: index, Error: fmt.Errorf("failed to decode reference image: %w", err)}
			}
			parts = append(parts, &genai.Part{
				InlineData: &genai.Blob{
					MIMEType: refImg.InlineData.MimeType,
					Data:     imageData,
				},
			})
		}
	}

	contents := []*genai.Content{
		{
			Parts: parts,
			Role:  "user",
		},
	}

	// Configure generation settings
	genConfig := &genai.GenerateContentConfig{
		ResponseModalities: []string{"TEXT", "IMAGE"},
	}

	// Call the API
	resp, err := client.Models.GenerateContent(ctx, VertexModel, contents, genConfig)
	if err != nil {
		return GenerationResult{Index: index, Error: fmt.Errorf("generation failed: %w", err)}
	}

	// Extract image from response
	for _, candidate := range resp.Candidates {
		if candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil && len(part.InlineData.Data) > 0 {
				return GenerationResult{Index: index, ImageData: part.InlineData.Data}
			}
		}
	}

	// No image found - check if there's text explaining why
	for _, candidate := range resp.Candidates {
		if candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				return GenerationResult{Index: index, Error: fmt.Errorf("no image in response. API said: %s", part.Text)}
			}
		}
	}

	return GenerationResult{Index: index, Error: fmt.Errorf("no image in response")}
}
