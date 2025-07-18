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

	// Extract tool name from file name (without .py extension)
	fileName := filepath.Base(filePath)
	toolName := strings.TrimSuffix(fileName, ".py")

	toolInfo := &ToolInfo{
		Name:         toolName,
		FilePath:     filePath,
		FunctionName: toolName,
		Config:       make(map[string]interface{}),
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

	// Look for the function definition (tools now use @mcp.tool() decorator)
	functionRegex := regexp.MustCompile(`^def\s+` + toolInfo.FunctionName + `\s*\(([^)]*)\)\s*(?:->\s*([^:]+))?\s*:`)

	var inFunction bool
	var functionDocstring string

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Check if we found the main function
		if matches := functionRegex.FindStringSubmatch(line); matches != nil {
			inFunction = true

			// Extract parameters
			if len(matches) > 1 && matches[1] != "" {
				params := d.parseParameters(matches[1])
				toolInfo.Parameters = params
			}

			// Extract return type
			if len(matches) > 2 && matches[2] != "" {
				toolInfo.ReturnType = strings.TrimSpace(matches[2])
			}

			// Check if function is async
			if strings.Contains(lines[i], "async def") {
				toolInfo.IsAsync = true
			}

			continue
		}

		// Extract docstring if we're in the function
		if inFunction && functionDocstring == "" {
			if strings.Contains(line, `"""`) || strings.Contains(line, `'''`) {
				// Extract docstring
				docstring := d.extractDocstring(lines, i)
				if docstring != "" {
					toolInfo.Description = docstring
					functionDocstring = docstring
				}
			}
		}

		// Stop parsing once we exit the function
		if inFunction && line != "" &&
			!strings.HasPrefix(line, " ") &&
			!strings.HasPrefix(line, "\t") &&
			!strings.Contains(line, `"""`) &&
			!strings.Contains(line, `'''`) {
			break
		}
	}

	// If no function found, this is an error for our dynamic loading approach
	if !inFunction {
		return fmt.Errorf("no function named '%s' found in file", toolInfo.FunctionName)
	}

	return nil
}

// parseParameters extracts parameter information from function signature
func (d *Discovery) parseParameters(paramStr string) []ParameterInfo {
	// Simple parameter parsing - can be enhanced later
	parts := strings.Split(paramStr, ",")
	params := make([]ParameterInfo, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "self" {
			continue
		}

		param := ParameterInfo{
			Name:     part,
			Type:     "str", // Default type
			Required: true,
		}

		// Check for type annotations
		if strings.Contains(part, ":") {
			parts := strings.Split(part, ":")
			param.Name = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				param.Type = strings.TrimSpace(parts[1])
			}
		}

		// Check for default values
		if strings.Contains(param.Name, "=") {
			parts := strings.Split(param.Name, "=")
			param.Name = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				param.Default = strings.TrimSpace(parts[1])
				param.Required = false
			}
		}

		params = append(params, param)
	}

	return params
}

// extractDocstring extracts docstring from function
func (d *Discovery) extractDocstring(lines []string, startLine int) string {
	var docstring strings.Builder
	var inDocstring bool
	var quoteType string

	for i := startLine; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if !inDocstring {
			if strings.Contains(line, `"""`) {
				inDocstring = true
				quoteType = `"""`
				// Extract content after opening quotes
				if idx := strings.Index(line, `"""`); idx != -1 {
					content := line[idx+3:]
					if strings.Contains(content, `"""`) {
						// Single line docstring
						content = strings.TrimSuffix(content, `"""`)
						return strings.TrimSpace(content)
					}
					if content != "" {
						docstring.WriteString(content)
					}
				}
			}
			if strings.Contains(line, `'''`) {
				inDocstring = true
				quoteType = `'''`
				// Extract content after opening quotes
				if idx := strings.Index(line, `'''`); idx != -1 {
					content := line[idx+3:]
					if strings.Contains(content, `'''`) {
						// Single line docstring
						content = strings.TrimSuffix(content, `'''`)
						return strings.TrimSpace(content)
					}
					if content != "" {
						docstring.WriteString(content)
					}
				}
			}
		} else {
			// We're inside a docstring
			if strings.Contains(line, quoteType) {
				// End of docstring
				content := strings.Split(line, quoteType)[0]
				if content != "" {
					docstring.WriteString(" " + content)
				}
				break
			}
			if docstring.Len() > 0 {
				docstring.WriteString(" ")
			}
			docstring.WriteString(line)
		}
	}

	return strings.TrimSpace(docstring.String())
}

// DiscoverTools discovers all tool files in the tools directory
func (d *Discovery) DiscoverTools(toolsDir string) ([]ToolInfo, error) {
	var tools []ToolInfo

	// Find all Python files in tools directory
	err := filepath.Walk(toolsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip __init__.py and other non-tool files
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".py") || info.Name() == "__init__.py" {
			return nil
		}

		// Analyze the tool file
		toolInfo, err := d.AnalyzeToolFile(path)
		if err != nil {
			return fmt.Errorf("failed to analyze tool file %s: %w", path, err)
		}

		tools = append(tools, *toolInfo)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to discover tools: %w", err)
	}

	return tools, nil
}
