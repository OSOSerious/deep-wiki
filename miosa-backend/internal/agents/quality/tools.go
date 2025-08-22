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
func (t *QualityReviewTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Expecting code snippet or identifier in params["input"]
    inputRaw, ok := params["input"]
    if !ok {
        return nil, fmt.Errorf("missing 'input' parameter for quality review")
    }
    input := strings.TrimSpace(fmt.Sprintf("%v", inputRaw))
    if input == "" {
        return nil, fmt.Errorf("empty 'input' parameter for quality review")
    }

    result, err := t.agent.Execute(ctx, agents.Task{
        Input:      input,
        Parameters: params,
    })
    if err != nil {
        return nil, err
    }
    return result.Output, nil
}

func (t *QualityReviewTool) Validate(input map[string]interface{}) error {
    if _, ok := input["input"]; !ok {
        return fmt.Errorf("missing required parameter: input")
    }
    return nil
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

func (t *QuickConfidenceTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Convert params to string map
    strParams := make(map[string]string)
    for k, v := range params {
        strParams[k] = fmt.Sprintf("%v", v)
    }
    
    // Read numeric params or apply defaults
    metrics := Metrics{
        TotalFiles:          parseInt(strParams["total_files"], 1),
        TotalLines:          parseInt(strParams["total_lines"], 100),
        IssuesFound:         parseInt(strParams["issues_found"], 0),
        TestsGenerated:      parseInt(strParams["tests_generated"], 1),
        TestsPassed:         parseInt(strParams["tests_passed"], 1),
        TestsFailed:         parseInt(strParams["tests_failed"], 0),
        CodeComplexityScore: parseFloat(strParams["complexity"], 1.0),
        CoveragePercent:     parseFloat(strParams["coverage"], 100.0),
    }
    conf := (&QualityAgent{}).calculateConfidence(metrics)
    return fmt.Sprintf("Estimated Confidence: %.1f / 10", conf), nil
}

func (t *QuickConfidenceTool) Validate(input map[string]interface{}) error {
    // All parameters are optional with defaults
    return nil
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

// Tools implementation
