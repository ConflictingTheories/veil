package plugins

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"veil/pkg/codex"
)

// TerminalScriptingPlugin handles shell script execution, package installation, and code generation
type TerminalScriptingPlugin struct {
	name            string
	version         string
	allowedCommands map[string]bool
	safeMode        bool
	repo            *codex.Repository
}

// NewTerminalScriptingPlugin creates a new terminal scripting plugin
func NewTerminalScriptingPlugin() *TerminalScriptingPlugin {
	return &TerminalScriptingPlugin{
		name:    "terminal",
		version: "1.0.0",
		allowedCommands: map[string]bool{
			"npm":    true,
			"yarn":   true,
			"pip":    true,
			"pip3":   true,
			"go":     true,
			"cargo":  true,
			"git":    true,
			"curl":   true,
			"wget":   true,
			"ls":     true,
			"pwd":    true,
			"echo":   true,
			"cat":    true,
			"head":   true,
			"tail":   true,
			"grep":   true,
			"find":   true,
			"which":  true,
			"mkdir":  true,
			"touch":  true,
			"cp":     true,
			"mv":     true,
			"rm":     true,
			"chmod":  true,
			"chown":  true,
			"ps":     true,
			"top":    true,
			"df":     true,
			"du":     true,
			"uname":  true,
			"whoami": true,
			"id":     true,
		},
		safeMode: true,
	}
}

// Initialize sets up the plugin with configuration
func (tsp *TerminalScriptingPlugin) Initialize(config map[string]interface{}) error {
	if safeMode, ok := config["safe_mode"].(bool); ok {
		tsp.safeMode = safeMode
	}

	if allowedCmds, ok := config["allowed_commands"].([]interface{}); ok {
		tsp.allowedCommands = make(map[string]bool)
		for _, cmd := range allowedCmds {
			if cmdStr, ok := cmd.(string); ok {
				tsp.allowedCommands[cmdStr] = true
			}
		}
	}

	log.Printf("Terminal scripting plugin initialized (safe mode: %v)", tsp.safeMode)
	return nil
}

// Name returns the plugin name
func (tsp *TerminalScriptingPlugin) Name() string {
	return "terminal"
}

// Version returns the plugin version
func (tsp *TerminalScriptingPlugin) Version() string {
	return "1.0.0"
}

// Validate checks if the plugin is properly configured
func (tsp *TerminalScriptingPlugin) Validate() error {
	if tsp.allowedCommands == nil {
		return fmt.Errorf("allowed commands not initialized")
	}
	return nil
}

// Shutdown cleans up plugin resources
func (tsp *TerminalScriptingPlugin) Shutdown() error {
	log.Printf("Terminal scripting plugin shutting down")
	return nil
}

// AttachRepository implements RepositoryAware to receive codex repository
func (tsp *TerminalScriptingPlugin) AttachRepository(r *codex.Repository) error {
	tsp.repo = r
	return nil
}

