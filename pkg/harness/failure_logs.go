package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/harness/harness-mcp/client"
	"github.com/harness/harness-mcp/client/dto"
	"github.com/harness/harness-mcp/cmd/harness-mcp-server/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

// GitInfo contains Git repository information
type GitInfo struct {
	RepoURL    string
	Branch     string
	CommitHash string
}

// GetPipelineFailureLogsTool creates a tool for retrieving failure logs from pipeline executions
func GetPipelineFailureLogsTool(config *config.Config, client *client.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_pipeline_failure_logs",
			mcp.WithDescription("Retrieves failure logs from Harness pipeline executions with Git context"),
			mcp.WithString("workspace_path",
				mcp.Description("Workspace path to extract Git context from"),
			),
			mcp.WithString("execution_id",
				mcp.Description("Optional: Specific execution ID to get logs for"),
			),
			mcp.WithNumber("max_pages",
				mcp.Description("Optional: Maximum number of pages to search through (default: 20)"),
			),
			WithScope(config, true),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			scope, err := fetchScope(config, request, true)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			executionID, err := OptionalParam[string](request, "execution_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			workspacePath, err := OptionalParam[string](request, "workspace_path")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			maxPages, err := OptionalIntParamWithDefault(request, "max_pages", 20)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			var gitInfo *GitInfo
			if workspacePath != "" {
				gitInfo, err = extractGitInfo(workspacePath)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to extract Git info: %v", err)), nil
				}
			}

			failureResponse := &dto.FailureLogResponse{}
			if executionID != "" {
				// If execution ID is provided, get logs directly for that execution
				failureResponse, err = getFailureLogsForExecution(ctx, client, scope, executionID)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to get logs: %v", err)), nil
				}
			} else if gitInfo != nil {
				// Otherwise, search for matching executions
				failureResponse, err = findMatchingExecutionAndGetLogs(ctx, client, scope, gitInfo, maxPages)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to find matching execution: %v", err)), nil
				}
			} else {
				return mcp.NewToolResultError("Either execution_id or workspace_path must be provided"), nil
			}

			// Marshal the response to JSON
			responseJSON, err := json.MarshalIndent(failureResponse, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
			}

			return mcp.NewToolResultText(string(responseJSON)), nil
		}
}

// extractGitInfo extracts Git repository information from a workspace path
func extractGitInfo(workspacePath string) (*GitInfo, error) {
	// Function to run git commands
	runGitCommand := func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = workspacePath
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(output)), nil
	}

	// Get repo URL
	repoURL, err := runGitCommand("config", "--get", "remote.origin.url")
	if err != nil {
		return nil, fmt.Errorf("failed to get repo URL: %w", err)
	}

	// Get branch name
	branch, err := runGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get branch name: %w", err)
	}

	// Get commit hash
	commitHash, err := runGitCommand("rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get commit hash: %w", err)
	}

	return &GitInfo{
		RepoURL:    normalizeGitURL(repoURL),
		Branch:     branch,
		CommitHash: commitHash,
	}, nil
}

// normalizeGitURL normalizes Git URLs for comparison
func normalizeGitURL(url string) string {
	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// Convert SSH URLs to HTTPS format
	sshPattern := regexp.MustCompile(`^git@([^:]+):(.+)$`)
	if matches := sshPattern.FindStringSubmatch(url); len(matches) == 3 {
		url = fmt.Sprintf("https://%s/%s", matches[1], matches[2])
	}

	return url
}

// findMatchingExecutionAndGetLogs finds executions that match the Git context and returns failure logs
func findMatchingExecutionAndGetLogs(ctx context.Context, client *client.Client, scope dto.Scope, gitInfo *GitInfo, maxPages int) (*dto.FailureLogResponse, error) {
	for page := 0; page < maxPages; page++ {
		// List pipeline executions with pagination
		opts := &dto.PipelineExecutionOptions{
			PaginationOptions: dto.PaginationOptions{
				Page: page,
				Size: 5, // Fetch 5 executions per page
			},
		}

		executions, err := client.Pipelines.ListExecutions(ctx, scope, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list executions: %w", err)
		}

		// No more executions to process
		if len(executions.Data.Content) == 0 {
			break
		}

		// Check each execution for matching Git context
		for _, execution := range executions.Data.Content {
			if isMatchingExecution(execution, gitInfo) && execution.Status == "Failed" {
				return getFailureLogsForExecution(ctx, client, scope, execution.PlanExecutionId)
			}
		}

		// Stop pagination if we've reached the last page
		if executions.Data.Last {
			break
		}
	}

	return nil, fmt.Errorf("no matching failed execution found")
}

