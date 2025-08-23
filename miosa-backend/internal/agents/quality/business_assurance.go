
// Package quality provides the Business Assurance module.
// It analyzes business artifacts (documents, processes, policies, contracts, etc.)
// against defined business rules, compliance frameworks, and operational guidelines.
// Designed for orchestration within the Quality Agent as the "business assurance" stage.
package quality

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "hash/fnv"
    "sort"
    "strings"
    "time"
)

// ChatMessage defines the role/content structure for LLM communication.
type ChatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

// ChatModel abstracts the underlying LLM client (e.g., Groq, OpenAI).
type ChatModel interface {
    Generate(ctx context.Context, messages []ChatMessage) (string, error)
}

// BusinessArtifact represents an individual business asset to be analyzed.
type BusinessArtifact struct {
    Name     string `json:"name"`
    Content  string `json:"content"`
    Category string `json:"category,omitempty"` // e.g., contract, policy, process
}

// BusinessAssuranceRequest is the input for the business assurance process.
type BusinessAssuranceRequest struct {
    Goal              string             `json:"goal,omitempty"`
    Artifacts         []BusinessArtifact `json:"artifacts"`
    Guidelines        []string           `json:"guidelines,omitempty"`
    SeverityThreshold string             `json:"severityThreshold,omitempty"` // low|medium|high|critical
    MaxFindings       int                `json:"maxFindings,omitempty"`
    RequestDiff       bool               `json:"requestDiff,omitempty"`
}

// BusinessFinding describes an issue found in an artifact.
type BusinessFinding struct {
    ID          string  `json:"id"`
    Title       string  `json:"title"`
    Description string  `json:"description"`
    Artifact    string  `json:"artifact"`
    Severity    string  `json:"severity"` // low|medium|high|critical
    Category    string  `json:"category"` // compliance|policy|process|data|quality
    Rule        string  `json:"rule,omitempty"`
    Impact      string  `json:"impact,omitempty"`
    Remediation string  `json:"remediation,omitempty"`
    Diff        string  `json:"diff,omitempty"`
    Confidence  float64 `json:"confidence,omitempty"`
}

// BusinessAssuranceResult is the aggregated output of the analysis.
type BusinessAssuranceResult struct {
    SchemaVersion string            `json:"schemaVersion"`
    Summary       string            `json:"summary"`
    Score         float64           `json:"score"`
    Confidence    float64           `json:"confidence"`
    Findings      []BusinessFinding `json:"findings"`
    ExecutionMS   int64             `json:"executionMS"`
}

// RunBusinessAssurance executes static heuristics and optionally augments with LLM analysis.
func RunBusinessAssurance(ctx context.Context, model ChatModel, req BusinessAssuranceRequest) (*BusinessAssuranceResult, error) {
    start := time.Now()

    if err := validateBusinessRequest(req); err != nil {
        return nil, err
    }
    minSeverity := normalizeSeverity(defaultSeverity(req.SeverityThreshold))

    // 1) Static heuristics
    staticFindings := runBusinessHeuristics(req)

    // 2) Optional LLM augmentation
    var llmFindings []BusinessFinding
    if model != nil {
        if f, err := runLLMBusinessAssurance(ctx, model, req); err == nil {
            llmFindings = f
        }
    }

    // Merge and post-process
    merged := append([]BusinessFinding{}, staticFindings...)
    merged = append(merged, llmFindings...)
    merged = normalizeBusinessFindings(merged)
    merged = filterBusinessBySeverity(merged, minSeverity)
    merged = dedupeBusinessFindings(merged)
    sortBusinessFindings(merged)
    if req.MaxFindings > 0 && len(merged) > req.MaxFindings {
        merged = merged[:req.MaxFindings]
    }

    score := computeBusinessScore(merged)
    confidence := computeBusinessConfidence(merged, llmFindings)

    return &BusinessAssuranceResult{
        SchemaVersion: "1.0.0",
        Summary:       summarizeBusiness(merged),
        Score:         score,
        Confidence:    confidence,
        Findings:      merged,
        ExecutionMS:   time.Since(start).Milliseconds(),
    }, nil
}

// ===== Validation & Normalization =====

func validateBusinessRequest(req BusinessAssuranceRequest) error {
    if len(req.Artifacts) == 0 {
        return errors.New("no business artifacts provided")
    }
    return nil
}