// Execute handles different terminal scripting actions
func (tsp *TerminalScriptingPlugin) Execute(ctx context.Context, action string, payload interface{}) (interface{}, error) {
	switch action {
	case "execute":
		return tsp.executeScript(ctx, payload)
	case "install_package":
		return tsp.installPackage(ctx, payload)
	case "generate_code":
		return tsp.generateCode(ctx, payload)
	case "run_tests":
		return tsp.runTests(ctx, payload)
	case "build_project":
		return tsp.buildProject(ctx, payload)
	case "check_dependencies":
		return tsp.checkDependencies(ctx, payload)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// executeScript runs a shell script with safety checks
func (tsp *TerminalScriptingPlugin) executeScript(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	command, ok := req["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command is required")
	}

	// Safety checks
	if tsp.safeMode {
		if err := tsp.validateCommand(command); err != nil {
			return nil, fmt.Errorf("command validation failed: %v", err)
		}
	}

	// Parse command and arguments
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	// Set working directory if specified
	if wd, ok := req["working_directory"].(string); ok {
		if tsp.safeMode {
			if !filepath.IsAbs(wd) {
				return nil, fmt.Errorf("working directory must be absolute path in safe mode")
			}
		}
		cmd.Dir = wd
	}

	// Set environment variables if specified
	if env, ok := req["environment"].(map[string]interface{}); ok {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%v", k, v))
		}
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	result := map[string]interface{}{
		"command": command,
		"output":  string(output),
		"success": err == nil,
	}

	if err != nil {
		result["error"] = err.Error()
		result["exit_code"] = cmd.ProcessState.ExitCode()
	}

	return result, nil
}

// validateCommand checks if a command is safe to execute
func (tsp *TerminalScriptingPlugin) validateCommand(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	baseCmd := parts[0]

	// Check if command is in allowed list
	if !tsp.allowedCommands[baseCmd] {
		return fmt.Errorf("command '%s' is not in allowed commands list", baseCmd)
	}

	// Additional safety checks
	dangerousPatterns := []string{
		`rm\s+(-rf|--force|--recursive)\s+/`,
		`>\s*/`,
		`sudo`,
		`su`,
		`passwd`,
		`chmod\s+777`,
		`dd\s+if=`,
		`mkfs`,
		`fdisk`,
		`format`,
		`shutdown`,
		`reboot`,
		`halt`,
		`poweroff`,
		`kill\s+-9`,
		`pkill\s+-9`,
		`killall`,
	}

	for _, pattern := range dangerousPatterns {
		if matched, _ := regexp.MatchString(pattern, command); matched {
			return fmt.Errorf("command contains dangerous pattern: %s", pattern)
		}
	}

	return nil
}

// installPackage installs packages using various package managers
func (tsp *TerminalScriptingPlugin) installPackage(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	packageManager, ok := req["manager"].(string)
	if !ok {
		return nil, fmt.Errorf("package manager is required")
	}

	packages, ok := req["packages"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("packages list is required")
	}

	var packageNames []string
	for _, pkg := range packages {
		if pkgStr, ok := pkg.(string); ok {
			packageNames = append(packageNames, pkgStr)
		}
	}

	var command string
	switch packageManager {
	case "npm":
		command = fmt.Sprintf("npm install %s", strings.Join(packageNames, " "))
	case "yarn":
		command = fmt.Sprintf("yarn add %s", strings.Join(packageNames, " "))
	case "pip":
		command = fmt.Sprintf("pip install %s", strings.Join(packageNames, " "))
	case "pip3":
		command = fmt.Sprintf("pip3 install %s", strings.Join(packageNames, " "))
	case "go":
		command = fmt.Sprintf("go get %s", strings.Join(packageNames, " "))
	case "cargo":
		command = fmt.Sprintf("cargo add %s", strings.Join(packageNames, " "))
	default:
		return nil, fmt.Errorf("unsupported package manager: %s", packageManager)
	}

	return tsp.executeScript(ctx, map[string]interface{}{
		"command": command,
	})
}

// generateCode generates boilerplate code for various languages/frameworks
func (tsp *TerminalScriptingPlugin) generateCode(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	template, ok := req["template"].(string)
	if !ok {
		return nil, fmt.Errorf("template is required")
	}

	name, ok := req["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}

	// Generate code based on template
	var code string
	var files map[string]string

	switch template {
	case "react-component":
		code = tsp.generateReactComponent(name)
		files = map[string]string{
			fmt.Sprintf("%s.jsx", name): code,
		}
	case "go-web-server":
		code = tsp.generateGoWebServer(name)
		files = map[string]string{
			"main.go": code,
			"go.mod":  fmt.Sprintf("module %s\n\ngo 1.21\n", name),
		}
	case "python-flask-app":
		code = tsp.generatePythonFlaskApp(name)
		files = map[string]string{
			"app.py":           code,
			"requirements.txt": "Flask==2.3.3\n",
		}
	case "rust-binary":
		code = tsp.generateRustBinary(name)
		files = map[string]string{
			"src/main.rs": code,
			"Cargo.toml": fmt.Sprintf(`[package]
name = "%s"
version = "0.1.0"
edition = "2021"

[dependencies]
`, name),
		}
	default:
		return nil, fmt.Errorf("unknown template: %s", template)
	}

	// Create files if directory is specified
	if dir, ok := req["directory"].(string); ok {
		for filename, content := range files {
			filePath := filepath.Join(dir, filename)
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory: %v", err)
			}
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return nil, fmt.Errorf("failed to write file %s: %v", filename, err)
			}
		}
	}

	return map[string]interface{}{
		"template": template,
		"name":     name,
		"files":    files,
		"success":  true,
	}, nil
}

