
// Package quality provides the Visual Assurance module.
// It analyzes visual artifacts (UI mocks, screenshots, design specs) against
// accessibility, brand, layout, and content guidelines. It detects issues and
// suggests actionable remediations. Designed for orchestration within the
// Quality Agent as the "visual assurance" stage.
package quality

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "hash/fnv"
    "math"
    "regexp"
    "sort"
    "strconv"
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

// VisualComponent describes a detected or specified UI element.
type VisualComponent struct {
    ID        string  `json:"id,omitempty"`
    Type      string  `json:"type,omitempty"`      // button|text|image|icon|input|link|card|chip|checkbox|radio|switch|menu|other
    Role      string  `json:"role,omitempty"`      // ARIA-like role if available
    Label     string  `json:"label,omitempty"`     // Accessible name / label
    Text      string  `json:"text,omitempty"`      // Visible text content
    Alt       string  `json:"alt,omitempty"`       // Alt text for images/icons
    X         float64 `json:"x"`                   // Top-left X (px)
    Y         float64 `json:"y"`                   // Top-left Y (px)
    W         float64 `json:"w"`                   // Width (px)
    H         float64 `json:"h"`                   // Height (px)
    FgColor   string  `json:"fgColor,omitempty"`   // Text/foreground color (#RRGGBB or #RGB)
    BgColor   string  `json:"bgColor,omitempty"`   // Background color (#RRGGBB or #RGB)
    FontSize  float64 `json:"fontSize,omitempty"`  // px
    Radius    float64 `json:"radius,omitempty"`    // corner radius (px)
    Clickable bool    `json:"clickable,omitempty"` // inferred or specified
}

// VisualArtifact represents a visual asset to be analyzed.
type VisualArtifact struct {
    Name         string            `json:"name"`
    Description  string            `json:"description,omitempty"`   // optional human description
    OCRText      string            `json:"ocrText,omitempty"`       // extracted text from the image
    Theme        string            `json:"theme,omitempty"`         // light|dark|auto|unspecified
    Dimensions   [2]float64        `json:"dimensions,omitempty"`    // [width, height] in px
    Colors       []string          `json:"colors,omitempty"`        // detected dominant colors (#RRGGBB)
    BrandColors  []string          `json:"brandColors,omitempty"`   // allowed brand palette colors (#RRGGBB)
    Components   []VisualComponent `json:"components,omitempty"`    // structured components if available
    Metadata     map[string]string `json:"metadata,omitempty"`      // free-form metadata
    Locale       string            `json:"locale,omitempty"`        // e.g., en-US, pt-BR
    PrimaryBG    string            `json:"primaryBg,omitempty"`     // overall background color if known
    PrimaryFG    string            `json:"primaryFg,omitempty"`     // overall text color if known
    MinTapTarget float64           `json:"minTapTarget,omitempty"`  // default minimum tap size (px), overrides 44px
    LargeTextPx  float64           `json:"largeTextPx,omitempty"`   // threshold for large text contrast (defaults to 18px)
}

// VisualAssuranceRequest holds all input for the visual review process.
type VisualAssuranceRequest struct {
    Goal              string           `json:"goal,omitempty"`               // Optional high-level purpose
    Guidelines        []string         `json:"guidelines,omitempty"`         // Business/brand/accessibility rules
    Artifacts         []VisualArtifact `json:"artifacts"`                    // Visual assets to analyze
    SeverityThreshold string           `json:"severityThreshold,omitempty"`  // low|medium|high|critical
    MaxFindings       int              `json:"maxFindings,omitempty"`        // 0 = no cap
    RequestAnnotations bool            `json:"requestAnnotations,omitempty"` // Ask LLM to include annotation hints
}

// VisualFinding represents a single detected issue.
type VisualFinding struct {
    ID          string  `json:"id"`
    Title       string  `json:"title"`
    Description string  `json:"description"`
    Artifact    string  `json:"artifact"`
    ComponentID string  `json:"componentId,omitempty"`
    Rect        string  `json:"rect,omitempty"`       // "x,y,w,h" for quick reference
    Severity    string  `json:"severity"`             // low|medium|high|critical
    Category    string  `json:"category"`             // accessibility|brand|layout|content|compliance|quality
    Rule        string  `json:"rule,omitempty"`       // Rule identifier
    WCAG        string  `json:"wcag,omitempty"`       // e.g., "1.4.3 Contrast (Minimum)"
    Impact      string  `json:"impact,omitempty"`     // Why it matters
    Remediation string  `json:"remediation,omitempty"`// How to fix
    Annotation  string  `json:"annotation,omitempty"` // Optional visual annotation hint
    Confidence  float64 `json:"confidence,omitempty"` // 0.0–1.0
}

