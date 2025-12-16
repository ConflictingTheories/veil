package main

import (
	"context"
	"fmt"
	"strings"
)

// === Shader Demo Editor Plugin ===

type ShaderPlugin struct {
	name    string
	version string
}

func NewShaderPlugin() *ShaderPlugin {
	return &ShaderPlugin{
		name:    "shader",
		version: "1.0.0",
	}
}

func (sp *ShaderPlugin) Name() string {
	return sp.name
}

func (sp *ShaderPlugin) Version() string {
	return sp.version
}

func (sp *ShaderPlugin) Initialize(config map[string]interface{}) error {
	// Store shader editor config if needed
	return nil
}

func (sp *ShaderPlugin) Validate() error {
	// No external dependencies required
	return nil
}

func (sp *ShaderPlugin) Execute(ctx context.Context, action string, payload interface{}) (interface{}, error) {
	switch action {
	case "create":
		return sp.createShader(ctx, payload)
	case "compile":
		return sp.compileShader(ctx, payload)
	case "preview":
		return sp.previewShader(ctx, payload)
	case "export":
		return sp.exportShader(ctx, payload)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (sp *ShaderPlugin) Shutdown() error {
	return nil
}

// Actions

type ShaderCreateRequest struct {
	Type   string `json:"type"` // vertex, fragment
	Name   string `json:"name"`
	Shader string `json:"shader,omitempty"`
}

func (sp *ShaderPlugin) createShader(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	shaderType := req["type"].(string)
	name := req["name"].(string)
	shader := ""
	if val, ok := req["shader"]; ok {
		shader = val.(string)
	}

	// Create default shader template based on type
	if shader == "" {
		switch strings.ToLower(shaderType) {
		case "vertex":
			shader = sp.getDefaultVertexShader()
		case "fragment":
			shader = sp.getDefaultFragmentShader()
		default:
			return nil, fmt.Errorf("unsupported shader type: %s", shaderType)
		}
	}

	html := sp.generateShaderHTML(name, shaderType, shader)

	return map[string]interface{}{
		"html":   html,
		"shader": shader,
		"type":   shaderType,
		"name":   name,
	}, nil
}

func (sp *ShaderPlugin) getDefaultVertexShader() string {
	return `
attribute vec3 position;
attribute vec2 uv;

uniform mat4 modelViewMatrix;
uniform mat4 projectionMatrix;

varying vec2 vUv;

void main() {
    vUv = uv;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
`
}

func (sp *ShaderPlugin) getDefaultFragmentShader() string {
	return `
precision mediump float;

uniform float time;
uniform vec2 resolution;

varying vec2 vUv;

void main() {
    vec2 uv = vUv;
    vec3 color = vec3(uv.x, uv.y, sin(time) * 0.5 + 0.5);
    gl_FragColor = vec4(color, 1.0);
}
`
}

func (sp *ShaderPlugin) generateShaderHTML(name, shaderType, shader string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - %s Shader</title>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/three.js/r128/three.min.js"></script>
    <style>
        body { margin: 0; overflow: hidden; }
        #shaderCanvas { width: 100vw; height: 100vh; display: block; }
        #controls { position: absolute; top: 10px; left: 10px; z-index: 100; }
        #controls button { margin: 5px; padding: 8px 16px; }
    </style>
</head>
<body>
    <div id="controls">
        <button onclick="toggleAnimation()">Play/Pause</button>
        <button onclick="resetTime()">Reset</button>
    </div>
    <canvas id="shaderCanvas"></canvas>

    <script>
        let scene, camera, renderer, material, mesh;
        let time = 0;
        let isPlaying = true;
        let clock = new THREE.Clock();

        init();
        animate();

        function init() {
            // Scene setup
            scene = new THREE.Scene();
            camera = new THREE.PerspectiveCamera(75, window.innerWidth / window.innerHeight, 0.1, 1000);
            camera.position.z = 5;

            renderer = new THREE.WebGLRenderer({ canvas: document.getElementById('shaderCanvas') });
            renderer.setSize(window.innerWidth, window.innerHeight);

            // Shader material
            const vertexShader = %s;
            const fragmentShader = %s;

            material = new THREE.ShaderMaterial({
                vertexShader: vertexShader,
                fragmentShader: fragmentShader,
                uniforms: {
                    time: { value: 0 },
                    resolution: { value: new THREE.Vector2(window.innerWidth, window.innerHeight) }
                }
            });

            // Create plane geometry
            const geometry = new THREE.PlaneGeometry(10, 10);
            mesh = new THREE.Mesh(geometry, material);
            scene.add(mesh);

            window.addEventListener('resize', onWindowResize);
        }

        function onWindowResize() {
            camera.aspect = window.innerWidth / window.innerHeight;
            camera.updateProjectionMatrix();
            renderer.setSize(window.innerWidth, window.innerHeight);
            material.uniforms.resolution.value.set(window.innerWidth, window.innerHeight);
        }

        function animate() {
            requestAnimationFrame(animate);

            if (isPlaying) {
                time += clock.getDelta();
                material.uniforms.time.value = time;
            }

            renderer.render(scene, camera);
        }

        function toggleAnimation() {
            isPlaying = !isPlaying;
            if (isPlaying) {
                clock.start();
            } else {
                clock.stop();
            }
        }

        function resetTime() {
            time = 0;
            material.uniforms.time.value = time;
        }
    </script>
</body>
</html>`, name, shaderType, "`"+sp.getDefaultVertexShader()+"`", "`"+shader+"`")
}

type ShaderCompileRequest struct {
	VertexShader   string `json:"vertexShader"`
	FragmentShader string `json:"fragmentShader"`
}

func (sp *ShaderPlugin) compileShader(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	vertexShader := req["vertexShader"].(string)
	fragmentShader := req["fragmentShader"].(string)

	// Basic validation - in production would use WebGL context to validate
	errors := sp.validateShader(vertexShader, fragmentShader)

	return map[string]interface{}{
		"valid":  len(errors) == 0,
		"errors": errors,
	}, nil
}

func (sp *ShaderPlugin) validateShader(vertex, fragment string) []map[string]interface{} {
	var errors []map[string]interface{}

	// Basic syntax checks
	if !strings.Contains(vertex, "void main()") {
		errors = append(errors, map[string]interface{}{
			"type":    "error",
			"message": "Vertex shader missing main function",
			"shader":  "vertex",
		})
	}

	if !strings.Contains(fragment, "void main()") {
		errors = append(errors, map[string]interface{}{
			"type":    "error",
			"message": "Fragment shader missing main function",
			"shader":  "fragment",
		})
	}

	// Check for basic syntax issues
	if strings.Contains(fragment, "gl_FragColor") && !strings.Contains(fragment, "precision") {
		errors = append(errors, map[string]interface{}{
			"type":    "warning",
			"message": "Fragment shader should specify precision",
			"shader":  "fragment",
		})
	}

	return errors
}

type ShaderPreviewRequest struct {
	VertexShader   string `json:"vertexShader"`
	FragmentShader string `json:"fragmentShader"`
	Name           string `json:"name"`
}

func (sp *ShaderPlugin) previewShader(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	// vertexShader := req["vertexShader"].(string) // Not currently used
	fragmentShader := req["fragmentShader"].(string)
	name := req["name"].(string)

	html := sp.generateShaderHTML(name, "fragment", fragmentShader)

	return map[string]interface{}{
		"html": html,
		"name": name,
	}, nil
}

type ShaderExportRequest struct {
	VertexShader   string `json:"vertexShader"`
	FragmentShader string `json:"fragmentShader"`
	Format         string `json:"format"` // html, json, glsl
}

func (sp *ShaderPlugin) exportShader(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	fragmentShader := req["fragmentShader"].(string)
	format := req["format"].(string)

	switch format {
	case "html":
		html := sp.generateShaderHTML("Exported Shader", "fragment", fragmentShader)
		return map[string]interface{}{
			"data":     html,
			"mimeType": "text/html",
			"filename": "shader.html",
		}, nil
	case "json":
		vertexShader, _ := req["vertexShader"].(string)
		shaderData := map[string]interface{}{
			"vertexShader":   vertexShader,
			"fragmentShader": fragmentShader,
		}
		return map[string]interface{}{
			"data":     shaderData,
			"mimeType": "application/json",
			"filename": "shader.json",
		}, nil
	case "glsl":
		vertexShader, _ := req["vertexShader"].(string)
		glsl := fmt.Sprintf("// Vertex Shader\n%s\n\n// Fragment Shader\n%s", vertexShader, fragmentShader)
		return map[string]interface{}{
			"data":     glsl,
			"mimeType": "text/plain",
			"filename": "shader.glsl",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}
