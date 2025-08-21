package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ChainExecution represents a multi-agent task chain
type ChainExecution struct {
	ID          uuid.UUID
	Task        Task
	Agents      []AgentType
	Results     []*Result
	StartTime   time.Time
	EndTime     time.Time
	TotalMS     int64
	Success     bool
	FinalOutput string
}

// ExecuteChain runs a task through multiple agents in sequence
func (o *Orchestrator) ExecuteChain(ctx context.Context, task Task, agentChain []AgentType) (*ChainExecution, error) {
	chainID := uuid.New()
	startTime := time.Now()
	
	o.logger.Info("Starting agent chain execution",
		zap.String("chain_id", chainID.String()),
		zap.String("task", task.Input),
		zap.Int("agents", len(agentChain)))
	
	chain := &ChainExecution{
		ID:        chainID,
		Task:      task,
		Agents:    agentChain,
		Results:   make([]*Result, 0),
		StartTime: startTime,
	}
	
	// Current task that gets passed between agents
	currentTask := task
	var lastResult *Result
	
	// Execute through each agent in the chain
	for i, agentType := range agentChain {
		o.logger.Info("Chain executing agent",
			zap.String("chain_id", chainID.String()),
			zap.Int("step", i+1),
			zap.String("agent", string(agentType)))
		
		// Get the agent
		agent, err := Get(agentType)
		if err != nil {
			o.logger.Error("Failed to get agent",
				zap.String("agent_type", string(agentType)),
				zap.Error(err))
			continue
		}
		
		// If we have a previous result, use its output as input
		if lastResult != nil && lastResult.Output != "" {
			// Enrich the task with previous agent's output
			currentTask.Input = fmt.Sprintf("Previous analysis:\n%s\n\nNow, %s", 
				truncateOutput(lastResult.Output, 500),
				currentTask.Input)
			
			// Add context from previous execution
			if currentTask.Context == nil {
				currentTask.Context = &TaskContext{}
			}
			currentTask.Context.History = append(currentTask.Context.History, Message{
				Role:      "assistant",
				Content:   lastResult.Output,
				Timestamp: time.Now(),
			})
		}
		
		// Execute with the agent
		agentStart := time.Now()
		result, err := agent.Execute(ctx, currentTask)
		if err != nil {
			o.logger.Error("Agent execution failed",
				zap.String("chain_id", chainID.String()),
				zap.String("agent", string(agentType)),
				zap.Error(err))
			
			result = &Result{
				Success:     false,
				Error:       err,
				ExecutionMS: time.Since(agentStart).Milliseconds(),
			}
		}
		
		// Add agent info to result
		if result.Data == nil {
			result.Data = make(map[string]interface{})
		}
		result.Data["agent"] = string(agentType)
		result.Data["chain_step"] = i + 1
		result.Data["chain_id"] = chainID.String()
		
		// Store the result
		chain.Results = append(chain.Results, result)
		lastResult = result
		
		// Log the handoff
		o.logger.Info("Agent completed in chain",
			zap.String("chain_id", chainID.String()),
			zap.String("agent", string(agentType)),
			zap.Bool("success", result.Success),
			zap.Float64("confidence", result.Confidence),
			zap.Int64("ms", result.ExecutionMS))
		
		// If agent failed and it's critical, stop the chain
		if !result.Success && i < len(agentChain)-1 {
			o.logger.Warn("Agent failed in chain, continuing anyway",
				zap.String("agent", string(agentType)))
		}
		
		// Check if agent suggests a different next agent
		if result.NextAgent != "" && i < len(agentChain)-1 && result.NextAgent != agentChain[i+1] {
			o.logger.Info("Agent suggests different next agent",
				zap.String("suggested", string(result.NextAgent)),
				zap.String("planned", string(agentChain[i+1])))
			// For now, we follow the planned chain
			// In production, we could dynamically adjust
		}
		
		// Small delay between agents to prevent overwhelming
		time.Sleep(100 * time.Millisecond)
	}
	
	// Finalize the chain execution
	chain.EndTime = time.Now()
	chain.TotalMS = chain.EndTime.Sub(chain.StartTime).Milliseconds()
	
	// Determine overall success
	chain.Success = false
	if len(chain.Results) > 0 {
		// Success if at least the last agent succeeded
		chain.Success = chain.Results[len(chain.Results)-1].Success
		chain.FinalOutput = chain.Results[len(chain.Results)-1].Output
	}
	
	// Log chain summary
	o.logger.Info("Chain execution completed",
		zap.String("chain_id", chainID.String()),
		zap.Bool("success", chain.Success),
		zap.Int("agents_executed", len(chain.Results)),
		zap.Int64("total_ms", chain.TotalMS))
	
	// Record for self-improvement
	o.recordChainExecution(chain)
	
	return chain, nil
}