// VisualAssuranceResult aggregates the outcomes.
type VisualAssuranceResult struct {
    SchemaVersion string          `json:"schemaVersion"`
    Summary       string          `json:"summary"`
    Score         float64         `json:"score"`      // 0–100, higher = better
    Confidence    float64         `json:"confidence"` // overall certainty (0–1)
    Findings      []VisualFinding `json:"findings"`
    ExecutionMS   int64           `json:"executionMS"`
}

// RunVisualAssurance executes static heuristics and optionally augments with LLM analysis.
func RunVisualAssurance(ctx context.Context, model ChatModel, req VisualAssuranceRequest) (*VisualAssuranceResult, error) {
    start := time.Now()

    if err := validateVisualRequest(req); err != nil {
        return nil, err
    }
    minSeverity := vaNormalizeSeverity(vaDefaultSeverity(req.SeverityThreshold))

    // 1) Static heuristics
    staticFindings := runStaticVisualHeuristics(req)

    // 2) Optional LLM analysis
    var llmFindings []VisualFinding
    if model != nil {
        if f, err := runLLMVisualAssurance(ctx, model, req); err == nil {
            llmFindings = f
        }
    }

    // Merge findings
    merged := append([]VisualFinding{}, staticFindings...)
    merged = append(merged, llmFindings...)

    // Post-processing
    merged = vaNormalizeFindings(merged)
    merged = vaFilterBySeverity(merged, minSeverity)
    merged = vaDedupeFindings(merged)
    vaSortFindings(merged)

    if req.MaxFindings > 0 && len(merged) > req.MaxFindings {
        merged = merged[:req.MaxFindings]
    }

    score := vaComputeQualityScore(merged)
    confidence := vaComputeConfidence(merged, llmFindings)

    return &VisualAssuranceResult{
        SchemaVersion: "1.0.0",
        Summary:       vaSummarize(merged),
        Score:         score,
        Confidence:    confidence,
        Findings:      merged,
        ExecutionMS:   time.Since(start).Milliseconds(),
    }, nil
}

// -------- Validation --------

func validateVisualRequest(req VisualAssuranceRequest) error {
    if len(req.Artifacts) == 0 {
        return errors.New("no visual artifacts provided")
    }
    return nil
}

// -------- Static heuristics (accessibility, layout, brand, content) --------

