// Package quality provides the Code Assurance module.
// It analyzes source code against defined quality guidelines, detects issues,
// and suggests actionable remediations. Designed for orchestration within the
// Quality Agent as the "code assurance" stage.
package quality

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "hash/fnv"
    "regexp"
    "sort"
    "strings"
    "time"
)

// ChatMessage defines the basic role/content structure for LLM communication.
type ChatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

// ChatModel abstracts the underlying LLM client (Groq, OpenAI, etc.).
// Supply an adapter that implements this to integrate your provider.
// If nil, the module runs static heuristics only.
type ChatModel interface {
    Generate(ctx context.Context, messages []ChatMessage) (string, error)
}

// CodeFile represents a single code artifact to be analyzed.
type CodeFile struct {
    Path     string `json:"path"`
    Content  string `json:"content"`
    Language string `json:"language,omitempty"` // Optional if known
}

// CodeAssuranceRequest holds all necessary input for the code review process.
type CodeAssuranceRequest struct {
    Goal               string     `json:"goal,omitempty"`               // Optional high-level purpose of the analysis
    Language           string     `json:"language,omitempty"`           // Predominant language (e.g., "go", "ts", "python")
    Files              []CodeFile `json:"files"`                        // Code files to analyze (required)
    Guidelines         []string   `json:"guidelines,omitempty"`         // Quality standards to apply
    SeverityThreshold  string     `json:"severityThreshold,omitempty"`  // Minimum severity to report (low|medium|high|critical)
    MaxFindings        int        `json:"maxFindings,omitempty"`        // Cap on reported issues (0 = no cap)
    RequestUnifiedDiff bool       `json:"requestUnifiedDiff,omitempty"` // Ask LLM to return unified diffs when applicable
}

// Finding represents a single detected issue in the analyzed code.
type Finding struct {
    ID          string  `json:"id"`
    Title       string  `json:"title"`
    Description string  `json:"description"`
    File        string  `json:"file"`
    LineStart   int     `json:"lineStart,omitempty"`
    LineEnd     int     `json:"lineEnd,omitempty"`
    Severity    string  `json:"severity"`                // low | medium | high | critical
    Category    string  `json:"category"`                // style | bug | security | performance | maintainability | compliance
    Rule        string  `json:"rule,omitempty"`          // Linter/static analysis rule name
    CWE         string  `json:"cwe,omitempty"`           // CWE identifier when applicable
    Evidence    string  `json:"evidence,omitempty"`      // Code snippet or rationale
    Impact      string  `json:"impact,omitempty"`        // Why this matters
    Likelihood  string  `json:"likelihood,omitempty"`    // Qualitative likelihood
    Remediation string  `json:"remediation,omitempty"`   // Recommended fix
    Diff        string  `json:"diff,omitempty"`          // Optional unified diff
    Confidence  float64 `json:"confidence,omitempty"`    // 0.0–1.0 confidence level
}

// CodeAssuranceResult aggregates all analysis outcomes.
type CodeAssuranceResult struct {
    SchemaVersion string    `json:"schemaVersion"`
    Summary       string    `json:"summary"`
    Score         float64   `json:"score"`      // 0–100, higher = better
    Confidence    float64   `json:"confidence"` // Overall certainty (0–1)
    Findings      []Finding `json:"findings"`
    ExecutionMS   int64     `json:"executionMS"`
}