func normalizeBusinessFindings(in []BusinessFinding) []BusinessFinding {
    out := make([]BusinessFinding, len(in))
    copy(out, in)
    for i := range out {
        out[i].Severity = normalizeSeverity(out[i].Severity)
        if out[i].Category == "" {
            out[i].Category = "quality"
        }
        if out[i].ID == "" {
            out[i].ID = genBusinessFindingID(out[i])
        }
        if out[i].Confidence <= 0 {
            out[i].Confidence = 0.7
        }
    }
    return out
}

func filterBusinessBySeverity(findings []BusinessFinding, min string) []BusinessFinding {
    minRank := severityRank(min)
    out := make([]BusinessFinding, 0, len(findings))
    for _, f := range findings {
        if severityRank(f.Severity) >= minRank {
            out = append(out, f)
        }
    }
    return out
}

func sortBusinessFindings(findings []BusinessFinding) {
    sort.SliceStable(findings, func(i, j int) bool {
        if severityRank(findings[i].Severity) != severityRank(findings[j].Severity) {
            return severityRank(findings[i].Severity) > severityRank(findings[j].Severity)
        }
        if findings[i].Confidence != findings[j].Confidence {
            return findings[i].Confidence > findings[j].Confidence
        }
        if findings[i].Artifact != findings[j].Artifact {
            return findings[i].Artifact < findings[j].Artifact
        }
        return findings[i].Title < findings[j].Title
    })
}

func dedupeBusinessFindings(in []BusinessFinding) []BusinessFinding {
    seen := make(map[string]struct{}, len(in))
    out := make([]BusinessFinding, 0, len(in))
    for _, f := range in {
        key := f.Artifact + "|" + f.Title
        if _, exists := seen[key]; exists {
            continue
        }
        seen[key] = struct{}{}
        out = append(out, f)
    }
    return out
}

// ===== Static Heuristics =====

func runBusinessHeuristics(req BusinessAssuranceRequest) []BusinessFinding {
    var findings []BusinessFinding
    for _, art := range req.Artifacts {
        name := art.Name
        content := strings.ToLower(art.Content)

        // Example: Placeholder detection
        if strings.Contains(content, "lorem ipsum") {
            findings = append(findings, BusinessFinding{
                Title:       "Placeholder text present",
                Description: "The artifact contains placeholder text, which suggests it may be incomplete.",
                Artifact:    name,
                Severity:    "low",
                Category:    "quality",
                Rule:        "Quality.Placeholder",
                Remediation: "Replace placeholder text with finalized, approved content.",
                Confidence:  0.8,
            })
        }

        // Example: Compliance keyword missing
        if strings.Contains(content, "personal data") && !strings.Contains(content, "GDPR") && !strings.Contains(content, "LGPD") {
            findings = append(findings, BusinessFinding{
                Title:       "Potential data protection compliance gap",
                Description: "Personal data is mentioned without reference to relevant data protection regulations.",
                Artifact:    name,
                Severity:    "high",
                Category:    "compliance",
                Rule:        "Compliance.DataProtection",
                Remediation: "Add explicit references to applicable data protection laws, such as GDPR or LGPD.",
                Confidence:  0.85,
            })
        }

        // Example: Missing approval section
        if strings.Contains(art.Category, "policy") && !strings.Contains(content, "approved by") {
            findings = append(findings, BusinessFinding{
                Title:       "Policy missing approval section",
                Description: "Policies should include an approval record for accountability.",
                Artifact:    name,
                Severity:    "medium",
                Category:    "policy",
                Rule:        "Policy.MissingApproval",
                Remediation: "Include an 'Approved by' section with responsible authority and date.",
                Confidence:  0.75,
            })
        }
    }
    return findings
}

// ===== LLM Augmentation =====

func runLLMBusinessAssurance(ctx context.Context, model ChatModel, req BusinessAssuranceRequest) ([]BusinessFinding, error) {
    sys := buildBusinessSystemPrompt(req)
    usr := buildBusinessUserPrompt(req)
    resp, err := model.Generate(ctx, []ChatMessage{
        {Role: "system", Content: sys},
        {Role: "user", Content: usr},
    })
    if err != nil {
        return nil, err