func runStaticVisualHeuristics(req VisualAssuranceRequest) []VisualFinding {
    var findings []VisualFinding

    for _, a := range req.Artifacts {
        name := a.Name
        theme := strings.ToLower(strings.TrimSpace(a.Theme))
        minTap := a.MinTapTarget
        if minTap <= 0 {
            minTap = 44 // default Apple/Google recommended minimum
        }
        largeTextPx := a.LargeTextPx
        if largeTextPx <= 0 {
            largeTextPx = 18 // WCAG large text threshold (~18px normal weight)
        }

        // 0) Content: placeholder or lorem ipsum in OCR text
        lowerOCR := strings.ToLower(a.OCRText)
        if strings.Contains(lowerOCR, "lorem ipsum") {
            findings = append(findings, VisualFinding{
                Title:       "Placeholder text present",
                Description: "The artifact contains placeholder text (e.g., 'Lorem ipsum'), indicating it may be unfinished.",
                Artifact:    name,
                Severity:    "low",
                Category:    "quality",
                Rule:        "Quality.Placeholder",
                Impact:      "Placeholder text can confuse users and undermine trust.",
                Remediation: "Replace all placeholder text with finalized content.",
                Confidence:  0.85,
            })
        }

        // 1) Accessibility: missing alt/label for images/icons/interactive elements
        for _, c := range a.Components {
            ctype := strings.ToLower(c.Type)
            if ctype == "" && c.Clickable {
                ctype = "interactive"
            }
            isImageLike := ctype == "image" || ctype == "icon"
            isInteractive := c.Clickable || ctype == "button" || ctype == "link" || ctype == "input" || ctype == "checkbox" || ctype == "radio" || ctype == "switch"
            hasAccessibleName := strings.TrimSpace(c.Alt) != "" || strings.TrimSpace(c.Label) != "" || strings.TrimSpace(c.Text) != ""

            if isImageLike && strings.TrimSpace(c.Alt) == "" {
                findings = append(findings, VisualFinding{
                    Title:       "Missing alternative text for non-text content",
                    Description: "Image/Icon lacks alt text or accessible description.",
                    Artifact:    name,
                    ComponentID: c.ID,
                    Rect:        rectStr(c),
                    Severity:    "high",
                    Category:    "accessibility",
                    Rule:        "A11y.NonTextContent",
                    WCAG:        "1.1.1 Non-text Content",
                    Impact:      "Screen reader users cannot understand the purpose of the image.",
                    Remediation: "Provide concise, meaningful alt text or aria-label/aria-labelledby.",
                    Confidence:  0.9,
                })
            }
            if isInteractive && !hasAccessibleName {
                findings = append(findings, VisualFinding{
                    Title:       "Interactive control without accessible name",
                    Description: "Control appears interactive but lacks a visible label or accessible name.",
                    Artifact:    name,
                    ComponentID: c.ID,
                    Rect:        rectStr(c),
                    Severity:    "high",
                    Category:    "accessibility",
                    Rule:        "A11y.AccessibleName",
                    WCAG:        "4.1.2 Name, Role, Value",
                    Impact:      "Assistive technology users cannot identify or operate the control.",
                    Remediation: "Add an accessible name via visible text, aria-label, or associated label.",
                    Confidence:  0.85,
                })
            }
        }

        // 2) Accessibility: color contrast checks (component-level, then page-level fallback)
        for _, c := range a.Components {
            fg, bg := c.FgColor, c.BgColor
            // If component colors are missing, fallback to page-level primary colors
            if fg == "" {
                fg = a.PrimaryFG
            }
            if bg == "" {
                bg = a.PrimaryBG
            }
            if fg == "" || bg == "" {
                continue
            }
            if ratio, ok := contrastRatio(fg, bg); ok {
                threshold := 4.5 // default for normal text
                if c.FontSize >= largeTextPx {
                    threshold = 3.0 // large text threshold
                }
                if ratio < threshold {
                    findings = append(findings, VisualFinding{
                        Title:       "Insufficient text contrast",
                        Description: fmt.Sprintf("Measured contrast ratio %.2f:1 is below the minimum threshold %.1f:1.", ratio, threshold),
                        Artifact:    name,
                        ComponentID: c.ID,
                        Rect:        rectStr(c),
                        Severity:    "high",
                        Category:    "accessibility",
                        Rule:        "A11y.ColorContrast",
                        WCAG:        "1.4.3 Contrast (Minimum)",
                        Impact:      "Low contrast reduces readability, especially for users with low vision.",
                        Remediation: "Adjust text or background color to meet or exceed the required contrast ratio.",
                        Confidence:  0.9,
                    })
                }
            }
        }
        // Page-level sanity check for theme contrast
        if a.PrimaryFG != "" && a.PrimaryBG != "" {
            if ratio, ok := contrastRatio(a.PrimaryFG, a.PrimaryBG); ok {
                if ratio < 4.5 && (theme == "light" || theme == "dark" || theme == "") {
                    findings = append(findings, VisualFinding{
                        Title:       "Overall foreground/background contrast may be low",
                        Description: fmt.Sprintf("Global contrast ratio appears to be %.2f:1, below common minimums.", ratio),
                        Artifact:    name,
                        Severity:    "medium",
                        Category:    "accessibility",
                        Rule:        "A11y.GlobalContrast",
                        WCAG:        "1.4.3 Contrast (Minimum)",
                        Impact:      "Global low contrast can reduce legibility across the UI.",
                        Remediation: "Review global palette or theme variables for sufficient contrast.",
                        Confidence:  0.75,
                    })
                }
            }
        }

        // 3) Accessibility: tap target size for interactive components
        for _, c := range a.Components {
            if isInteractiveType(c) || c.Clickable {
                if c.W < minTap || c.H < minTap {
                    findings = append(findings, VisualFinding{
                        Title:       "Touch target smaller than recommended minimum",
                        Description: fmt.Sprintf("Component size %.0fx%.0fpx is below the recommended %.0fx%.0fpx.", c.W, c.H, minTap, minTap),
                        Artifact:    name,
                        ComponentID: c.ID,
                        Rect:        rectStr(c),
                        Severity:    "medium",
                        Category:    "accessibility",
                        Rule:        "A11y.TapTarget",
                        Impact:      "Small touch targets increase error rate and reduce usability on touch devices.",
                        Remediation: fmt.Sprintf("Increase the target size to at least %.0fx%.0fpx or add sufficient padding.", minTap, minTap),
                        Confidence:  0.85,
                    })
                }
            }
        }

        // 4) Layout: overlapping components and minimal spacing
        findings = append(findings, detectOverlaps(name, a.Components)...)
        findings = append(findings, detectTightSpacing(name, a.Components)...)

        // 5) Brand: off-palette colors
        if len(a.BrandColors) > 0 {
            brandSet := make(map[string]struct{}, len(a.BrandColors))
            for _, bc := range a.BrandColors {
                brandSet[strings.ToLower(normalizeHex(bc))] = struct{}{}
            }
            for _, c := range a.Components {
                for _, col := range []string{c.FgColor, c.BgColor} {
                    if col == "" {
                        continue
                    }
                    norm := strings.ToLower(normalizeHex(col))
                    if _, ok := brandSet[norm]; !ok {
                        findings = append(findings, VisualFinding{
                            Title:       "Non-brand color detected",
                            Description: fmt.Sprintf("Color %s is not part of the declared brand palette.", norm),
                            Artifact:    name,
                            ComponentID: c.ID,
                            Rect:        rectStr(c),
                            Severity:    "low",
                            Category:    "brand",
                            Rule:        "Brand.Palette",
                            Impact:      "Off-brand colors can dilute brand consistency and recognition.",
                            Remediation: "Replace with the closest approved brand color from the palette.",
                            Confidence:  0.7,
                        })
                    }
                }
            }
        }

        // 6) Content: potential PII exposure (Brazil + generic)
        findings = append(findings, detectPII(name, a.OCRText)...)

        // 7) Compliance: data capture without consent notice (heuristic)
        if looksLikeSignup(a.OCRText) && !mentionsConsent(a.OCRText) {
            findings = append(findings, VisualFinding{
                Title:       "Sign-up form without visible consent notice",
                Description: "Form appears to collect personal data without referencing terms or privacy notice.",
                Artifact:    name,
                Severity:    "medium",
                Category:    "compliance",
                Rule:        "Compliance.ConsentNotice",
                Impact:      "May not meet user consent expectations or applicable privacy requirements.",
                Remediation: "Add a clear link to Terms and Privacy Policy and a concise consent statement.",
                Confidence:  0.7,
            })
        }
    }

    return findings
}

