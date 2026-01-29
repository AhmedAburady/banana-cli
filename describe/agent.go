package describe

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
)

// StyleAnalysis represents the structured output for -json flag
// Designed to handle simple to complex image analysis with optional nested details
// The entire JSON serves as the comprehensive style guide for generation
type StyleAnalysis struct {
	// Core description (required) - summary of the style
	Description   string   `json:"description"`                  // Main style description (required)
	StyleSummary  string   `json:"style_summary,omitempty"`      // One-line style classification
	NegativePrompt []string `json:"negative_prompt,omitempty"`   // What to avoid in generation

	// Aspect & Format
	AspectRatio *AspectRatioInfo `json:"aspect_ratio,omitempty"`

	// Visual DNA - detailed visual characteristics
	VisualDNA *VisualDNA `json:"visual_dna,omitempty"`

	// Color information
	ColorPalette *ColorPalette `json:"color_palette,omitempty"`

	// Typography/Script (if present)
	ScriptClassification *ScriptInfo `json:"script_classification,omitempty"`
	CalligraphyLogic     *CalligraphyLogic `json:"calligraphy_logic,omitempty"`

	// Ornamentation & decorative elements
	Ornamentation *Ornamentation `json:"ornamentation,omitempty"`

	// Generation guidance
	RenderingDirectives []string `json:"rendering_directives,omitempty"`
	DoNotChange         []string `json:"do_not_change,omitempty"`

	// Variation & fidelity controls
	VariationControls *VariationControls `json:"variation_controls,omitempty"`
	FidelityTargets   *FidelityTargets   `json:"fidelity_targets,omitempty"`
}

// AspectRatioInfo describes image dimensions
type AspectRatioInfo struct {
	Ratio       string `json:"ratio,omitempty"`       // e.g., "1:1", "16:9"
	Orientation string `json:"orientation,omitempty"` // "square", "landscape", "portrait"
}

// VisualDNA captures detailed visual characteristics
type VisualDNA struct {
	StrokeDynamics      *StrokeDynamics      `json:"stroke_dynamics,omitempty"`
	Composition         *CompositionInfo     `json:"composition,omitempty"`
	ProductionAesthetic *ProductionAesthetic `json:"production_aesthetic,omitempty"`
}

// StrokeDynamics describes line and stroke qualities
type StrokeDynamics struct {
	WeightContrast string   `json:"weight_contrast,omitempty"` // "high", "low", "medium"
	EdgeQuality    string   `json:"edge_quality,omitempty"`    // "clean", "rough", "distressed"
	Terminals      string   `json:"terminals,omitempty"`       // "flat-cut", "rounded", "tapered"
	Flow           string   `json:"flow,omitempty"`            // "fluid", "staccato", "geometric"
	MicroFeatures  []string `json:"micro_features,omitempty"`  // Detailed stroke characteristics
}

// CompositionInfo describes layout and arrangement
type CompositionInfo struct {
	LayoutType      string `json:"layout_type,omitempty"`      // "centered", "diagonal", "grid"
	DensityStrategy string `json:"density_strategy,omitempty"` // "sparse", "balanced", "dense"
	RepetitionMode  string `json:"repetition_mode,omitempty"`  // "tiled", "mirrored", "non-repeating"
	Cropping        string `json:"cropping,omitempty"`         // "full-bleed", "contained", "partial"
	BalanceNotes    string `json:"balance_notes,omitempty"`    // Description of visual balance
}

// ProductionAesthetic describes the medium and finish
type ProductionAesthetic struct {
	Medium       string   `json:"medium,omitempty"`        // "digital", "screenprint", "watercolor"
	Artefacts    []string `json:"artefacts,omitempty"`     // "grain", "ink bleed", "scratches"
	Finish       string   `json:"finish,omitempty"`        // "matte", "glossy", "textured"
	TextureNotes string   `json:"texture_notes,omitempty"` // Detailed texture description
}

// ColorPalette describes colors used
type ColorPalette struct {
	Hex                 []string `json:"hex,omitempty"`                   // Color hex codes
	PaletteRules        string   `json:"palette_rules,omitempty"`         // How colors are used
	BackgroundTreatment string   `json:"background_treatment,omitempty"`  // Background color/treatment
	OverprintOrHalftone string   `json:"overprint_or_halftone,omitempty"` // Print technique
}

