package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// MCP Protocol types
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// Server state
type Server struct {
	browser *rod.Browser
	page    *rod.Page
}

func main() {
	server := &Server{}
	defer server.cleanup()

	// Read requests from stdin
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var req MCPRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		resp := server.handleRequest(req)
		if err := encoder.Encode(resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding response: %v\n", err)
		}
	}
}

func (s *Server) handleRequest(req MCPRequest) MCPResponse {
	switch req.Method {
	case "initialize":
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]bool{},
				},
				"serverInfo": map[string]string{
					"name":    "rod-mcp-server",
					"version": "1.0.0",
				},
			},
		}

	case "tools/list":
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": s.getTools(),
			},
		}

	case "tools/call":
		return s.handleToolCall(req)

	default:
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Method not found: " + req.Method,
			},
		}
	}
}

func (s *Server) getTools() []Tool {
	return []Tool{
		{
			Name:        "rod_navigate",
			Description: "Navigate to a URL in the browser",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The URL to navigate to",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        "rod_click",
			Description: "Click an element by CSS selector",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element to click",
					},
				},
				"required": []string{"selector"},
			},
		},
		{
			Name:        "rod_screenshot",
			Description: "Take a screenshot of the current page",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filename": map[string]interface{}{
						"type":        "string",
						"description": "Optional filename for the screenshot (default: timestamp)",
					},
					"fullPage": map[string]interface{}{
						"type":        "boolean",
						"description": "Capture full page or just viewport (default: false)",
					},
				},
			},
		},
		{
			Name:        "rod_get_attribute",
			Description: "Get an HTML attribute value from an element (perfect for HTMX-R state)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element",
					},
					"attribute": map[string]interface{}{
						"type":        "string",
						"description": "Attribute name to read (e.g., 'data-state-loading')",
					},
				},
				"required": []string{"selector", "attribute"},
			},
		},
		{
			Name:        "rod_get_text",
			Description: "Get the text content of an element",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element",
					},
				},
				"required": []string{"selector"},
			},
		},
		{
			Name:        "rod_wait_for",
			Description: "Wait for an element to appear",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element to wait for",
					},
					"timeout": map[string]interface{}{
						"type":        "number",
						"description": "Timeout in seconds (default: 30)",
					},
				},
				"required": []string{"selector"},
			},
		},
		{
			Name:        "rod_eval",
			Description: "Execute JavaScript in the page context",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"script": map[string]interface{}{
						"type":        "string",
						"description": "JavaScript code to execute",
					},
				},
				"required": []string{"script"},
			},
		},
		{
			Name:        "rod_fill",
			Description: "Fill an input field with text",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the input element",
					},
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Text to fill into the input",
					},
				},
				"required": []string{"selector", "text"},
			},
		},
	}
}

func (s *Server) handleToolCall(req MCPRequest) MCPResponse {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params: " + err.Error(),
			},
		}
	}

	// Ensure browser is initialized
	if s.browser == nil {
		if err := s.initBrowser(); err != nil {
			return MCPResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &MCPError{
					Code:    -32603,
					Message: "Failed to initialize browser: " + err.Error(),
				},
			}
		}
	}

	var result interface{}
	var err error

	switch params.Name {
	case "rod_navigate":
		result, err = s.navigate(params.Arguments)
	case "rod_click":
		result, err = s.click(params.Arguments)
	case "rod_screenshot":
		result, err = s.screenshot(params.Arguments)
	case "rod_get_attribute":
		result, err = s.getAttribute(params.Arguments)
	case "rod_get_text":
		result, err = s.getText(params.Arguments)
	case "rod_wait_for":
		result, err = s.waitFor(params.Arguments)
	case "rod_eval":
		result, err = s.eval(params.Arguments)
	case "rod_fill":
		result, err = s.fill(params.Arguments)
	default:
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Unknown tool: " + params.Name,
			},
		}
	}

	if err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32603,
				Message: err.Error(),
			},
		}
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("%v", result),
				},
			},
		},
	}
}