// -------- LLM augmentation --------

func runLLMVisualAssurance(ctx context.Context, model ChatModel, req VisualAssuranceRequest) ([]VisualFinding, error) {
    sys := buildVisualSystemPrompt(req)
    usr := buildVisualUserPrompt(req)

    resp, err := model.Generate(ctx, []ChatMessage{
        {Role: "system", Content: sys},
        {Role: "user", Content: usr},
    })
    if err != nil {
        return nil, err
    }

    // Try to parse as full result, then as an object with findings, then as an array
    if f, ok := vaParseFindingsFromJSON(resp); ok {
        return f, nil
    }
    // Fallback: attempt largest JSON fragment extraction
    if fragment := vaExtractJSONFragment(resp); fragment != "" {
        if f, ok := vaParseFindingsFromJSON(fragment); ok {
            return f, nil
        }
    }

    return nil, fmt.Errorf("unable to parse LLM response into findings")
}

func buildVisualSystemPrompt(req VisualAssuranceRequest) string {
    minSeverity := vaDefaultSeverity(req.SeverityThreshold)
    sb := &strings.Builder{}
    fmt.Fprintf(sb, "You are a senior visual accessibility and UX auditor.\n")
    fmt.Fprintf(sb, "Assess the provided visual artifacts with a focus on accessibility (WCAG), brand consistency, layout clarity, and content risks.\n")
    if len(req.Guidelines) > 0 {
        fmt.Fprintf(sb, "Apply these guidelines:\n")
        for _, g := range req.Guidelines {
            fmt.Fprintf(sb, "- %s\n", g)
        }
    }
    fmt.Fprintf(sb, "Only report issues with severity '%s' or higher.\n", minSeverity)
    fmt.Fprintf(sb, "Respond in strict JSON: either {\"findings\":[...]} or a top-level array [...].\n")
    fmt.Fprintf(sb, "Each finding must include: title, description, artifact, severity, category, remediation. Include componentId and a rect 'x,y,w,h' when applicable.\n")
    if req.RequestAnnotations {
        fmt.Fprintf(sb, "Where relevant, include a brief 'annotation' hint describing how to visually mark the issue.\n")
    }
    return sb.String()
}