// ScriptInfo describes typography/calligraphy classification
type ScriptInfo struct {
	ScriptFamily string `json:"script_family,omitempty"` // "Kufic", "Thuluth", "Sans-serif"
	Legibility   string `json:"legibility,omitempty"`    // "legible", "decorative", "abstract"
	HybridNotes  string `json:"hybrid_notes,omitempty"`  // Notes on mixed styles
}

// CalligraphyLogic describes calligraphy-specific rules
type CalligraphyLogic struct {
	JoinBehaviour    string `json:"join_behaviour,omitempty"`    // How letters connect
	BaselineLogic    string `json:"baseline_logic,omitempty"`    // Baseline treatment
	KashidaStrategy  string `json:"kashida_strategy,omitempty"`  // Arabic extension strategy
	DiacriticStrategy string `json:"diacritic_strategy,omitempty"` // Dots/marks handling
	LetterformRules  string `json:"letterform_rules,omitempty"`  // Specific letter rules
}

// Ornamentation describes decorative elements
type Ornamentation struct {
	ElementsPresent []string `json:"elements_present,omitempty"` // List of decorative elements
	Rules           []string `json:"rules,omitempty"`            // Rules for ornamentation
}

// VariationControls guides what can/cannot be changed
type VariationControls struct {
	SafeToVary   []string `json:"safe_to_vary,omitempty"`   // Elements that can change
	UnsafeToVary []string `json:"unsafe_to_vary,omitempty"` // Elements that must stay
	RangeNotes   string   `json:"range_notes,omitempty"`    // Variation guidance
}

// FidelityTargets describes matching priorities
type FidelityTargets struct {
	MustMatch   []string `json:"must_match,omitempty"`   // Critical elements
	ShouldMatch []string `json:"should_match,omitempty"` // Important elements
	CanExplore  []string `json:"can_explore,omitempty"`  // Flexible elements
	Assumptions []string `json:"assumptions,omitempty"`  // Inferred information
}

// DescriptionResult holds the output from the describe agent
type DescriptionResult struct {
	Text     string         // Plain text output
	Analysis *StyleAnalysis // Structured output (when -json)
	IsJSON   bool
}

// DescribeAgent wraps the ADK agent for image description
type DescribeAgent struct {
	apiKey string
}

// NewDescribeAgent creates a new ADK-powered describe agent
func NewDescribeAgent(ctx context.Context, apiKey string) (*DescribeAgent, error) {
	return &DescribeAgent{
		apiKey: apiKey,
	}, nil
}

// defaultTextInstruction returns instruction for plain text output
func defaultTextInstruction() string {
	return `You are an expert image style analyst. Analyze the provided image(s) and create a detailed description that captures the visual style.

When multiple images are provided, identify the UNIFIED style elements across all images - treat them as style references to extract a cohesive style description.

Your description should be detailed enough that it can be used as a prompt to recreate this exact style. Focus on what you actually SEE in the image(s):

- Visual characteristics (colors, shapes, patterns, textures)
- Art style if applicable (illustration, photography, 3D render, vector art, etc.)
- Mood and atmosphere
- Any distinctive visual elements
- Composition and layout characteristics

Be specific and descriptive. The description you write will be used directly as a generation prompt, so make it actionable and clear.

IMPORTANT: Only describe what is actually present in the image. Don't make up elements that aren't there. If something like lighting or perspective isn't relevant (e.g., flat vector art), don't mention it.

Output only the description text, nothing else. No preamble, no "Here is the description:", just the description itself.`
}

// jsonOutputInstruction returns instruction for structured JSON output
func jsonOutputInstruction() string {
	return `You are an expert image style analyst. Analyze the provided image(s) and produce a COMPREHENSIVE structured style guide.

THE ENTIRE JSON OUTPUT IS THE STYLE GUIDE - be thorough and fill in as many fields as possible.

FILL THESE FIELDS:
- description: Detailed style description (REQUIRED)
- style_summary: One-line style classification
- negative_prompt: What to AVOID when recreating
- color_palette.hex: Extract hex color codes from the image
- color_palette.palette_rules: How colors are used together
- color_palette.background_treatment: Background color/style
- visual_dna.stroke_dynamics: Line qualities, edge treatment, weight
- visual_dna.composition: Layout type, density, repetition mode, balance
- visual_dna.production_aesthetic: Medium, finish, texture notes
- rendering_directives: Technical instructions for recreation
- do_not_change: Critical elements that define this style
- ornamentation: Decorative elements and their rules
- variation_controls: What can/cannot vary while maintaining style
- fidelity_targets: must_match, should_match, can_explore priorities
- script_classification: For text/typography/calligraphy
- calligraphy_logic: For calligraphic works

Extract ACTUAL hex color codes from the image. Be comprehensive and detailed.`
}

