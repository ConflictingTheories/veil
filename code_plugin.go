package main

import (
	"context"
	"fmt"
	"strings"
)

// === Code Snippet Editor Plugin ===

type CodePlugin struct {
	name    string
	version string
}

func NewCodePlugin() *CodePlugin {
	return &CodePlugin{
		name:    "code",
		version: "1.0.0",
	}
}

func (cp *CodePlugin) Name() string {
	return cp.name
}

func (cp *CodePlugin) Version() string {
	return cp.version
}

func (cp *CodePlugin) Initialize(config map[string]interface{}) error {
	// Store code editor config if needed
	return nil
}

func (cp *CodePlugin) Validate() error {
	// No external dependencies required
	return nil
}

func (cp *CodePlugin) Execute(ctx context.Context, action string, payload interface{}) (interface{}, error) {
	switch action {
	case "create":
		return cp.createSnippet(ctx, payload)
	case "execute":
		return cp.executeCode(ctx, payload)
	case "format":
		return cp.formatCode(ctx, payload)
	case "highlight":
		return cp.highlightCode(ctx, payload)
	case "lint":
		return cp.lintCode(ctx, payload)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (cp *CodePlugin) Shutdown() error {
	return nil
}

// Actions

type CodeCreateRequest struct {
	Language string `json:"language"`
	Title    string `json:"title"`
	Content  string `json:"content"`
}

func (cp *CodePlugin) createSnippet(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	language := req["language"].(string)
	title := req["title"].(string)
	content := req["content"].(string)

	// Create code snippet with syntax highlighting wrapper
	snippet := fmt.Sprintf("```%s\n%s\n```", language, content)
	// Create code snippet with syntax highlighting wrapper

	return map[string]interface{}{
		"snippet":  snippet,
		"language": language,
		"title":    title,
		"type":     "code-snippet",
	}, nil
}

type CodeExecuteRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
	Input    string `json:"input,omitempty"`
}

func (cp *CodePlugin) executeCode(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	code := req["code"].(string)
	language := req["language"].(string)
	input := ""
	if val, ok := req["input"]; ok {
		input = val.(string)
	}

	// Execute code based on language
	switch strings.ToLower(language) {
	case "javascript", "js":
		return cp.executeJavaScript(code, input)
	case "python", "py":
		return cp.executePython(code, input)
	case "go":
		return cp.executeGo(code, input)
	case "bash", "shell":
		return cp.executeBash(code, input)
	default:
		return map[string]interface{}{
			"output":   "Language not supported for execution",
			"language": language,
			"status":   "error",
		}, nil
	}
}

func (cp *CodePlugin) executeJavaScript(code, input string) (interface{}, error) {
	// In production, would use a JavaScript runtime like Otto or V8
	return map[string]interface{}{
		"output":   "JavaScript execution not implemented (would use runtime)",
		"language": "javascript",
		"status":   "mock",
	}, nil
}

func (cp *CodePlugin) executePython(code, input string) (interface{}, error) {
	// In production, would use Python interpreter
	return map[string]interface{}{
		"output":   "Python execution not implemented (would use interpreter)",
		"language": "python",
		"status":   "mock",
	}, nil
}

func (cp *CodePlugin) executeGo(code, input string) (interface{}, error) {
	// In production, would compile and run Go code
	return map[string]interface{}{
		"output":   "Go execution not implemented (would compile and run)",
		"language": "go",
		"status":   "mock",
	}, nil
}

func (cp *CodePlugin) executeBash(code, input string) (interface{}, error) {
	// In production, would execute in shell
	return map[string]interface{}{
		"output":   "Bash execution not implemented (would run in shell)",
		"language": "bash",
		"status":   "mock",
	}, nil
}

type CodeFormatRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

func (cp *CodePlugin) formatCode(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	code := req["code"].(string)
	language := req["language"].(string)

	// Basic formatting - in production would use language-specific formatters
	formatted := cp.basicFormat(code, language)

	return map[string]interface{}{
		"formatted": formatted,
		"language":  language,
	}, nil
}

func (cp *CodePlugin) basicFormat(code, language string) string {
	// Simple indentation and spacing fixes
	lines := strings.Split(code, "\n")
	var formatted []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			formatted = append(formatted, trimmed)
		}
	}

	return strings.Join(formatted, "\n")
}

type CodeHighlightRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

func (cp *CodePlugin) highlightCode(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	code := req["code"].(string)
	language := req["language"].(string)

	// Basic syntax highlighting - in production would use a proper highlighter
	highlighted := cp.basicHighlight(code, language)

	return map[string]interface{}{
		"highlighted": highlighted,
		"language":    language,
	}, nil
}

func (cp *CodePlugin) basicHighlight(code, language string) string {
	// Simple keyword highlighting for demonstration
	keywords := map[string][]string{
		"go":         {"func", "package", "import", "var", "const", "if", "else", "for", "return", "type", "struct"},
		"javascript": {"function", "var", "let", "const", "if", "else", "for", "return", "class"},
		"python":     {"def", "class", "if", "else", "for", "return", "import", "from"},
	}

	if kw, ok := keywords[strings.ToLower(language)]; ok {
		for _, keyword := range kw {
			code = strings.ReplaceAll(code, keyword, fmt.Sprintf(`<span class="keyword">%s</span>`, keyword))
		}
	}

	return fmt.Sprintf(`<pre class="code-block language-%s"><code>%s</code></pre>`, language, code)
}

type CodeLintRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

func (cp *CodePlugin) lintCode(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	code := req["code"].(string)
	language := req["language"].(string)

	// Basic linting - in production would use language-specific linters
	issues := cp.basicLint(code, language)

	return map[string]interface{}{
		"issues":   issues,
		"language": language,
	}, nil
}

func (cp *CodePlugin) basicLint(code, language string) []map[string]interface{} {
	var issues []map[string]interface{}

	lines := strings.Split(code, "\n")
	for i, line := range lines {
		// Check for trailing whitespace
		if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
			issues = append(issues, map[string]interface{}{
				"type":     "warning",
				"message":  "Trailing whitespace",
				"line":     i + 1,
				"severity": "low",
			})
		}

		// Check for long lines
		if len(line) > 100 {
			issues = append(issues, map[string]interface{}{
				"type":     "info",
				"message":  "Line too long",
				"line":     i + 1,
				"severity": "low",
			})
		}
	}

	return issues
}