func buildVisualUserPrompt(req VisualAssuranceRequest) string {
    sb := &strings.Builder{}
    if strings.TrimSpace(req.Goal) != "" {
        fmt.Fprintf(sb, "Goal: %s\n\n", req.Goal)
    }
    for _, a := range req.Artifacts {
        fmt.Fprintf(sb, "=== ARTIFACT: %s ===\n", a.Name)
        if a.Description != "" {
            fmt.Fprintf(sb, "Description: %s\n", a.Description)
        }
        fmt.Fprintf(sb, "Theme: %s | Dimensions: %.0fx%.0f | Locale: %s\n", safe(a.Theme, "unspecified"), a.Dimensions[0], a.Dimensions[1], safe(a.Locale, "unspecified"))
        if len(a.BrandColors) > 0 {
            fmt.Fprintf(sb, "BrandColors: %s\n", strings.Join(a.BrandColors, ", "))
        }
        if a.PrimaryFG != "" || a.PrimaryBG != "" {
            fmt.Fprintf(sb, "PrimaryFG: %s | PrimaryBG: %s\n", a.PrimaryFG, a.PrimaryBG)
        }
        if a.OCRText != "" {
            fmt.Fprintf(sb, "OCRText:\n%s\n", a.OCRText)
        }
        if len(a.Components) > 0 {
            fmt.Fprintf(sb, "Components (%d):\n", len(a.Components))
            for _, c := range a.Components {
                fmt.Fprintf(sb, "- id=%s type=%s role=%s label=%q text=%q alt=%q rect=%.0f,%.0f,%.0f,%.0f fg=%s bg=%s font=%.1f clickable=%v\n",
                    safe(c.ID, "-"), safe(c.Type, "-"), safe(c.Role, "-"),
                    c.Label, c.Text, c.Alt, c.X, c.Y, c.W, c.H,
                    safe(c.FgColor, "-"), safe(c.BgColor, "-"), c.FontSize, c.Clickable)
            }
        }
        fmt.Fprintf(sb, "\n")
    }
    return sb.String()
}

// -------- Parsing utilities --------

func vaParseFindingsFromJSON(s string) ([]VisualFinding, bool) {
    // Try full result
    var result VisualAssuranceResult
    if err := json.Unmarshal([]byte(s), &result); err == nil && len(result.Findings) > 0 {
        return result.Findings, true
    }
    // Try object with findings
    var obj map[string]json.RawMessage
    if err := json.Unmarshal([]byte(s), &obj); err == nil {
        if raw, ok := obj["findings"]; ok {
            var f []VisualFinding
            if err := json.Unmarshal(raw, &f); err == nil && len(f) > 0 {
                return f, true
            }
        }
    }
    // Try bare array
    var arr []VisualFinding
    if err := json.Unmarshal([]byte(s), &arr); err == nil && len(arr) > 0 {
        return arr, true
    }
    return nil, false
}