// isMatchingExecution checks if a pipeline execution matches the Git context
func isMatchingExecution(execution dto.PipelineExecution, gitInfo *GitInfo) bool {
	// Extract CI module info if available
	ciInfo, ok := execution.ModuleInfo["ci"]
	if !ok {
		return false
	}

	// Extract data from the CI module info map
	ciMap, ok := ciInfo.(map[string]interface{})
	if !ok {
		return false
	}

	// Try to match by branch
	branch, _ := ciMap["branch"].(string)
	if branch != "" && branch == gitInfo.Branch {
		return true
	}

	// Try to match by repo URL
	scmDetailsList, ok := ciMap["scmDetailsList"].([]interface{})
	if ok && len(scmDetailsList) > 0 {
		scmDetails, ok := scmDetailsList[0].(map[string]interface{})
		if ok {
			scmURL, _ := scmDetails["scmUrl"].(string)
			if scmURL != "" && strings.Contains(normalizeGitURL(scmURL), gitInfo.RepoURL) {
				return true
			}
		}
	}

	// Check for commits in ciExecutionInfoDTO
	ciExecutionInfo, ok := ciMap["ciExecutionInfoDTO"].(map[string]interface{})
	if ok {
		branch, ok := ciExecutionInfo["branch"].(map[string]interface{})
		if ok {
			commits, ok := branch["commits"].([]interface{})
			if ok && len(commits) > 0 {
				commit, ok := commits[0].(map[string]interface{})
				if ok {
					commitID, _ := commit["id"].(string)
					if commitID == gitInfo.CommitHash {
						return true
					}
				}
			}
		}
	}

	return false
}

// getFailureLogsForExecution retrieves failure logs for a specific execution
func getFailureLogsForExecution(ctx context.Context, client *client.Client, scope dto.Scope, executionID string) (*dto.FailureLogResponse, error) {
	// Get execution details
	execution, err := client.Pipelines.GetExecution(ctx, scope, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution details: %w", err)
	}

	// Check if execution is failed
	if execution.Data.Status != "Failed" {
		return nil, fmt.Errorf("execution is not in Failed state: %s", execution.Data.Status)
	}

	// Get execution graph for more details
	graph, err := client.Pipelines.GetExecutionGraph(ctx, scope, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution graph: %w", err)
	}

	// Extract failed nodes and log keys
	var failedNodes []dto.FailedNodeInfo
	for nodeID, node := range graph.NodeMap {
		if node.Status == "Failed" {
			failureMessage := ""
			// Extract failure message if available
			if stepParams, ok := node.StepParameters["failureInfo"].(map[string]interface{}); ok {
				if message, ok := stepParams["message"].(string); ok {
					failureMessage = message
				}
			}

			// Extract step and stage IDs
			stageID := ""
			stepID := ""
			if baseFqn := strings.Split(node.BaseFqn, "."); len(baseFqn) > 2 {
				stageID = baseFqn[2]
			}
			stepID = node.Identifier

			failedNodes = append(failedNodes, dto.FailedNodeInfo{
				NodeID:        nodeID,
				StageID:       stageID,
				StepID:        stepID,
				FailureMessage: failureMessage,
			})
		}
	}

	// Extract Git context
	gitContext := extractGitContextFromExecution(execution.Data)

	// Get log service token
	token, err := client.Logs.GetLogServiceToken(ctx, scope.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get log service token: %w", err)
	}

	// Prepare response
	response := &dto.FailureLogResponse{
		PipelineID:  execution.Data.PipelineIdentifier,
		ExecutionID: executionID,
		Status:      execution.Data.Status,
		GitContext:  gitContext,
		Failures:    []dto.FailureDetails{},
	}

	// Get logs for each failed node
	for _, failedNode := range failedNodes {
		node := graph.NodeMap[failedNode.NodeID]
		
		// Get log key - first try from executable responses, then fall back to logBaseKey
		var logKey string
		if len(node.ExecutableResponses) > 0 && len(node.ExecutableResponses[0].Async.LogKeys) > 0 {
			// Get the first log key
			for _, key := range node.ExecutableResponses[0].Async.LogKeys {
				logKey = key
				break
			}
		}
		
		// Fall back to logBaseKey if executable responses don't have log keys
		if logKey == "" && node.LogBaseKey != "" {
			logKey = node.LogBaseKey
		}

		if logKey == "" {
			// No log key found, skip this node
			continue
		}

		// Get logs
		logs, err := client.Logs.GetStepLogs(
			ctx,
			scope.AccountID,
			execution.Data.OrgIdentifier,
			execution.Data.ProjectIdentifier,
			execution.Data.PipelineIdentifier,
			logKey,
			token,
		)
		if err != nil {
			logrus.Warnf("Failed to get logs for node %s: %v", failedNode.NodeID, err)
			continue
		}

		// Format logs
		formattedLogs := formatLogs(logs)

		// Add to response
		response.Failures = append(response.Failures, dto.FailureDetails{
			Stage:    failedNode.StageID,
			Step:     failedNode.StepID,
			Message:  failedNode.FailureMessage,
			Logs:     formattedLogs,
		})
	}

	return response, nil
}