// createOutputSchema builds the JSON schema for structured output
func createOutputSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			// Core fields
			"description":    {Type: genai.TypeString, Description: "Main style description - detailed summary of the visual style"},
			"style_summary":  {Type: genai.TypeString, Description: "One-line style classification"},
			"negative_prompt": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "Elements to avoid in generation"},

			// Aspect ratio
			"aspect_ratio": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"ratio":       {Type: genai.TypeString, Description: "Ratio like 1:1, 16:9, 4:3"},
					"orientation": {Type: genai.TypeString, Description: "square, landscape, or portrait"},
				},
			},

			// Visual DNA - detailed characteristics
			"visual_dna": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"stroke_dynamics": {
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"weight_contrast": {Type: genai.TypeString, Description: "high, low, medium"},
							"edge_quality":    {Type: genai.TypeString, Description: "clean, rough, distressed, soft"},
							"terminals":       {Type: genai.TypeString, Description: "flat-cut, rounded, tapered"},
							"flow":            {Type: genai.TypeString, Description: "fluid, staccato, geometric"},
							"micro_features":  {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "Detailed stroke characteristics"},
						},
					},
					"composition": {
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"layout_type":      {Type: genai.TypeString, Description: "centered, diagonal, grid, radial, etc."},
							"density_strategy": {Type: genai.TypeString, Description: "sparse, balanced, dense"},
							"repetition_mode":  {Type: genai.TypeString, Description: "tiled, mirrored, non-repeating"},
							"cropping":         {Type: genai.TypeString, Description: "full-bleed, contained, partial"},
							"balance_notes":    {Type: genai.TypeString, Description: "Description of visual balance"},
						},
					},
					"production_aesthetic": {
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"medium":        {Type: genai.TypeString, Description: "digital, screenprint, watercolor, oil, etc."},
							"artefacts":     {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "grain, ink bleed, scratches, noise"},
							"finish":        {Type: genai.TypeString, Description: "matte, glossy, textured, vintage"},
							"texture_notes": {Type: genai.TypeString, Description: "Detailed texture description"},
						},
					},
				},
			},

			// Color palette
			"color_palette": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"hex":                   {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "Color hex codes"},
					"palette_rules":        {Type: genai.TypeString, Description: "How colors are used together"},
					"background_treatment": {Type: genai.TypeString, Description: "Background color and treatment"},
					"overprint_or_halftone": {Type: genai.TypeString, Description: "Print technique if applicable"},
				},
			},

			// Typography/Script classification
			"script_classification": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"script_family": {Type: genai.TypeString, Description: "Font family or script type"},
					"legibility":    {Type: genai.TypeString, Description: "legible, decorative, abstract"},
					"hybrid_notes":  {Type: genai.TypeString, Description: "Notes on mixed styles"},
				},
			},

			// Calligraphy logic
			"calligraphy_logic": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"join_behaviour":    {Type: genai.TypeString, Description: "How letters connect"},
					"baseline_logic":    {Type: genai.TypeString, Description: "Baseline treatment"},
					"kashida_strategy":  {Type: genai.TypeString, Description: "Extension strategy for Arabic"},
					"diacritic_strategy": {Type: genai.TypeString, Description: "Dots and marks handling"},
					"letterform_rules":  {Type: genai.TypeString, Description: "Specific letterform rules"},
				},
			},

			// Ornamentation
			"ornamentation": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"elements_present": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "Decorative elements"},
					"rules":            {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "Rules for ornamentation"},
				},
			},

			// Generation guidance
			"rendering_directives": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "Specific rendering instructions"},
			"do_not_change":        {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "Critical elements that must be preserved"},

			// Variation controls
			"variation_controls": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"safe_to_vary":   {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "Elements that can change"},
					"unsafe_to_vary": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "Elements that must stay fixed"},
					"range_notes":    {Type: genai.TypeString, Description: "Guidance on variation range"},
				},
			},

			// Fidelity targets
			"fidelity_targets": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"must_match":   {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "Critical elements to match"},
					"should_match": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "Important elements to match"},
					"can_explore":  {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "Flexible elements"},
					"assumptions":  {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}, Description: "Inferred information"},
				},
			},
		},
		Required: []string{"description"},
	}
}