// RunCodeAssurance executes static heuristics and optionally augments with LLM analysis.
// The LLM is expected to return structured JSON that matches CodeAssuranceResult or contains a "findings" array.
func RunCodeAssurance(ctx context.Context, model ChatModel, req CodeAssuranceRequest) (*CodeAssuranceResult, error) {
    start := time.Now()

    if err := validateRequest(req); err != nil {
        return nil, err
    }
    minSeverity := normalizeSeverity(defaultSeverity(req.SeverityThreshold))

    // 1) Static heuristics (fast, deterministic)
    staticFindings := runStaticHeuristics(req)

    // 2) Optional LLM analysis for deeper insights
    var llmFindings []Finding
    if model != nil {
        if f, err := runLLMAssurance(ctx, model, req); err == nil {
            llmFindings = f
        } else {
            // Non-fatal: keep static results if LLM fails
        }
    }

    // Merge findings (static first, then LLM)
    merged := append([]Finding{}, staticFindings...)
    merged = append(merged, llmFindings...)

    // Post-process: normalize, filter by severity, deduplicate, sort, cap
    merged = normalizeFindings(merged)
    merged = filterBySeverity(merged, minSeverity)
    merged = dedupeFindings(merged)
    sortFindings(merged)
    if req.MaxFindings > 0 && len(merged) > req.MaxFindings {
        merged = merged[:req.MaxFindings]
    }

    score := computeQualityScore(merged)
    confidence := computeConfidence(merged, llmFindings)

    result := &CodeAssuranceResult{
        SchemaVersion: "1.0.0",
        Summary:       summarize(merged),
        Score:         score,
        Confidence:    confidence,
        Findings:      merged,
        ExecutionMS:   time.Since(start).Milliseconds(),
    }
    return result, nil
}

// -------- Validation and normalization --------

func validateRequest(req CodeAssuranceRequest) error {
    if len(req.Files) == 0 {
        return errors.New("no code files provided")
    }
    return nil
}

func defaultSeverity(s string) string {
    switch strings.ToLower(strings.TrimSpace(s)) {
    case "low", "medium", "high", "critical":
        return strings.ToLower(s)
    default:
        return "low"
    }
}

func normalizeSeverity(s string) string {
    switch strings.ToLower(strings.TrimSpace(s)) {
    case "critical":
        return "critical"
    case "high":
        return "high"
    case "medium":
        return "medium"
    default:
        return "low"
    }
}

func severityWeight(s string) int {
    switch normalizeSeverity(s) {
    case "critical":
        return 12
    case "high":
        return 7
    case "medium":
        return 3
    default:
        return 1
    }
}

func severityRank(s string) int {
    switch normalizeSeverity(s) {
    case "critical":
        return 4
    case "high":
        return 3
    case "medium":
        return 2
    default:
        return 1
    }
}

func normalizeFindings(in []Finding) []Finding {
    out := make([]Finding, len(in))
    copy(out, in)
    for i := range out {
        out[i].Severity = normalizeSeverity(out[i].Severity)
        if out[i].Category == "" {
            out[i].Category = "maintainability"
        }
        if out[i].ID == "" {
            out[i].ID = genFindingID(out[i])
        }
        if out[i].Confidence <= 0 {
            out[i].Confidence = 0.7
        }
    }
    return out
}

func filterBySeverity(findings []Finding, min string) []Finding {
    minRank := severityRank(min)
    out := make([]Finding, 0, len(findings))
    for _, f := range findings {
        if severityRank(f.Severity) >= minRank {
            out = append(out, f)
        }
    }
    return out
}

func sortFindings(findings []Finding) {
    sort.SliceStable(findings, func(i, j int) bool {
        // Desc by severity, then confidence, then file, then line
        if severityRank(findings[i].Severity) != severityRank(findings[j].Severity) {
            return severityRank(findings[i].Severity) > severityRank(findings[j].Severity)
        }
        if findings[i].Confidence != findings[j].Confidence {
            return findings[i].Confidence > findings[j].Confidence
        }
        if findings[i].File != findings[j].File {
            return findings[i].File < findings[j].File
        }
        return findings[i].LineStart < findings[j].LineStart
    })
}

func dedupeFindings(in []Finding) []Finding {
    seen := make(map[string]struct{}, len(in))
    out := make([]Finding, 0, len(in))
    for _, f := range in {
        key := f.File + "|" + f.Title + "|" + fmt.Sprintf("%d-%d", f.LineStart, f.LineEnd)
        if _, exists := seen[key]; exists {
            continue
        }
        seen[key] = struct{}{}
        out = append(out, f)
    }
    return out
}

// -------- Static heuristics (language-agnostic + light language-aware) --------

