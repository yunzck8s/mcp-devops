// client/pkg/chains/chain_tool_wrapper.go
package chains

import (
	"context"
	"encoding/json"
	"errors" // Import errors
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// DiagnosePodToolWrapper wraps DiagnosePodChain to be used as a tool.BaseTool
type DiagnosePodToolWrapper struct {
	chain *DiagnosePodChain
}

// NewDiagnosePodToolWrapper creates a new wrapper instance.
func NewDiagnosePodToolWrapper(chain *DiagnosePodChain) tool.BaseTool {
	if chain == nil {
		// Handle nil chain input gracefully
		// Perhaps return an error or a disabled tool implementation
		panic("Cannot create tool wrapper with a nil chain") // Or log.Fatal
	}
	return &DiagnosePodToolWrapper{chain: chain}
}

// Info provides metadata about the chain tool for the Agent.
func (w *DiagnosePodToolWrapper) Info(ctx context.Context) (*schema.ToolInfo, error) {
	// Carefully check syntax and structure here
	return &schema.ToolInfo{
		Name:        "diagnose_kubernetes_pod", // Name agent uses
		Description: "Runs a diagnostic sequence for a specific Kubernetes pod. Fetches pod description and recent logs. Requires 'namespace' and 'pod_name' as input.",
		Schema: &schema.ToolSchema{ // Pointer to ToolSchema
			Type: "object", // Type of the overall input schema
			Properties: map[string]*schema.ToolSchema{ // Map of property names to their schemas (pointers)
				"namespace": { // Schema for the 'namespace' property
					Type:        "string",
					Description: "The Kubernetes namespace where the pod resides.",
				}, // Comma after property schema
				"pod_name": { // Schema for the 'pod_name' property
					Type:        "string",
					Description: "The name of the Kubernetes pod to diagnose.",
				}, // Comma after property schema
			},                                           // Comma after Properties map
			Required: []string{"namespace", "pod_name"}, // Slice of required property names
		}, // Comma after Schema field
	}, nil // Return nil error on success
} // End of Info function

// Call executes the underlying chain when the Agent invokes this tool.
func (w *DiagnosePodToolWrapper) Call(ctx context.Context, input map[string]any) (any, error) {
	if w.chain == nil {
		return nil, errors.New("internal error: chain is nil in tool wrapper")
	}

	fmt.Println("[ToolWrapper] DiagnosePodChain tool called with input:", input)

	// Decode map[string]any into the specific chain input struct
	var chainInput DiagnosePodChainInput

	// More robust decoding: check for nil input map first
	if input == nil {
		return nil, errors.New("tool input cannot be nil")
	}

	inputBytes, err := json.Marshal(input)
	if err != nil {
		// Wrap error for context
		return nil, fmt.Errorf("failed to marshal input map for chain: %w", err)
	}
	if err := json.Unmarshal(inputBytes, &chainInput); err != nil {
		// Provide more specific error message
		return nil, fmt.Errorf("failed to unmarshal input for chain: %w. Ensure 'namespace' (string) and 'pod_name' (string) are provided", err)
	}

	// Add validation for decoded input fields
	if chainInput.Namespace == "" || chainInput.PodName == "" {
		return nil, errors.New("invalid input: 'namespace' and 'pod_name' cannot be empty")
	}

	// Run the actual chain
	result, err := w.chain.Run(ctx, &chainInput)
	if err != nil {
		// Return the error so the agent knows the tool failed
		// Wrap the error for more context
		return nil, fmt.Errorf("diagnose_kubernetes_pod tool execution failed: %w", err)
	}

	// Return the string result from the chain
	return result, nil // Return nil error on success
}

// Verify interface compliance
var _ tool.BaseTool = (*DiagnosePodToolWrapper)(nil)