// DetermineAgentChain determines the best agent chain for a task
func (o *Orchestrator) DetermineAgentChain(ctx context.Context, task Task) []AgentType {
	taskLower := strings.ToLower(task.Input)
	
	// Complex development task
	if strings.Contains(taskLower, "build") || strings.Contains(taskLower, "create") || 
	   strings.Contains(taskLower, "implement") {
		return []AgentType{
			AnalysisAgent,     // First analyze requirements
			ArchitectAgent,    // Design the architecture  
			DevelopmentAgent,  // Generate code
			QualityAgent,      // Review and test
		}
	}
	
	// Analysis and planning task
	if strings.Contains(taskLower, "analyze") || strings.Contains(taskLower, "plan") ||
	   strings.Contains(taskLower, "design") {
		return []AgentType{
			AnalysisAgent,   // Analyze requirements
			StrategyAgent,   // Strategic planning
			ArchitectAgent,  // Technical design
		}
	}
	
	// Deployment task
	if strings.Contains(taskLower, "deploy") || strings.Contains(taskLower, "release") {
		return []AgentType{
			AnalysisAgent,    // Analyze what to deploy
			DeploymentAgent,  // Configure deployment
			MonitoringAgent,  // Set up monitoring
		}
	}
	
	// Simple communication task
	if strings.Contains(taskLower, "explain") || strings.Contains(taskLower, "describe") {
		return []AgentType{
			CommunicationAgent, // Just communication
		}
	}
	
	// Default chain for unknown tasks
	return []AgentType{
		AnalysisAgent,    // Start with analysis
		ArchitectAgent,   // Design solution
		DevelopmentAgent, // Implement
	}
}

// recordChainExecution records chain execution for learning
func (o *Orchestrator) recordChainExecution(chain *ChainExecution) {
	// Calculate average confidence across the chain
	totalConfidence := 0.0
	for _, result := range chain.Results {
		totalConfidence += result.Confidence
	}
	avgConfidence := totalConfidence / float64(len(chain.Results))
	
	// Create a workflow pattern
	pattern := &WorkflowPattern{
		ID:           chain.ID,
		TaskType:     chain.Task.Type,
		AgentSequence: chain.Agents,
		SuccessRate:  0.0,
		AvgDuration:  time.Duration(chain.TotalMS) * time.Millisecond,
		Confidence:   avgConfidence,
		LastUpdated:  time.Now(),
	}
	
	if chain.Success {
		pattern.SuccessRate = 1.0
	}
	
	// Store in workflow analyzer
	if o.workflowAnalyzer != nil {
		o.workflowAnalyzer.mu.Lock()
		o.workflowAnalyzer.patterns[chain.ID.String()] = pattern
		o.workflowAnalyzer.mu.Unlock()
	}
	
	o.logger.Info("Recorded chain pattern",
		zap.String("chain_id", chain.ID.String()),
		zap.Float64("avg_confidence", avgConfidence),
		zap.Bool("success", chain.Success))
}

// truncateOutput truncates output to specified length
func truncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + "..."
}