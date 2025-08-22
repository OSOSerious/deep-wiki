package quality

import (
    "context"
    "fmt"
    "strings"

    "github.com/sormind/OSA/miosa-backend/internal/agents"
)

// QualityReviewTool runs a lightweight QA report using the QualityAgent pipeline.
type QualityReviewTool struct {
    agent *QualityAgent
}

// GetName returns the unique tool name.
func (t *QualityReviewTool) GetName() string {
    return "quality_review"
}

// GetDescription describes the tool’s purpose.
func (t *QualityReviewTool) GetDescription() string {
    return "Run a code quality review and receive a structured report with metrics and recommendations."
}

// Execute runs the tool using the agent logic.
func (t *QualityReviewTool) Execute(ctx context.Context, params map[string]string) (string, error) {
    // Expecting code snippet or identifier in params["input"]
    input := strings.TrimSpace(params["input"])
    if input == "" {
        return "", fmt.Errorf("missing 'input' parameter for quality review")
    }

    result, err := t.agent.Execute(ctx, agents.Task{
        Input:      input,
        Parameters: params,
    })
    if err != nil {
        return "", err
    }
    return result.Output, nil
}

// -------- Another example tool --------

// QuickConfidenceTool calculates only the confidence score from mocked or provided metrics.
type QuickConfidenceTool struct{}

func (t *QuickConfidenceTool) GetName() string {
    return "quality_confidence_estimator"
}

func (t *QuickConfidenceTool) GetDescription() string {
    return "Quickly estimate QA confidence score given basic metrics."
}

func (t *QuickConfidenceTool) Execute(ctx context.Context, params map[string]string) (string, error) {
    // Read numeric params or apply defaults
    metrics := Metrics{
        TotalFiles:          parseInt(params["total_files"], 1),
        TotalLines:          parseInt(params["total_lines"], 100),
        IssuesFound:         parseInt(params["issues_found"], 0),
        TestsGenerated:      parseInt(params["tests_generated"], 1),
        TestsPassed:         parseInt(params["tests_passed"], 1),
        TestsFailed:         parseInt(params["tests_failed"], 0),
        CodeComplexityScore: parseFloat(params["complexity"], 1.0),
        CoveragePercent:     parseFloat(params["coverage"], 100.0),
    }
    conf := (&QualityAgent{}).calculateConfidence(metrics)
    return fmt.Sprintf("Estimated Confidence: %.1f / 10", conf), nil
}

// -------- Registration helper --------

// RegisterQualityTools adds quality-related tools to the global registry.
func RegisterQualityTools(agent *QualityAgent) error {
    // Main review tool
    if err := agents.RegisterTool(&QualityReviewTool{agent: agent}); err != nil {
        return err
    }
    // Quick estimator tool
    if err := agents.RegisterTool(&QuickConfidenceTool{}); err != nil {
        return err
    }
    // Associate with agent type
    if err := agents.RegisterToolForAgent(agent.GetType(), "quality_review"); err != nil {
        return err
    }
    if err := agents.RegisterToolForAgent(agent.GetType(), "quality_confidence_estimator"); err != nil {
        return err
    }
    return nil
}

// -------- Utility parsing --------

func parseInt(s string, def int) int {
    if s == "" {
        return def
    }
    var v int
    _, err := fmt.Sscanf(s, "%d", &v)
    if err != nil {
        return def
    }
    return v
}

func parseFloat(s string, def float64) float64 {
    if s == "" {
        return def
    }
    var v float64
    _, err := fmt.Sscanf(s, "%f", &v)
    if err != nil {
        return def
    }
    return v
}
package quality
// Tools implementation