func vaExtractJSONFragment(s string) string {
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

// -------- Post-processing and scoring --------

func vaNormalizeFindings(in []VisualFinding) []VisualFinding {
    out := make([]VisualFinding, len(in))
    copy(out, in)
    for i := range out {
        out[i].Severity = vaNormalizeSeverity(out[i].Severity)
        if out[i].Category == "" {
            out[i].Category = "quality"
        }
        if out[i].ID == "" {
            out[i].ID = vaGenFindingID(out[i])
        }
        if out[i].Confidence <= 0 {
            out[i].Confidence = 0.7
        }
    }
    return out
}

func vaFilterBySeverity(findings []VisualFinding, min string) []VisualFinding {
    minRank := vaSeverityRank(min)
    out := make([]VisualFinding, 0, len(findings))
    for _, f := range findings {
        if vaSeverityRank(f.Severity) >= minRank {
            out = append(out, f)
        }
    }
    return out
}

func vaSortFindings(findings []VisualFinding) {
    sort.SliceStable(findings, func(i, j int) bool {
        // Desc by severity, then confidence, then artifact, then componentID
        if vaSeverityRank(findings[i].Severity) != vaSeverityRank(findings[j].Severity) {
            return vaSeverityRank(findings[i].Severity) > vaSeverityRank(findings[j].Severity)
        }
        if findings[i].Confidence != findings[j].Confidence {
            return findings[i].Confidence > findings[j].Confidence
        }
        if findings[i].Artifact != findings[j].Artifact {
            return findings[i].Artifact < findings[j].Artifact
        }
        if findings[i].ComponentID != findings[j].ComponentID {
            return findings[i].ComponentID < findings[j].ComponentID
        }
        return findings[i].Title < findings[j].Title
    })
}

func vaDedupeFindings(in []VisualFinding) []VisualFinding {
    seen := make(map[string]struct{}, len(in))
    out := make([]VisualFinding, 0, len(in))
    for _, f := range in {
        key := f.Artifact + "|" + f.ComponentID + "|" + f.Title + "|" + f.Rect
        if _, exists := seen[key]; exists {
            continue
        }
        seen[key] = struct{}{}
        out = append(out, f)
    }
    return out
}

func vaComputeQualityScore(findings []VisualFinding) float64 {
    score := 100.0
    for _, f := range findings {
        score -= float64(vaSeverityWeight(f.Severity))
    }
    if score < 0 {
        score = 0
    }
    if score > 100 {
        score = 100
    }
    return score
}

func vaComputeConfidence(all, llm []VisualFinding) float64 {
    base := 0.7
    if len(llm) > 0 {
        base = 0.75
    }
    if len(all) == 0 {
        return base
    }
    var sum float64
    for _, f := range all {
        sum += vaClamp(f.Confidence, 0.4, 0.98)
    }
    avg := sum / float64(len(all))
    return vaClamp((base+avg)/2.0, 0.5, 0.95)
}

func vaSummarize(findings []VisualFinding) string {
    if len(findings) == 0 {
        return "No issues found at or above the configured severity threshold."
    }
    var crit, high, med, low int
    for _, f := range findings {
        switch vaNormalizeSeverity(f.Severity) {
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

// -------- Heuristic helpers --------

func isInteractiveType(c VisualComponent) bool {
    t := strings.ToLower(strings.TrimSpace(c.Type))
    switch t {
    case "button", "link", "input", "checkbox", "radio", "switch", "menu", "icon":
        return true
    default:
        return c.Clickable
    }
}

func rectStr(c VisualComponent) string {
    return fmt.Sprintf("%.0f,%.0f,%.0f,%.0f", c.X, c.Y, c.W, c.H)
}

func detectOverlaps(artifactName string, comps []VisualComponent) []VisualFinding {
    var out []VisualFinding
    for i := 0; i < len(comps); i++ {
        for j := i + 1; j < len(comps); j++ {
            if overlapArea(comps[i], comps[j]) > 0 {
                out = append(out, VisualFinding{
                    Title:       "Overlapping components detected",
                    Description: fmt.Sprintf("Components %s and %s appear to overlap.", safe(comps[i].ID, "#1"), safe(comps[j].ID, "#2")),
                    Artifact:    artifactName,
                    ComponentID: comps[i].ID,
                    Rect:        rectStr(comps[i]),
                    Severity:    "medium",
                    Category:    "layout",
                    Rule:        "Layout.Overlap",
                    Impact:      "Overlaps can cause visual clutter and interaction ambiguity.",
                    Remediation: "Adjust layout positions or z-index to remove overlaps; review constraints.",
                    Confidence:  0.75,
                })
            }
        }
    }
    return out
}

func detectTightSpacing(artifactName string, comps []VisualComponent) []VisualFinding {
    const minGap = 4.0 // px, heuristic minimum comfortable spacing
    var out []VisualFinding
    for i := 0; i < len(comps); i++ {
        for j := i + 1; j < len(comps); j++ {
            gap := minDistance(comps[i], comps[j])
            if gap >= 0 && gap < minGap {
                out = append(out, VisualFinding{
                    Title:       "Insufficient spacing between components",
                    Description: fmt.Sprintf("Components %s and %s are only %.0fpx apart.", safe(comps[i].ID, "#1"), safe(comps[j].ID, "#2"), gap),
                    Artifact:    artifactName,
                    ComponentID: comps[i].ID,
                    Rect:        rectStr(comps[i]),
                    Severity:    "low",
                    Category:    "layout",
                    Rule:        "Layout.Spacing",
                    Impact:      "Tight spacing reduces scannability and increases mis-taps.",
                    Remediation: "Increase spacing to improve readability and touch accuracy.",
                    Confidence:  0.7,
                })
            }
        }
    }
    return out
}

func detectPII(artifactName, text string) []VisualFinding {
    var findings []VisualFinding
    lower := strings.ToLower(text)

    // Email
    emailRe := regexp.MustCompile(`[\w.\-+%]+@[\w.\-]+\.[A-Za-z]{2,}`)
    // Phone (simple)
    phoneRe := regexp.MustCompile(`(\+?\d{1,3}[\s\-\.]?)?(\(?\d{2,3}\)?[\s\-\.]?)?\d{4,5}[\s\-\.]?\d{4}`)
    // Credit card (very broad Luhn-less)
    ccRe := regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
    // CPF (Brazil): 000.000.000-00
    cpfRe := regexp.MustCompile(`\b\d{3}\.\d{3}\.\d{3}-\d{2}\b`)
    // CNPJ (Brazil): 00.000.000/0000-00
    cnpjRe := regexp.MustCompile(`\b\d{2}\.\d{3}\.\d{3}/\d{4}-\d{2}\b`)

    piiHits := 0
    if emailRe.FindString(lower) != "" {
        piiHits++
    }
    if phoneRe.FindString(lower) != "" {
        piiHits++
    }
    if ccRe.FindString(lower) != "" {
        piiHits++
    }
    if cpfRe.FindString(lower) != "" || cnpjRe.FindString(lower) != "" {
        piiHits++
    }

    if piiHits > 0 {
        findings = append(findings, VisualFinding{
            Title:       "Potential exposure of personal or sensitive data",
            Description: "Screen text appears to contain email, phone, or identifier patterns.",
            Artifact:    artifactName,
            Severity:    "medium",
            Category:    "content",
            Rule:        "Content.PIIExposure",
            Impact:      "Unnecessary exposure of PII can violate privacy expectations or regulations.",
            Remediation: "Mask or redact sensitive data; show partial values or placeholders in non-secure contexts.",
            Confidence:  0.75,
        })
    }

    return findings
}

func looksLikeSignup(text string) bool {
    l := strings.ToLower(text)
    hints := []string{"sign up", "signup", "register", "cadastre-se", "create account", "create an account", "join now"}
    for _, h := range hints {
        if strings.Contains(l, h) {
            return true
        }
    }
    // Fields that suggest signup
    fieldHints := []string{"email", "senha", "password", "name", "nome"}
    score := 0
    for _, f := range fieldHints {
        if strings.Contains(l, f) {
            score++
        }
    }
    return score >= 2
}

func mentionsConsent(text string) bool {
    l := strings.ToLower(text)
    terms := []string{
        "privacy", "política de privacidade", "política de privacidade",
        "terms", "termos de uso", "termos e condições",
        "consent", "consentimento",
    }
    for _, t := range terms {
        if strings.Contains(l, t) {
            return true
        }
    }
    return false
}

// -------- Color and geometry utilities --------

func normalizeHex(h string) string {
    h = strings.TrimSpace(h)
    if strings.HasPrefix(h, "#") {
        h = h[1:]
    }
    if len(h) == 3 {
        // expand #RGB -> #RRGGBB
        return strings.ToLower(fmt.Sprintf("#%c%c%c%c%c%c", h[0], h[0], h[1], h[1], h[2], h[2]))
    }
    if len(h) == 6 {
        return strings.ToLower("#" + h)
    }
    // try to parse like rgb(255,255,255)
    if strings.HasPrefix(strings.ToLower(h), "rgb(") && strings.HasSuffix(h, ")") {
        inside := h[4 : len(h)-1]
        parts := strings.Split(inside, ",")
        if len(parts) == 3 {
            r, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
            g, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
            b, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
            return fmt.Sprintf("#%02x%02x%02x", clampInt(r, 0, 255), clampInt(g, 0, 255), clampInt(b, 0, 255))
        }
    }
    return strings.ToLower(h)
}

func hexToRGB(hex string) (float64, float64, float64, bool) {
    h := normalizeHex(hex)
    if len(h) != 7 || h[0] != '#' {
        return 0, 0, 0, false
    }
    r, err1 := strconv.ParseInt(h[1:3], 16, 64)
    g, err2 := strconv.ParseInt(h[3:5], 16, 64)
    b, err3 := strconv.ParseInt(h[5:7], 16, 64)
    if err1 != nil || err2 != nil || err3 != nil {
        return 0, 0, 0, false
    }
    return float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0, true
}

// Relative luminance per WCAG
func relativeLuminance(r, g, b float64) float64 {
    var conv = func(c float64) float64 {
        if c <= 0.03928 {
            return c / 12.92
        }
        return math.Pow((c+0.055)/1.055, 2.4)
    }
    R := conv(r)
    G := conv(g)
    B := conv(b)
    return 0.2126*R + 0.7152*G + 0.0722*B
}

// Contrast ratio per WCAG: (L1+0.05)/(L2+0.05)
func contrastRatio(fg, bg string) (float64, bool) {
    fr, fgG, fb, ok1 := hexToRGB(fg)
    br, bgG, bb, ok2 := hexToRGB(bg)
    if !ok1 || !ok2 {
        return 0, false
    }
    L1 := relativeLuminance(fr, fgG, fb)
    L2 := relativeLuminance(br, bgG, bb)
    if L1 < L2 {
        L1, L2 = L2, L1
    }
    return (L1 + 0.05) / (L2 + 0.05), true
}

func overlapArea(a, b VisualComponent) float64 {
    ax2 := a.X + a.W
    ay2 := a.Y + a.H
    bx2 := b.X + b.W
    by2 := b.Y + b.H
    ix := math.Max(0, math.Min(ax2, bx2)-math.Max(a.X, b.X))
    iy := math.Max(0, math.Min(ay2, by2)-math.Max(a.Y, b.Y))
    return ix * iy
}

func minDistance(a, b VisualComponent) float64 {
    // Distance between axis-aligned rectangles (0 if overlapping)
    // Compute horizontal gap
    var dx float64
    if a.X+a.W < b.X {
        dx = b.X - (a.X + a.W)
    } else if b.X+b.W < a.X {
        dx = a.X - (b.X + b.W)
    } else {
        dx = 0
    }
    // Compute vertical gap
    var dy float64
    if a.Y+a.H < b.Y {
        dy = b.Y - (a.Y + a.H)
    } else if b.Y+b.H < a.Y {
        dy = a.Y - (b.Y + b.H)
    } else {
        dy = 0
    }
    if dx == 0 && dy == 0 {
        return 0 // overlap
    }
    // If they are diagonally apart, report the minimal edge-to-edge distance
    if dx > 0 && dy > 0 {
        // Euclidean distance between nearest corners
        return math.Hypot(dx, dy)
    }
    // Otherwise the gap is along one axis
    if dx > 0 {
        return dx
    }
    return dy
}

// -------- Severity, IDs, and misc utilities --------

func vaDefaultSeverity(s string) string {
    switch strings.ToLower(strings.TrimSpace(s)) {
    case "low", "medium", "high", "critical":
        return strings.ToLower(s)
    default:
        return "low"
    }
}

func vaNormalizeSeverity(s string) string {
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

func vaSeverityWeight(s string) int {
    switch vaNormalizeSeverity(s) {
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

func vaSeverityRank(s string) int {
    switch vaNormalizeSeverity(s) {
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

func vaGenFindingID(f VisualFinding) string {
    h := fnv.New64a()
    _, _ = h.Write([]byte(f.Artifact))
    _, _ = h.Write([]byte(f.Title))
    _, _ = h.Write([]byte(f.ComponentID))
    _, _ = h.Write([]byte(f.Rect))
    return fmt.Sprintf("V%08x", h.Sum64())
}

func vaClamp(v, lo, hi float64) float64 {
    if v < lo {
        return lo
    }
    if v > hi {
        return hi
    }
    return v
}

func clampInt(v, lo, hi int) int {
    if v < lo {
        return lo
    }
    if v > hi {
        return hi
    }
    return v
}

func safe(s, fallback string) string {
    if strings.TrimSpace(s) == "" {
        return fallback
    }
    return s
}
