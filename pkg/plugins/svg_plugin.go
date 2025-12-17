package plugins

import (
	"context"
	"fmt"
	"strings"
)

// === SVG Drawing Plugin ===

type SVGPlugin struct {
	name    string
	version string
}

func NewSVGPlugin() *SVGPlugin {
	return &SVGPlugin{
		name:    "svg",
		version: "1.0.0",
	}
}

func (sp *SVGPlugin) Name() string {
	return sp.name
}

func (sp *SVGPlugin) Version() string {
	return sp.version
}

func (sp *SVGPlugin) Initialize(config map[string]interface{}) error {
	// Store SVG config if needed
	return nil
}

func (sp *SVGPlugin) Validate() error {
	// No external dependencies required
	return nil
}

func (sp *SVGPlugin) Execute(ctx context.Context, action string, payload interface{}) (interface{}, error) {
	switch action {
	case "create":
		return sp.createSVG(ctx, payload)
	case "update":
		return sp.updateSVG(ctx, payload)
	case "export":
		return sp.exportSVG(ctx, payload)
	case "import":
		return sp.importSVG(ctx, payload)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (sp *SVGPlugin) Shutdown() error {
	return nil
}

// Actions

type SVGCreateRequest struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Name   string `json:"name"`
}

func (sp *SVGPlugin) createSVG(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	width := int(req["width"].(float64))
	height := int(req["height"].(float64))
	name := req["name"].(string)

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">
  <title>%s</title>
  <rect width="100%%" height="100%%" fill="white"/>
  <!-- SVG content goes here -->
</svg>`, width, height, width, height, name)

	return map[string]interface{}{
		"svg":  svg,
		"name": name,
		"type": "canvas",
	}, nil
}

type SVGUpdateRequest struct {
	SVG     string `json:"svg"`
	Changes string `json:"changes"` // JSON string of changes
}

func (sp *SVGPlugin) updateSVG(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	svg := req["svg"].(string)
	changes := req["changes"].(string)

	// Simple update - in production this would parse and apply changes
	updatedSVG := strings.Replace(svg, "<!-- SVG content goes here -->", changes, 1)

	return map[string]interface{}{
		"svg": updatedSVG,
	}, nil
}

type SVGExportRequest struct {
	SVG    string `json:"svg"`
	Format string `json:"format"` // svg, png, pdf
}

func (sp *SVGPlugin) exportSVG(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	svg := req["svg"].(string)
	format := req["format"].(string)

	switch format {
	case "svg":
		return map[string]interface{}{
			"data":     svg,
			"mimeType": "image/svg+xml",
			"filename": "drawing.svg",
		}, nil
	case "png":
		// In production, would convert SVG to PNG
		return map[string]interface{}{
			"data":     "PNG conversion not implemented",
			"mimeType": "image/png",
			"filename": "drawing.png",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

type SVGImportRequest struct {
	Data string `json:"data"`
	Type string `json:"type"` // url, file, string
}

func (sp *SVGPlugin) importSVG(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	data := req["data"].(string)
	dataType := req["type"].(string)

	switch dataType {
	case "string":
		// Validate it's SVG
		if !strings.Contains(data, "<svg") {
			return nil, fmt.Errorf("invalid SVG data")
		}
		return map[string]interface{}{
			"svg": data,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported import type: %s", dataType)
	}
}