func runStaticHeuristics(req CodeAssuranceRequest) []Finding {
    var findings []Finding
    for _, file := range req.Files {
        path := file.Path
        lines := strings.Split(file.Content, "\n")

        // 1) TODO/FIXME
        for i, line := range lines {
            if strings.Contains(line, "TODO") || strings.Contains(line, "FIXME") {
                findings = append(findings, Finding{
                    Title:       "Work-in-progress marker present (TODO/FIXME)",
                    Description: "Found a TODO/FIXME marker. Consider resolving or converting into a tracked issue.",
                    File:        path,
                    LineStart:   i + 1,
                    Severity:    "low",
                    Category:    "maintainability",
                    Rule:        "WIP.Marker",
                    Evidence:    trimEvidence(line),
                    Remediation: "Address the pending task or link to an issue; avoid leaving TODO/FIXME in production code.",
                    Confidence:  0.65,
                })
            }
        }

        // 2) Hard-coded secrets (basic heuristics)
        findings = append(findings, scanSecrets(path, lines)...)

        // 3) Dangerous dynamic execution patterns
        findings = append(findings, scanDynamicExecution(path, lines)...)

        // 4) Large file heuristic
        if len(lines) > 1000 {
            findings = append(findings, Finding{
                Title:       "Large file",
                Description: "File is large; consider splitting into smaller modules to improve readability and testability.",
                File:        path,
                LineStart:   1,
                Severity:    "medium",
                Category:    "maintainability",
                Rule:        "File.Size",
                Evidence:    fmt.Sprintf("%d lines", len(lines)),
                Remediation: "Refactor into cohesive components with clear responsibilities.",
                Confidence:  0.8,
            })
        }

        // 5) Console/log noise in JS/TS
        if isJavaScriptLike(file.Path, file.Language) {
            for i, line := range lines {
                if strings.Contains(line, "console.log(") || strings.Contains(line, "console.debug(") {
                    findings = append(findings, Finding{
                        Title:       "Debug logging present",
                        Description: "Debug logging statements found; remove or guard with environment flags for production.",
                        File:        path,
                        LineStart:   i + 1,
                        Severity:    "low",
                        Category:    "style",
                        Rule:        "Logging.DebugNoise",
                        Evidence:    trimEvidence(line),
                        Remediation: "Use a structured logger with levels and avoid noisy logs in hot paths.",
                        Confidence:  0.7,
                    })
                }
            }
        }

        // 6) Go-specific risky patterns (very light-touch)
        if isGoLike(file.Path, file.Language) {
            for i, line := range lines {
                if strings.Contains(line, "panic(") {
                    findings = append(findings, Finding{
                        Title:       "Use of panic in application code",
                        Description: "Panic should be avoided in application/runtime paths; prefer error returns and handling.",
                        File:        path,
                        LineStart:   i + 1,
                        Severity:    "medium",
                        Category:    "reliability",
                        Rule:        "Go.PanicUsage",
                        Evidence:    trimEvidence(line),
                        Remediation: "Return errors and handle them at appropriate boundaries; reserve panic for unrecoverable programmer errors.",
                        Confidence:  0.75,
                    })
                }
                if strings.Contains(line, "os/exec") || strings.Contains(line, "exec.Command(") {
                    findings = append(findings, Finding{
                        Title:       "External command execution",
                        Description: "Executing external commands can be dangerous and platform-dependent.",
                        File:        path,
                        LineStart:   i + 1,
                        Severity:    "medium",
                        Category:    "security",
                        Rule:        "Go.ExecUsage",
                        Evidence:    trimEvidence(line),
                        Remediation: "Validate inputs rigorously, sandbox execution, and capture/limit resources and time.",
                        Confidence:  0.75,
                    })
                }
            }
        }

        // 7) Naive SQL concatenation detection (any language)
        for i, line := range lines {
            if strings.Contains(strings.ToLower(line), "select ") && strings.Contains(line, "+") {
                findings = append(findings, Finding{
                    Title:       "Potential SQL string concatenation",
                    Description: "String concatenation in SQL may lead to SQL injection vulnerabilities.",
                    File:        path,
                    LineStart:   i + 1,
                    Severity:    "high",
                    Category:    "security",
                    Rule:        "SQL.Concat",
                    CWE:         "CWE-89",
                    Evidence:    trimEvidence(line),
                    Remediation: "Use prepared statements or parameterized queries.",
                    Confidence:  0.7,
                })
            }
        }
    }
    return findings
}