// extractGitContextFromExecution extracts Git context from execution data
func extractGitContextFromExecution(execution dto.PipelineExecution) dto.GitContext {
	gitContext := dto.GitContext{}

	// Extract CI module info if available
	ciInfo, ok := execution.ModuleInfo["ci"]
	if !ok {
		return gitContext
	}

	// Extract data from the CI module info map
	ciMap, ok := ciInfo.(map[string]interface{})
	if !ok {
		return gitContext
	}

	// Try to get branch, tag and repo name
	if branch, ok := ciMap["branch"].(string); ok {
		gitContext.Branch = branch
	}
	if tag, ok := ciMap["tag"].(string); ok {
		gitContext.Tag = tag
	}

	// Try to get commit info from ciExecutionInfoDTO
	ciExecutionInfo, ok := ciMap["ciExecutionInfoDTO"].(map[string]interface{})
	if ok {
		branchInfo, ok := ciExecutionInfo["branch"].(map[string]interface{})
		if ok {
			commits, ok := branchInfo["commits"].([]interface{})
			if ok && len(commits) > 0 {
				commit, ok := commits[0].(map[string]interface{})
				if ok {
					if commitHash, ok := commit["id"].(string); ok {
						gitContext.CommitHash = commitHash
					}
					if commitMsg, ok := commit["message"].(string); ok {
						gitContext.CommitMessage = commitMsg
					}
				}
			}
		}
	}

	// Try to get repo URL
	scmDetailsList, ok := ciMap["scmDetailsList"].([]interface{})
	if ok && len(scmDetailsList) > 0 {
		scmDetails, ok := scmDetailsList[0].(map[string]interface{})
		if ok {
			if scmURL, ok := scmDetails["scmUrl"].(string); ok {
				gitContext.RepoURL = scmURL
			}
		}
	}

	return gitContext
}

// formatLogs formats the raw logs for better readability
func formatLogs(rawLogs string) string {
	// Remove ANSI color codes
	ansiPattern := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	logs := ansiPattern.ReplaceAllString(rawLogs, "")

	// Try to parse as JSON log entries
	var formattedLogs strings.Builder
	lines := strings.Split(logs, "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		// Try to parse as JSON
		var logEntry struct {
			Level string    `json:"level"`
			Pos   int       `json:"pos"`
			Out   string    `json:"out"`
			Time  string    `json:"time"`
		}
		
		if err := json.Unmarshal([]byte(line), &logEntry); err == nil && logEntry.Out != "" {
			// Format as timestamp + message
			formattedLogs.WriteString(fmt.Sprintf("[%s] %s", logEntry.Time, logEntry.Out))
			formattedLogs.WriteString("\n")
		} else {
			// Just include the line as is
			formattedLogs.WriteString(line)
			formattedLogs.WriteString("\n")
		}
	}
	
	return formattedLogs.String()
}