// extractTextFromParts extracts text content from genai.Part slice
func extractTextFromParts(parts []*genai.Part) string {
	for _, part := range parts {
		if part == nil {
			continue
		}
		if part.Text != "" {
			return part.Text
		}
	}
	return ""
}

// DescribeImages analyzes images using ADK agent
func (a *DescribeAgent) DescribeImages(ctx context.Context, imageParts []*genai.Part, customPrompt string, additional string, jsonOutput bool) (*DescriptionResult, error) {
	llmModel, err := gemini.NewModel(ctx, "gemini-3-pro-preview", &genai.ClientConfig{
		APIKey: a.apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	// Determine instruction: -p overrides default, -a prepends context to default
	var instruction string
	if customPrompt != "" {
		// -p flag: completely override default instruction
		instruction = customPrompt
	} else if jsonOutput {
		instruction = jsonOutputInstruction()
	} else {
		instruction = defaultTextInstruction()
	}

	// -a flag: prepend additional instructions so they take priority
	if additional != "" {
		instruction = "CRITICAL USER CONTEXT - You MUST incorporate this into your analysis:\n" + additional + "\n\n" + instruction
	}

	// Create agent config with thinking enabled for better analysis
	agentConfig := llmagent.Config{
		Name:        "style_analyzer",
		Model:       llmModel,
		Description: "Analyzes images and extracts style descriptions",
		Instruction: instruction,
		GenerateContentConfig: &genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				ThinkingLevel: genai.ThinkingLevelHigh,
			},
		},
	}
	// -json flag always uses schema
	if jsonOutput {
		agentConfig.OutputSchema = createOutputSchema()
	}

	describeAgent, err := llmagent.New(agentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	sessionService := session.InMemoryService()

	// Create session before running
	_, err = sessionService.Create(ctx, &session.CreateRequest{
		AppName:   "banana-describe",
		UserID:    "cli-user",
		SessionID: "describe-session",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	r, err := runner.New(runner.Config{
		AppName:        "banana-describe",
		Agent:          describeAgent,
		SessionService: sessionService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	parts := make([]*genai.Part, 0, len(imageParts)+1)
	parts = append(parts, imageParts...)
	parts = append(parts, genai.NewPartFromText("Analyze the style of the provided image(s)."))

	userContent := &genai.Content{
		Role:  "user",
		Parts: parts,
	}

	var resultText string
	for event, err := range r.Run(ctx, "cli-user", "describe-session", userContent, agent.RunConfig{}) {
		if err != nil {
			return nil, fmt.Errorf("agent run error: %w", err)
		}
		if event.IsFinalResponse() && event.Content != nil {
			resultText = extractTextFromParts(event.Content.Parts)
			if resultText != "" {
				break
			}
		}
	}

	if resultText == "" {
		return nil, fmt.Errorf("no description generated")
	}

	return parseResult(resultText, jsonOutput)
}

// parseResult parses the model output into DescriptionResult
func parseResult(text string, isStructured bool) (*DescriptionResult, error) {
	result := &DescriptionResult{IsJSON: isStructured}

	if isStructured {
		var analysis StyleAnalysis
		if err := json.Unmarshal([]byte(text), &analysis); err != nil {
			// If JSON parsing fails, return as plain text
			result.Text = text
			result.IsJSON = false
		} else {
			result.Analysis = &analysis
		}
	} else {
		result.Text = text
	}

	return result, nil
}

// FormatOutput returns the result formatted for display
func (r *DescriptionResult) FormatOutput() string {
	if r.IsJSON && r.Analysis != nil {
		data, _ := json.MarshalIndent(r.Analysis, "", "  ")
		return string(data)
	}
	return r.Text
}
