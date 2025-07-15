package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Discovery handles analyzing existing Python tool files
type Discovery struct {
	projectDir string
}

// NewDiscovery creates a new discovery instance
func NewDiscovery(projectDir string) *Discovery {
	return &Discovery{
		projectDir: projectDir,
	}
}

// AnalyzeToolFile analyzes a Python tool file and extracts tool information
func (d *Discovery) AnalyzeToolFile(filePath string) (*ToolInfo, error) {
	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	toolInfo := &ToolInfo{
		Name:     filepath.Base(filePath),
		FilePath: filePath,
		Methods:  []MethodInfo{},
		Imports:  []string{},
		Config:   make(map[string]interface{}),
	}

	// Parse the file content
	if err := d.parseFileContent(string(content), toolInfo); err != nil {
		return nil, fmt.Errorf("failed to parse file content: %w", err)
	}

	return toolInfo, nil
}

// parseFileContent parses Python file content and extracts tool information
func (d *Discovery) parseFileContent(content string, toolInfo *ToolInfo) error {
	lines := strings.Split(content, "\n")

	// Regular expressions for parsing Python code
	classRegex := regexp.MustCompile(`^class\s+(\w+).*:`)
	methodRegex := regexp.MustCompile(`^\s+(async\s+)?def\s+(\w+)\s*\(([^)]*)\)\s*(->\s*([^:]+))?\s*:`)
	importRegex := regexp.MustCompile(`^(from\s+\S+\s+)?import\s+(.+)`)

	var currentClass string
	var currentMethod *MethodInfo
	var inDocstring bool
	var docstringContent strings.Builder

	for i, line := range lines {
		line = strings.TrimRight(line, "\r\n")

		// Handle multi-line docstrings
		if inDocstring {
			if strings.Contains(line, `"""`) {
				inDocstring = false
				// Apply collected docstring to current method or class
				if currentMethod != nil {
					currentMethod.Description = strings.TrimSpace(docstringContent.String())
				}
				docstringContent.Reset()
			} else {
				docstringContent.WriteString(line)
				docstringContent.WriteString(" ")
			}
			continue
		}

		// Check for class definitions
		if classMatch := classRegex.FindStringSubmatch(line); classMatch != nil {
			currentClass = classMatch[1]
			toolInfo.ClassName = currentClass
			continue
		}

		// Check for method definitions
		if methodMatch := methodRegex.FindStringSubmatch(line); methodMatch != nil {
			isAsync := strings.TrimSpace(methodMatch[1]) == "async"
			methodName := methodMatch[2]
			paramsStr := methodMatch[3]
			returnType := strings.TrimSpace(methodMatch[5])

			// Skip private methods and __init__
			if strings.HasPrefix(methodName, "_") {
				continue
			}

			method := MethodInfo{
				Name:       methodName,
				IsAsync:    isAsync,
				ReturnType: returnType,
				Parameters: d.parseParameters(paramsStr),
				Config:     make(map[string]interface{}),
			}

			// Look for docstring on next lines
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(nextLine, `"""`) {
					if strings.HasSuffix(nextLine, `"""`) && len(nextLine) > 6 {
						// Single line docstring
						method.Description = strings.Trim(nextLine, `"`)
					} else {
						// Multi-line docstring
						inDocstring = true
						currentMethod = &method
						docstringContent.WriteString(strings.TrimPrefix(nextLine, `"""`))
					}
				}
			}

			toolInfo.Methods = append(toolInfo.Methods, method)
			continue
		}

		// Check for imports
		if importMatch := importRegex.FindStringSubmatch(line); importMatch != nil {
			toolInfo.Imports = append(toolInfo.Imports, strings.TrimSpace(line))
			continue
		}
	}

	return nil
}

// parseParameters parses method parameters from a parameter string
func (d *Discovery) parseParameters(paramsStr string) []ParameterInfo {
	if paramsStr == "" {
		return []ParameterInfo{}
	}

	params := []ParameterInfo{}

	// Split parameters by comma, but handle nested structures
	paramParts := d.splitParameters(paramsStr)

	for _, part := range paramParts {
		part = strings.TrimSpace(part)

		// Skip 'self' parameter
		if part == "self" {
			continue
		}

		param := d.parseParameter(part)
		if param.Name != "" {
			params = append(params, param)
		}
	}

	return params
}

// splitParameters splits parameter string by comma while respecting nested structures
func (d *Discovery) splitParameters(paramsStr string) []string {
	parts := []string{}
	current := strings.Builder{}
	depth := 0

	for _, char := range paramsStr {
		switch char {
		case '(', '[', '{':
			depth++
			current.WriteRune(char)
		case ')', ']', '}':
			depth--
			current.WriteRune(char)
		case ',':
			if depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// parseParameter parses a single parameter definition
func (d *Discovery) parseParameter(paramStr string) ParameterInfo {
	param := ParameterInfo{}

	// Handle default values
	if strings.Contains(paramStr, "=") {
		parts := strings.SplitN(paramStr, "=", 2)
		paramStr = strings.TrimSpace(parts[0])
		param.Default = strings.TrimSpace(parts[1])
		param.Required = false
	} else {
		param.Required = true
	}

	// Handle type annotations
	if strings.Contains(paramStr, ":") {
		parts := strings.SplitN(paramStr, ":", 2)
		param.Name = strings.TrimSpace(parts[0])
		param.Type = strings.TrimSpace(parts[1])
	} else {
		param.Name = strings.TrimSpace(paramStr)
		param.Type = "Any"
	}

	return param
}

// ScanDirectory scans a directory for Python tool files
func (d *Discovery) ScanDirectory(dir string) ([]ToolInfo, error) {
	toolsDir := filepath.Join(d.projectDir, dir)

	var tools []ToolInfo

	err := filepath.Walk(toolsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process Python files
		if !strings.HasSuffix(path, ".py") || strings.HasSuffix(path, "__init__.py") {
			return nil
		}

		// Analyze the tool file
		toolInfo, err := d.AnalyzeToolFile(path)
		if err != nil {
			return fmt.Errorf("failed to analyze %s: %w", path, err)
		}

		// Only include files that have at least one method
		if len(toolInfo.Methods) > 0 {
			tools = append(tools, *toolInfo)
		}

		return nil
	})

	return tools, err
}

// ListAvailableTools lists all available tools in the project
func (d *Discovery) ListAvailableTools() ([]ToolInfo, error) {
	return d.ScanDirectory("src/tools")
}