func (s *Server) initBrowser() error {
	path, _ := launcher.LookPath()
	u := launcher.New().Bin(path).MustLaunch()
	s.browser = rod.New().ControlURL(u).MustConnect()
	s.page = s.browser.MustPage()
	return nil
}

func (s *Server) navigate(args map[string]interface{}) (interface{}, error) {
	url, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url must be a string")
	}

	if err := s.page.Navigate(url); err != nil {
		return nil, err
	}

	if err := s.page.WaitLoad(); err != nil {
		return nil, err
	}

	return fmt.Sprintf("Successfully navigated to %s", url), nil
}

func (s *Server) click(args map[string]interface{}) (interface{}, error) {
	selector, ok := args["selector"].(string)
	if !ok {
		return nil, fmt.Errorf("selector must be a string")
	}

	elem, err := s.page.Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	if err := elem.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, err
	}

	return fmt.Sprintf("Successfully clicked %s", selector), nil
}

func (s *Server) screenshot(args map[string]interface{}) (interface{}, error) {
	filename, ok := args["filename"].(string)
	if !ok || filename == "" {
		filename = fmt.Sprintf("screenshot_%d.png", time.Now().Unix())
	}

	fullPage := false
	if fp, ok := args["fullPage"].(bool); ok {
		fullPage = fp
	}

	// Create screenshots directory
	screenshotDir := filepath.Join(os.TempDir(), "rod-screenshots")
	os.MkdirAll(screenshotDir, 0755)

	path := filepath.Join(screenshotDir, filename)

	// Save screenshot
	data, err := s.page.Screenshot(fullPage, nil)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, err
	}

	return fmt.Sprintf("Screenshot saved to %s", path), nil
}

func (s *Server) getAttribute(args map[string]interface{}) (interface{}, error) {
	selector, ok := args["selector"].(string)
	if !ok {
		return nil, fmt.Errorf("selector must be a string")
	}

	attribute, ok := args["attribute"].(string)
	if !ok {
		return nil, fmt.Errorf("attribute must be a string")
	}

	elem, err := s.page.Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	value, err := elem.Attribute(attribute)
	if err != nil {
		return nil, err
	}

	if value == nil {
		return fmt.Sprintf("Attribute '%s' not found on %s", attribute, selector), nil
	}

	return fmt.Sprintf("Attribute '%s' on %s = '%s'", attribute, selector, *value), nil
}

func (s *Server) getText(args map[string]interface{}) (interface{}, error) {
	selector, ok := args["selector"].(string)
	if !ok {
		return nil, fmt.Errorf("selector must be a string")
	}

	elem, err := s.page.Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	text, err := elem.Text()
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("Text content of %s: '%s'", selector, text), nil
}

func (s *Server) waitFor(args map[string]interface{}) (interface{}, error) {
	selector, ok := args["selector"].(string)
	if !ok {
		return nil, fmt.Errorf("selector must be a string")
	}

	timeout := 30.0
	if t, ok := args["timeout"].(float64); ok {
		timeout = t
	}

	s.page.Timeout(time.Duration(timeout) * time.Second)
	defer s.page.Timeout(0)

	_, err := s.page.Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element %s did not appear within %v seconds", selector, timeout)
	}

	return fmt.Sprintf("Element %s appeared", selector), nil
}

func (s *Server) eval(args map[string]interface{}) (interface{}, error) {
	script, ok := args["script"].(string)
	if !ok {
		return nil, fmt.Errorf("script must be a string")
	}

	result, err := s.page.Eval(script)
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("JavaScript result: %v", result.Value), nil
}

func (s *Server) fill(args map[string]interface{}) (interface{}, error) {
	selector, ok := args["selector"].(string)
	if !ok {
		return nil, fmt.Errorf("selector must be a string")
	}

	text, ok := args["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text must be a string")
	}

	elem, err := s.page.Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	if err := elem.SelectAllText(); err != nil {
		return nil, err
	}

	if err := elem.Input(text); err != nil {
		return nil, err
	}

	return fmt.Sprintf("Filled %s with '%s'", selector, text), nil
}

func (s *Server) cleanup() {
	if s.page != nil {
		s.page.Close()
	}
	if s.browser != nil {
		s.browser.Close()
	}
}
