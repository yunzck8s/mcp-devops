// client/pkg/chains/diagnose_pod.go
package chains

import (
	"context"
	"encoding/json"
	"errors" // Import errors package for checking specific errors if needed
	"fmt"
	"log"
	"strings"
	"sync" // Import sync for potential future caching improvements
	"time"

	"github.com/cloudwego/eino/components/tool"
	"mcp-devops/client/pkg/mcp"
)

// DiagnosePodChainInput defines the expected input for the chain.
type DiagnosePodChainInput struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"pod_name"`
}

// DiagnosePodChain orchestrates the pod diagnosis workflow.
type DiagnosePodChain struct {
	clientManager *mcp.ClientManager
	// Consider adding a simple cache for tools if needed
	// toolsCache     []tool.BaseTool
	// cacheMutex     sync.RWMutex
	// lastFetchTime  time.Time
	// cacheTTL       time.Duration
}

// NewDiagnosePodChain creates a new instance of the chain.
func NewDiagnosePodChain(cm *mcp.ClientManager) *DiagnosePodChain {
	if cm == nil {
		// Use log.Fatalf or return an error to prevent creating an invalid chain
		log.Fatalf("[DiagnosePodChain] ClientManager cannot be nil")
	}
	return &DiagnosePodChain{
		clientManager: cm,
		// cacheTTL: 5 * time.Minute, // Example cache TTL
	}
}

// Run executes the pod diagnosis chain.
func (c *DiagnosePodChain) Run(ctx context.Context, input *DiagnosePodChainInput) (string, error) {
	if input == nil || input.Namespace == "" || input.PodName == "" {
		return "", errors.New("invalid input: namespace and pod_name are required") // Use errors.New
	}

	// 1. Get available tools
	// Use a reasonable timeout for tool fetching within the chain execution
	// Consider if GetMCPTools should be called here or if tools should be passed in
	toolCtx, toolCancel := context.WithTimeout(ctx, 30*time.Second) // Slightly longer timeout for fetching
	defer toolCancel()

	// Fetch tools - might need forceRefresh=false depending on overall app logic
	tools, err := mcp.GetMCPTools(toolCtx, c.clientManager, false, false)
	if err != nil {
		// Use fmt.Errorf for wrapping errors
		return "", fmt.Errorf("failed to get MCP tools: %w", err)
	}
	if len(tools) == 0 {
		return "", errors.New("no MCP tools available")
	}

	// --- Find Required Tools ---
	// !! IMPORTANT: Adjust these tool names to match your actual MCP tool names !!
	describeToolName := "kubernetes_describe_pod"
	logsToolName := "kubernetes_get_pod_logs"

	var describeTool, logsTool tool.BaseTool
	// Find tools sequentially
	for _, t := range tools {
		// Use a short timeout for getting individual tool info
		infoCtx, infoCancel := context.WithTimeout(ctx, 5*time.Second)
		info, toolInfoErr := t.Info(infoCtx) // Assign error to a different variable
		infoCancel()                         // Cancel context immediately after use

		if toolInfoErr != nil {
			// Log as warning and continue trying to find tools
			log.Printf("[DiagnosePodChain] Warning: failed to get info for a tool: %v\n", toolInfoErr)
			continue
		}
		// Ensure info is not nil before accessing Name
		if info != nil {
			if info.Name == describeToolName {
				describeTool = t
			}
			if info.Name == logsToolName {
				logsTool = t
			}
		}

		// Optimization: break if both found
		if describeTool != nil && logsTool != nil {
			break
		}
	}

	// Check if tools were found *after* the loop
	if describeTool == nil {
		return "", fmt.Errorf("required tool '%s' not found", describeToolName)
	}
	if logsTool == nil {
		return "", fmt.Errorf("required tool '%s' not found", logsToolName)
	}

	// --- Execute Tools Sequentially ---
	var results strings.Builder // More efficient string building
	results.WriteString(fmt.Sprintf("Diagnosis results for pod '%s' in namespace '%s':\n\n", input.PodName, input.Namespace))

	// 2. Execute Describe Pod Tool
	describeInput := map[string]any{
		"namespace": input.Namespace,
		"name":      input.PodName,
		// Add other parameters if the tool requires them
	}
	fmt.Printf("[DiagnosePodChain] Calling tool: %s\n", describeToolName)
	// **** Explicit Check before calling ****
	if describeTool == nil {
		results.WriteString(fmt.Sprintf("--- Error: Describe tool '%s' became nil before call ---\n\n", describeToolName))
	} else {
		describeOutputStr, callErr := c.callTool(ctx, describeTool, describeInput) // Use different error var name
		if callErr != nil {
			// Standard error handling using callErr.Error()
			results.WriteString(fmt.Sprintf("--- Error getting pod description ---\n%s\n\n", callErr.Error()))
		} else {
			results.WriteString(fmt.Sprintf("--- Pod Description ---\n%s\n\n", describeOutputStr))
		}
	}

	// 3. Execute Get Logs Tool (e.g., last 50 lines)
	logsInput := map[string]any{
		"namespace":  input.Namespace,
		"name":       input.PodName,
		"tail_lines": 50, // Or make this configurable via input
		// Add other parameters like 'container' if needed
	}
	fmt.Printf("[DiagnosePodChain] Calling tool: %s\n", logsToolName)
	// **** Explicit Check before calling ****
	if logsTool == nil {
		results.WriteString(fmt.Sprintf("--- Error: Logs tool '%s' became nil before call ---\n\n", logsToolName))
	} else {
		logsOutputStr, callErr := c.callTool(ctx, logsTool, logsInput) // Use different error var name
		if callErr != nil {
			// Standard error handling using callErr.Error()
			results.WriteString(fmt.Sprintf("--- Error getting pod logs ---\n%s\n\n", callErr.Error()))
		} else {
			results.WriteString(fmt.Sprintf("--- Pod Logs (last 50 lines) ---\n%s\n\n", logsOutputStr))
		}
	}

	results.WriteString("\n--- End of Diagnosis ---")

	return results.String(), nil // Return nil error on success
}

// callTool is a helper to call a tool with a specific timeout and handle output.
// Input tool 't' should be guaranteed non-nil by the caller.
func (c *DiagnosePodChain) callTool(ctx context.Context, t tool.BaseTool, input map[string]any) (string, error) {
	// Add a check just in case, although caller should ensure t is not nil
	if t == nil {
		return "", errors.New("callTool received a nil tool")
	}

	callCtx, callCancel := context.WithTimeout(ctx, 45*time.Second) // Timeout for individual tool call
	defer callCancel()

	// Refresh session before critical calls might be excessive here,
	// consider if needed based on session duration and chain length.
	// c.clientManager.RefreshSession()

	// This is the call that might have caused "Unresolved reference 'Call'" if 't' was nil or not a BaseTool
	output, err := t.Call(callCtx, input)
	if err != nil {
		// Check for connection errors and mark if necessary
		errMsg := err.Error() // Standard way to get error string
		if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "Invalid session ID") {
			// Mark failure non-blockingly if possible, or log it clearly
			go c.clientManager.MarkConnectionFailed(err) // Run in goroutine to avoid blocking chain
			log.Printf("[DiagnosePodChain] Marked connection as failed due to tool call error: %v", err)
		}
		return "", fmt.Errorf("tool call failed: %w", err) // Wrap error
	}

	// Attempt to format output as JSON, fallback to string
	outputBytes, jsonErr := json.MarshalIndent(output, "", "  ")
	if jsonErr != nil {
		// Fallback to simple string conversion if JSON marshalling fails
		return fmt.Sprintf("%v", output), nil
	}
	return string(outputBytes), nil
}
