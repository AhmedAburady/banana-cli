package ui

// FormConfig defines a form configuration
type FormConfig struct {
	Title  string
	Fields []FieldConfig
}

// FieldConfig defines a field configuration
type FieldConfig struct {
	Type        FieldType
	Key         string
	Label       string
	Description string
	Placeholder string
	Default     string
	BoolDefault bool
	Lines       int           // For textarea
	DirsOnly    bool          // For path
	AllowedExts []string      // For path
	Options     []SelectOption // For select
	DefaultIdx  int           // For select
}

// Common field configs shared between forms
var (
	OutputFolderField = FieldConfig{
		Type:        FieldPath,
		Key:         "output",
		Label:       "Output Folder",
		Description: "Where to save generated images",
		Placeholder: ".",
		Default:     ".",
		DirsOnly:    true,
	}

	NumImagesField = FieldConfig{
		Type:        FieldInput,
		Key:         "num",
		Label:       "Number of Images",
		Description: "How many images to generate (1-20)",
		Placeholder: "5",
		Default:     "5",
	}

	AspectRatioField = FieldConfig{
		Type:       FieldSelect,
		Key:        "aspect",
		Label:      "Aspect Ratio",
		Options:    AspectRatioOptions(),
		DefaultIdx: 0,
	}

	ImageSizeField = FieldConfig{
		Type:       FieldSelect,
		Key:        "size",
		Label:      "Image Size",
		Options:    ImageSizeOptions(),
		DefaultIdx: 1, // 2K default
	}

	GroundingField = FieldConfig{
		Type:        FieldToggle,
		Key:         "grounding",
		Label:       "Grounding",
		Description: "Enable Google Search grounding",
		BoolDefault: false,
	}
)

// GenerateFormConfig returns the configuration for the generate form
func GenerateFormConfig() FormConfig {
	return FormConfig{
		Title: "Generate Image",
		Fields: []FieldConfig{
			OutputFolderField,
			NumImagesField,
			{
				Type:        FieldTextArea,
				Key:         "prompt",
				Label:       "Prompt",
				Description: "Describe what you want to generate",
				Placeholder: "A beautiful sunset over mountains...",
				Lines:       3,
			},
			AspectRatioField,
			ImageSizeField,
			GroundingField,
		},
	}
}

// EditFormConfig returns the configuration for the edit form
func EditFormConfig() FormConfig {
	imageExts := []string{".png", ".jpg", ".jpeg", ".gif", ".webp"}

	return FormConfig{
		Title: "Edit Image",
		Fields: []FieldConfig{
			{
				Type:        FieldPath,
				Key:         "ref",
				Label:       "Reference Input",
				Description: "Folder or image file",
				Placeholder: "./refs or ./image.png",
				Default:     "",
				DirsOnly:    false,
				AllowedExts: imageExts,
			},
			OutputFolderField,
			NumImagesField,
			{
				Type:        FieldTextArea,
				Key:         "prompt",
				Label:       "Prompt",
				Description: "Editing instructions for the image",
				Placeholder: "A 2D vector art pattern inspired by the reference...",
				Lines:       3,
			},
			AspectRatioField,
			ImageSizeField,
			GroundingField,
		},
	}
}

// BuildForm creates a Form from a FormConfig
func BuildForm(config FormConfig) *Form {
	form := NewForm(config.Title)

	for _, field := range config.Fields {
		switch field.Type {
		case FieldInput:
			form.AddInput(field.Key, field.Label, field.Description, field.Placeholder, field.Default)
		case FieldTextArea:
			form.AddTextArea(field.Key, field.Label, field.Description, field.Placeholder, field.Lines)
		case FieldSelect:
			form.AddSelect(field.Key, field.Label, field.Description, field.Options, field.DefaultIdx)
		case FieldToggle:
			form.AddToggle(field.Key, field.Label, field.Description, field.BoolDefault)
		case FieldPath:
			form.AddPath(field.Key, field.Label, field.Description, field.Placeholder, field.Default, field.DirsOnly, field.AllowedExts)
		}
	}

	return form
}