func scanSecrets(path string, lines []string) []Finding {
    var findings []Finding

    awsKey := regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
    genericPass := regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*['"][^'"]+['"]`)
    genericSecret := regexp.MustCompile(`(?i)(api[_-]?key|secret|token)\s*[:=]\s*['"][^'"]+['"]`)

    for i, line := range lines {
        switch {
        case awsKey.MatchString(line):
            findings = append(findings, Finding{
                Title:       "Hard-coded AWS Access Key detected",
                Description: "Embedding AWS keys in source code risks account compromise.",
                File:        path,
                LineStart:   i + 1,
                Severity:    "critical",
                Category:    "security",
                Rule:        "Secrets.AWSKey",
                CWE:         "CWE-798",
                Evidence:    trimEvidence(line),
                Remediation: "Remove the key from code, rotate the credentials, and use a secrets manager or environment variables.",
                Confidence:  0.95,
            })
        case genericPass.MatchString(line), genericSecret.MatchString(line):
            findings = append(findings, Finding{
                Title:       "Potential hard-coded secret",
                Description: "Sensitive credentials appear to be hard-coded.",
                File:        path,
                LineStart:   i + 1,
                Severity:    "high",
                Category:    "security",
                Rule:        "Secrets.Generic",
                CWE:         "CWE-798",
                Evidence:    trimEvidence(line),
                Remediation: "Move secrets to a secure store or environment variables; rotate any exposed credentials.",
                Confidence:  0.85,
            })
        }
    }

    return findings
}

func scanDynamicExecution(path string, lines []string) []Finding {
    var findings []Finding

    // Language-agnostic suspicious patterns
    evalLike := regexp.MustCompile(`(?i)\beval\s*\(`)
    funcCtor := regexp.MustCompile(`(?i)new\s+Function\s*\(`)
    processExec := regexp.MustCompile(`(?i)\b(exec|popen|system)\s*\(`)

    for i, line := range lines {
        switch {
        case evalLike.MatchString(line):
            findings = append(findings, Finding{
                Title:       "Use of eval-like dynamic code execution",
                Description: "Dynamic code evaluation is dangerous and often unnecessary.",
                File:        path,
                LineStart:   i + 1,
                Severity:    "high",
                Category:    "security",
                Rule:        "Exec.Eval",
                CWE:         "CWE-94",
                Evidence:    trimEvidence(line),
                Remediation: "Avoid eval; use safer parsing/serialization strategies or whitelisted interpreters.",
                Confidence:  0.85,
            })
        case funcCtor.MatchString(line):
            findings = append(findings, Finding{
                Title:       "Dynamic function constructor",
                Description: "Constructing functions from strings can lead to code injection.",
                File:        path,
                LineStart:   i + 1,
                Severity:    "high",
                Category:    "security",
                Rule:        "Exec.FunctionConstructor",
                CWE:         "CWE-94",
                Evidence:    trimEvidence(line),
                Remediation: "Refactor to avoid runtime code construction; validate and limit inputs strictly.",
                Confidence:  0.8,
            })
        case processExec.MatchString(line):
            findings = append(findings, Finding{
                Title:       "Process execution from code",
                Description: "Spawning external commands can introduce security and portability risks.",
                File:        path,
                LineStart:   i + 1,
                Severity:    "medium",
                Category:    "security",
                Rule:        "Exec.Process",
                Evidence:    trimEvidence(line),
                Remediation: "Validate arguments; sandbox and limit resources; prefer native libraries where possible.",
                Confidence:  0.75,
            })
        }
    }
    return findings
}

// -------- LLM augmentation --------

func runLLMAssurance(ctx context.Context, model ChatModel, req CodeAssuranceRequest) ([]Finding, error) {
    sys := buildSystemPrompt(req)
    usr := buildUserPrompt(req)

    resp, err := model.Generate(ctx, []ChatMessage{
        {Role: "system", Content: sys},
        {Role: "user", Content: usr},
    })
    if err != nil {
        return nil, err
    }

    // Try to parse as full result, then as an object with findings, then as an array
    if f, ok := parseFindingsFromJSON(resp); ok {
        return f, nil
    }
    // Fallback: try extracting the largest JSON fragment
    if fragment := extractJSONFragment(resp); fragment != "" {
        if f, ok := parseFindingsFromJSON(fragment); ok {
            return f, nil
        }
    }

    // If model returned plain text, we fail softly (no extra findings)
    return nil, fmt.Errorf("unable to parse LLM response into findings")
}

func buildSystemPrompt(req CodeAssuranceRequest) string {
    minSeverity := defaultSeverity(req.SeverityThreshold)
    builder := &strings.Builder{}
    fmt.Fprintf(builder, "You are a senior code quality auditor.\n")
    fmt.Fprintf(builder, "Assess the provided code with a focus on correctness, security, performance, maintainability, and compliance.\n")
    fmt.Fprintf(builder, "Language context: %s.\n", safe(req.Language, "unspecified"))
    if len(req.Guidelines) > 0 {
        fmt.Fprintf(builder, "Apply these guidelines:\n")
        for _, g := range req.Guidelines {
            fmt.Fprintf(builder, "- %s\n", g)
        }
    }
    fmt.Fprintf(builder, "Only report issues with severity '%s' or higher.\n", minSeverity)
    fmt.Fprintf(builder, "Respond in strict JSON. Either:\n")
    fmt.Fprintf(builder, "1) {\"findings\":[...]}\n")
    fmt.Fprintf(builder, "or 2) a plain array: [{...}]\n")
    fmt.Fprintf(builder, "Each finding must include: title, description, file, lineStart, severity, category, remediation.\n")
    if req.RequestUnifiedDiff {
        fmt.Fprintf(builder, "Include a unified diff patch where appropriate in a 'diff' field.\n")
    }
    return builder.String()
}

func buildUserPrompt(req CodeAssuranceRequest) string {
    builder := &strings.Builder{}
    if strings.TrimSpace(req.Goal) != "" {
        fmt.Fprintf(builder, "Goal: %s\n\n", req.Goal)
    }
    fmt.Fprintf(builder, "Files:\n")
    for _, f := range req.Files {
        lang := f.Language
        if lang == "" {
            lang = guessLanguageFromPath(f.Path)
        }
        fmt.Fprintf(builder, "=== FILE: %s (lang: %s) ===\n", f.Path, lang)
        // Keep size practical; LLM adapters should handle chunking if needed
        fmt.Fprintf(builder, "%s\n\n", f.Content)
    }
    return builder.String()
}

func parseFindingsFromJSON(s string) ([]Finding, bool) {
    // Try top-level result
    var result CodeAssuranceResult
    if err := json.Unmarshal([]byte(s), &result); err == nil && len(result.Findings) > 0 {
        return result.Findings, true
    }
    // Try object with findings
    var obj map[string]json.RawMessage
    if err := json.Unmarshal([]byte(s), &obj); err == nil {
        if raw, ok := obj["findings"]; ok {
            var f []Finding
            if err := json.Unmarshal(raw, &f); err == nil && len(f) > 0 {
                return f, true
            }
        }
    }
    // Try bare array
    var arr []Finding
    if err := json.Unmarshal([]byte(s), &arr); err == nil && len(arr) > 0 {
        return arr, true
    }
    return nil, false
}

func extractJSONFragment(s string) string {
    // Simple heuristic: find the largest {...} or [...] block
    type pair struct{ start, end int }
    best := pair{-1, -1}
    stack := []struct {
        ch    rune
        index int
    }{}
    for i, r := range s {
        switch r {
        case '{', '[':
            stack = append(stack, struct {
                ch    rune
                index int
            }{r, i})
        case '}', ']':
            if len(stack) == 0 {
                continue
            }
            top := stack[len(stack)-1]
            stack = stack[:len(stack)-1]
            if (top.ch == '{' && r == '}') || (top.ch == '[' && r == ']') {
                if top.index >= 0 && i > top.index {
                    if (best.start == -1 && best.end == -1) || (i-top.index) > (best.end-best.start) {
                        best = pair{top.index, i + 1}
                    }
                }
            }
        }
    }
    if best.start >= 0 && best.end > best.start {
        return s[best.start:best.end]
    }
    return ""
}

// -------- Scoring, confidence, and utilities --------

func computeQualityScore(findings []Finding) float64 {
    score := 100.0
    for _, f := range findings {
        score -= float64(severityWeight(f.Severity))
    }
    if score < 0 {
        score = 0
    }
    if score > 100 {
        score = 100
    }
    return score
}

func computeConfidence(all, llm []Finding) float64 {
    // If LLM contributed, blend a bit higher confidence
    base := 0.7
    if len(llm) > 0 {
        base = 0.75
    }
    // Slightly adjust by average per-finding confidence (bounded)
    if len(all) == 0 {
        return base
    }
    var sum float64
    for _, f := range all {
        sum += clamp(f.Confidence, 0.4, 0.98)
    }
    avg := sum / float64(len(all))
    // Blend averages
    return clamp((base+avg)/2.0, 0.5, 0.95)
}

func summarize(findings []Finding) string {
    if len(findings) == 0 {
        return "No issues found at or above the configured severity threshold."
    }
    var crit, high, med, low int
    for _, f := range findings {
        switch normalizeSeverity(f.Severity) {
        case "critical":
            crit++
        case "high":
            high++
        case "medium":
            med++
        default:
            low++
        }
    }
    return fmt.Sprintf("Issues found — critical: %d, high: %d, medium: %d, low: %d.", crit, high, med, low)
}

func genFindingID(f Finding) string {
    h := fnv.New64a()
    _, _ = h.Write([]byte(f.File))
    _, _ = h.Write([]byte(f.Title))
    _, _ = h.Write([]byte(fmt.Sprintf("%d-%d", f.LineStart, f.LineEnd)))
    return fmt.Sprintf("F%08x", h.Sum64())
}

func trimEvidence(line string) string {
    line = strings.TrimSpace(line)
    const max = 200
    if len(line) > max {
        return line[:max] + "..."
    }
    return line
}

func isJavaScriptLike(path, lang string) bool {
    l := strings.ToLower(strings.TrimSpace(lang))
    if l == "js" || l == "javascript" || l == "ts" || l == "typescript" {
        return true
    }
    p := strings.ToLower(path)
    return strings.HasSuffix(p, ".js") || strings.HasSuffix(p, ".jsx") || strings.HasSuffix(p, ".ts") || strings.HasSuffix(p, ".tsx")
}

func isGoLike(path, lang string) bool {
    l := strings.ToLower(strings.TrimSpace(lang))
    if l == "go" || l == "golang" {
        return true
    }
    return strings.HasSuffix(strings.ToLower(path), ".go")
}

func guessLanguageFromPath(path string) string {
    p := strings.ToLower(path)
    switch {
    case strings.HasSuffix(p, ".go"):
        return "go"
    case strings.HasSuffix(p, ".ts"), strings.HasSuffix(p, ".tsx"):
        return "ts"
    case strings.HasSuffix(p, ".js"), strings.HasSuffix(p, ".jsx"):
        return "js"
    case strings.HasSuffix(p, ".py"):
        return "python"
    case strings.HasSuffix(p, ".rb"):
        return "ruby"
    case strings.HasSuffix(p, ".java"):
        return "java"
    case strings.HasSuffix(p, ".cs"):
        return "csharp"
    case strings.HasSuffix(p, ".php"):
        return "php"
    default:
        return "plain"
    }
}

func safe(s, fallback string) string {
    if strings.TrimSpace(s) == "" {
        return fallback
    }
    return s
}

func clamp(v, lo, hi float64) float64 {
    if v < lo {
        return lo
    }
    if v > hi {
        return hi
    }
    return v
}