// runTests runs tests for various project types
func (tsp *TerminalScriptingPlugin) runTests(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	projectType, ok := req["type"].(string)
	if !ok {
		return nil, fmt.Errorf("project type is required")
	}

	var command string
	switch projectType {
	case "node":
		command = "npm test"
	case "python":
		command = "python -m pytest"
	case "go":
		command = "go test ./..."
	case "rust":
		command = "cargo test"
	default:
		return nil, fmt.Errorf("unsupported project type: %s", projectType)
	}

	return tsp.executeScript(ctx, map[string]interface{}{
		"command": command,
	})
}

// buildProject builds projects for various languages
func (tsp *TerminalScriptingPlugin) buildProject(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	projectType, ok := req["type"].(string)
	if !ok {
		return nil, fmt.Errorf("project type is required")
	}

	var command string
	switch projectType {
	case "node":
		command = "npm run build"
	case "go":
		command = "go build"
	case "rust":
		command = "cargo build --release"
	case "python":
		command = "python setup.py build"
	default:
		return nil, fmt.Errorf("unsupported project type: %s", projectType)
	}

	return tsp.executeScript(ctx, map[string]interface{}{
		"command": command,
	})
}

// checkDependencies checks if required dependencies are installed
func (tsp *TerminalScriptingPlugin) checkDependencies(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	dependencies, ok := req["dependencies"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("dependencies list is required")
	}

	results := make(map[string]interface{})

	for _, dep := range dependencies {
		if depStr, ok := dep.(string); ok {
			// Check if command exists
			_, err := exec.LookPath(depStr)
			results[depStr] = err == nil
		}
	}

	return map[string]interface{}{
		"dependencies": results,
		"timestamp":    time.Now().Unix(),
	}, nil
}

// Code generation templates

func (tsp *TerminalScriptingPlugin) generateReactComponent(name string) string {
	return fmt.Sprintf(`import React, { useState, useEffect } from 'react';

const %s = () => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Component logic here
    setLoading(false);
  }, []);

  if (loading) {
    return <div>Loading...</div>;
  }

  return (
    <div className="%s">
      <h2>%s Component</h2>
      {/* Component JSX here */}
    </div>
  );
};

export default %s;
`, name, strings.ToLower(name), name, name)
}

func (tsp *TerminalScriptingPlugin) generateGoWebServer(name string) string {
	return fmt.Sprintf(`package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from %s!")
	})

	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "{"status": "ok"}")
	})

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
`, name)
}

func (tsp *TerminalScriptingPlugin) generatePythonFlaskApp(name string) string {
	return fmt.Sprintf(`from flask import Flask, jsonify

app = Flask(__name__)

@app.route('/')
def hello():
    return f'Hello from {name}!'

@app.route('/api/health')
def health():
    return jsonify({'status': 'ok'})

if __name__ == '__main__':
    app.run(debug=True, host='0.0.0.0', port=5000)
`)
}

func (tsp *TerminalScriptingPlugin) generateRustBinary(name string) string {
	return fmt.Sprintf(`fn main() {
    println!("Hello from %s!");

    // Application logic here
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_main() {
        // Test logic here
        assert_eq!(2 + 2, 4);
    }
}
`, name)
}
