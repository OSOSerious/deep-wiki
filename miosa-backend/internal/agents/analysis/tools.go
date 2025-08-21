package analysis

import (
	"context"
	"fmt"
	"strings"

	"github.com/sormind/OSA/miosa-backend/internal/tools"
)

// AnalysisTools provides tools specific to the analysis agent
type AnalysisTools struct {
	fileSystem  *tools.FileSystemTool
	search      *tools.SearchTool
	codeAnalyzer *tools.CodeAnalyzerTool
	schemaGen   *tools.SchemaGeneratorTool
}

// NewAnalysisTools creates tools for the analysis agent
func NewAnalysisTools() *AnalysisTools {
	return &AnalysisTools{
		fileSystem:  tools.NewFileSystemTool(),
		search:      tools.NewSearchTool(),
		codeAnalyzer: tools.NewCodeAnalyzerTool(),
		schemaGen:   tools.NewSchemaGeneratorTool(),
	}
}

// AnalyzeRequirements analyzes requirements and breaks them down
func (t *AnalysisTools) AnalyzeRequirements(ctx context.Context, requirements string) (map[string]interface{}, error) {
	// Use search tool to find similar implementations
	searchResult, err := t.search.Execute(ctx, map[string]interface{}{
		"query": requirements,
		"type":  "documentation",
	})
	if err != nil {
		return nil, err
	}
	
	// Break down requirements into components
	components := t.breakdownRequirements(requirements)
	
	return map[string]interface{}{
		"original":        requirements,
		"components":      components,
		"similar_found":   searchResult,
		"complexity":      t.assessComplexity(components),
		"estimated_effort": t.estimateEffort(components),
	}, nil
}

// AnalyzeCodebase analyzes existing codebase for patterns
func (t *AnalysisTools) AnalyzeCodebase(ctx context.Context, path string) (map[string]interface{}, error) {
	// List files in the path
	files, err := t.fileSystem.Execute(ctx, map[string]interface{}{
		"operation": "list",
		"path":      path,
	})
	if err != nil {
		return nil, err
	}
	
	fileList := files.([]string)
	analysis := make(map[string]interface{})
	
	// Analyze each file
	for _, file := range fileList {
		if strings.HasSuffix(file, ".go") || strings.HasSuffix(file, ".js") || strings.HasSuffix(file, ".py") {
			content, err := t.fileSystem.Execute(ctx, map[string]interface{}{
				"operation": "read",
				"path":      fmt.Sprintf("%s/%s", path, file),
			})
			if err != nil {
				continue
			}
			
			codeAnalysis, _ := t.codeAnalyzer.Execute(ctx, map[string]interface{}{
				"code":     content,
				"language": t.detectLanguage(file),
			})
			
			analysis[file] = codeAnalysis
		}
	}
	
	return map[string]interface{}{
		"path":          path,
		"total_files":   len(fileList),
		"analyzed":      len(analysis),
		"file_analysis": analysis,
		"patterns":      t.extractPatterns(analysis),
	}, nil
}

// GenerateDataModel generates data models based on requirements
func (t *AnalysisTools) GenerateDataModel(ctx context.Context, entities []string) (map[string]interface{}, error) {
	models := make(map[string]interface{})
	
	for _, entity := range entities {
		// Generate database schema
		dbSchema, _ := t.schemaGen.Execute(ctx, map[string]interface{}{
			"type": "database",
			"name": entity,
		})
		
		// Generate API schema
		apiSchema, _ := t.schemaGen.Execute(ctx, map[string]interface{}{
			"type": "api",
			"name": entity,
		})
		
		models[entity] = map[string]interface{}{
			"database": dbSchema,
			"api":      apiSchema,
		}
	}
	
	return models, nil
}

// Helper methods

func (t *AnalysisTools) breakdownRequirements(requirements string) []map[string]interface{} {
	// Simple breakdown - in production, use NLP
	lines := strings.Split(requirements, "\n")
	components := []map[string]interface{}{}
	
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			components = append(components, map[string]interface{}{
				"id":          i + 1,
				"description": line,
				"type":        t.classifyRequirement(line),
				"priority":    t.assessPriority(line),
			})
		}
	}
	
	return components
}

func (t *AnalysisTools) classifyRequirement(req string) string {
	req = strings.ToLower(req)
	
	if strings.Contains(req, "api") || strings.Contains(req, "endpoint") {
		return "api"
	}
	if strings.Contains(req, "database") || strings.Contains(req, "data") {
		return "data"
	}
	if strings.Contains(req, "ui") || strings.Contains(req, "interface") {
		return "ui"
	}
	if strings.Contains(req, "auth") || strings.Contains(req, "security") {
		return "security"
	}
	
	return "general"
}

func (t *AnalysisTools) assessPriority(req string) string {
	req = strings.ToLower(req)
	
	if strings.Contains(req, "must") || strings.Contains(req, "critical") {
		return "high"
	}
	if strings.Contains(req, "should") || strings.Contains(req, "important") {
		return "medium"
	}
	
	return "low"
}

func (t *AnalysisTools) assessComplexity(components []map[string]interface{}) string {
	if len(components) < 5 {
		return "simple"
	}
	if len(components) < 15 {
		return "moderate"
	}
	return "complex"
}

func (t *AnalysisTools) estimateEffort(components []map[string]interface{}) string {
	hours := len(components) * 4 // Simple estimate
	
	if hours < 20 {
		return fmt.Sprintf("%d hours", hours)
	}
	
	days := hours / 8
	return fmt.Sprintf("%d days", days)
}

func (t *AnalysisTools) detectLanguage(filename string) string {
	if strings.HasSuffix(filename, ".go") {
		return "go"
	}
	if strings.HasSuffix(filename, ".js") || strings.HasSuffix(filename, ".jsx") {
		return "javascript"
	}
	if strings.HasSuffix(filename, ".py") {
		return "python"
	}
	if strings.HasSuffix(filename, ".java") {
		return "java"
	}
	
	return "unknown"
}

func (t *AnalysisTools) extractPatterns(analysis map[string]interface{}) []string {
	patterns := []string{}
	
	// Extract common patterns from analysis
	// This is simplified - in production, use more sophisticated pattern recognition
	
	hasAPI := false
	hasDatabase := false
	hasTests := false
	
	for filename := range analysis {
		if strings.Contains(filename, "api") || strings.Contains(filename, "handler") {
			hasAPI = true
		}
		if strings.Contains(filename, "model") || strings.Contains(filename, "schema") {
			hasDatabase = true
		}
		if strings.Contains(filename, "test") {
			hasTests = true
		}
	}
	
	if hasAPI {
		patterns = append(patterns, "REST API pattern detected")
	}
	if hasDatabase {
		patterns = append(patterns, "Database layer pattern detected")
	}
	if hasTests {
		patterns = append(patterns, "Test coverage present")
	}
	
	return patterns
}
