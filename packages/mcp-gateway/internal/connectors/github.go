package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
)

// GitHubConnector handles GitHub API interactions
type GitHubConnector struct {
	client *github.Client
	ctx    context.Context
}

// NewGitHubConnector creates a new GitHub connector
func NewGitHubConnector() *GitHubConnector {
	ctx := context.Background()
	
	// Get GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Println("Warning: GITHUB_TOKEN not set, using unauthenticated client")
		return &GitHubConnector{
			client: github.NewClient(nil),
			ctx:    ctx,
		}
	}
	
	// Create authenticated client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	
	return &GitHubConnector{
		client: github.NewClient(tc),
		ctx:    ctx,
	}
}

// ReadRepositoryRequest represents a request to read a repository
type ReadRepositoryRequest struct {
	URL    string `json:"url"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
}

// RepositoryInfo contains repository information
type RepositoryInfo struct {
	Owner       string                 `json:"owner"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Language    string                 `json:"language"`
	Stars       int                    `json:"stars"`
	Forks       int                    `json:"forks"`
	Branch      string                 `json:"branch"`
	Files       []FileInfo             `json:"files"`
	Structure   map[string]interface{} `json:"structure"`
	Statistics  RepoStatistics         `json:"statistics"`
}

// FileInfo contains file information
type FileInfo struct {
	Path     string `json:"path"`
	Type     string `json:"type"`
	Size     int    `json:"size"`
	Language string `json:"language,omitempty"`
}

// RepoStatistics contains repository statistics
type RepoStatistics struct {
	TotalFiles      int            `json:"total_files"`
	TotalSize       int64          `json:"total_size"`
	Languages       map[string]int `json:"languages"`
	HasTests        bool           `json:"has_tests"`
	HasCI           bool           `json:"has_ci"`
	HasDocumentation bool          `json:"has_documentation"`
}

// ReadRepository reads and analyzes a GitHub repository
func (g *GitHubConnector) ReadRepository(input json.RawMessage) (interface{}, error) {
	var req ReadRepositoryRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}
	
	// Parse GitHub URL
	owner, repo, err := parseGitHubURL(req.URL)
	if err != nil {
		return nil, err
	}
	
	log.Printf("Reading GitHub repository: %s/%s", owner, repo)
	
	// Get repository information
	repoInfo, _, err := g.client.Repositories.Get(g.ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	
	// Default to main/master branch
	branch := req.Branch
	if branch == "" {
		branch = repoInfo.GetDefaultBranch()
	}
	
	// Get repository contents
	files, structure, err := g.getRepositoryContents(owner, repo, branch, req.Path)
	if err != nil {
		return nil, err
	}
	
	// Get language statistics
	languages, _, err := g.client.Repositories.ListLanguages(g.ctx, owner, repo)
	if err != nil {
		log.Printf("Failed to get languages: %v", err)
		languages = map[string]int{}
	}
	
	// Calculate statistics
	stats := g.calculateStatistics(files, languages)
	
	result := RepositoryInfo{
		Owner:       owner,
		Name:        repo,
		Description: repoInfo.GetDescription(),
		Language:    repoInfo.GetLanguage(),
		Stars:       repoInfo.GetStargazersCount(),
		Forks:       repoInfo.GetForksCount(),
		Branch:      branch,
		Files:       files,
		Structure:   structure,
		Statistics:  stats,
	}
	
	return result, nil
}

// CreatePullRequestRequest represents a request to create a pull request
type CreatePullRequestRequest struct {
	Owner       string            `json:"owner"`
	Repo        string            `json:"repo"`
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Head        string            `json:"head"`
	Base        string            `json:"base"`
	Files       map[string]string `json:"files"`
	CommitMsg   string            `json:"commit_message"`
}

// CreatePullRequest creates a new pull request
func (g *GitHubConnector) CreatePullRequest(input json.RawMessage) (interface{}, error) {
	var req CreatePullRequestRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}
	
	log.Printf("Creating pull request in %s/%s", req.Owner, req.Repo)
	
	// Create a new branch if files are provided
	if len(req.Files) > 0 {
		branchName := fmt.Sprintf("quantumlayer-%d", time.Now().Unix())
		if err := g.createBranchWithFiles(req.Owner, req.Repo, branchName, req.Files, req.CommitMsg); err != nil {
			return nil, fmt.Errorf("failed to create branch: %w", err)
		}
		req.Head = branchName
	}
	
	// Create pull request
	newPR := &github.NewPullRequest{
		Title: github.String(req.Title),
		Body:  github.String(req.Body),
		Head:  github.String(req.Head),
		Base:  github.String(req.Base),
	}
	
	pr, _, err := g.client.PullRequests.Create(g.ctx, req.Owner, req.Repo, newPR)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}
	
	return map[string]interface{}{
		"id":     pr.GetNumber(),
		"url":    pr.GetHTMLURL(),
		"title":  pr.GetTitle(),
		"state":  pr.GetState(),
		"created": pr.GetCreatedAt(),
	}, nil
}

// CreateIssueRequest represents a request to create an issue
type CreateIssueRequest struct {
	Owner    string   `json:"owner"`
	Repo     string   `json:"repo"`
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Labels   []string `json:"labels,omitempty"`
	Assignee string   `json:"assignee,omitempty"`
}

// CreateIssue creates a new issue
func (g *GitHubConnector) CreateIssue(input json.RawMessage) (interface{}, error) {
	var req CreateIssueRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}
	
	issueRequest := &github.IssueRequest{
		Title:    github.String(req.Title),
		Body:     github.String(req.Body),
		Labels:   &req.Labels,
	}
	
	if req.Assignee != "" {
		issueRequest.Assignee = github.String(req.Assignee)
	}
	
	issue, _, err := g.client.Issues.Create(g.ctx, req.Owner, req.Repo, issueRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}
	
	return map[string]interface{}{
		"id":     issue.GetNumber(),
		"url":    issue.GetHTMLURL(),
		"title":  issue.GetTitle(),
		"state":  issue.GetState(),
		"created": issue.GetCreatedAt(),
	}, nil
}

// ListRepositories lists user's repositories
func (g *GitHubConnector) ListRepositories(input json.RawMessage) (interface{}, error) {
	var req struct {
		User string `json:"user,omitempty"`
		Org  string `json:"org,omitempty"`
		Type string `json:"type,omitempty"` // all, owner, public, private, member
	}
	
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}
	
	opts := &github.RepositoryListOptions{
		Type: req.Type,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	
	var repos []*github.Repository
	var err error
	
	if req.Org != "" {
		repos, _, err = g.client.Repositories.ListByOrg(g.ctx, req.Org, opts)
	} else if req.User != "" {
		repos, _, err = g.client.Repositories.List(g.ctx, req.User, opts)
	} else {
		// List authenticated user's repos
		repos, _, err = g.client.Repositories.List(g.ctx, "", opts)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	
	result := []map[string]interface{}{}
	for _, repo := range repos {
		result = append(result, map[string]interface{}{
			"name":        repo.GetName(),
			"full_name":   repo.GetFullName(),
			"description": repo.GetDescription(),
			"url":         repo.GetHTMLURL(),
			"language":    repo.GetLanguage(),
			"stars":       repo.GetStargazersCount(),
			"private":     repo.GetPrivate(),
		})
	}
	
	return result, nil
}

// Helper functions

func (g *GitHubConnector) getRepositoryContents(owner, repo, branch, path string) ([]FileInfo, map[string]interface{}, error) {
	if path == "" {
		path = "/"
	}
	
	_, directoryContent, _, err := g.client.Repositories.GetContents(
		g.ctx, owner, repo, path,
		&github.RepositoryContentGetOptions{Ref: branch},
	)
	
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get repository contents: %w", err)
	}
	
	files := []FileInfo{}
	structure := make(map[string]interface{})
	
	for _, content := range directoryContent {
		fileInfo := FileInfo{
			Path: content.GetPath(),
			Type: content.GetType(),
			Size: content.GetSize(),
		}
		
		// Detect language by extension
		if content.GetType() == "file" {
			fileInfo.Language = detectLanguageFromPath(content.GetPath())
		}
		
		files = append(files, fileInfo)
		
		// Build structure map
		parts := strings.Split(content.GetPath(), "/")
		current := structure
		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = content.GetType()
			} else {
				if _, exists := current[part]; !exists {
					current[part] = make(map[string]interface{})
				}
				current = current[part].(map[string]interface{})
			}
		}
	}
	
	return files, structure, nil
}

func (g *GitHubConnector) calculateStatistics(files []FileInfo, languages map[string]int) RepoStatistics {
	stats := RepoStatistics{
		TotalFiles: len(files),
		Languages:  languages,
	}
	
	for _, file := range files {
		stats.TotalSize += int64(file.Size)
		
		// Check for test directories
		if strings.Contains(file.Path, "test") || strings.Contains(file.Path, "spec") {
			stats.HasTests = true
		}
		
		// Check for CI configuration
		if strings.Contains(file.Path, ".github/workflows") || strings.Contains(file.Path, ".gitlab-ci") {
			stats.HasCI = true
		}
		
		// Check for documentation
		if strings.HasSuffix(file.Path, ".md") || strings.Contains(file.Path, "docs/") {
			stats.HasDocumentation = true
		}
	}
	
	return stats
}

func (g *GitHubConnector) createBranchWithFiles(owner, repo, branchName string, files map[string]string, commitMsg string) error {
	// Get default branch
	repoInfo, _, err := g.client.Repositories.Get(g.ctx, owner, repo)
	if err != nil {
		return err
	}
	
	defaultBranch := repoInfo.GetDefaultBranch()
	
	// Get reference of default branch
	ref, _, err := g.client.Git.GetRef(g.ctx, owner, repo, "refs/heads/"+defaultBranch)
	if err != nil {
		return err
	}
	
	// Create new branch
	newRef := &github.Reference{
		Ref: github.String("refs/heads/" + branchName),
		Object: &github.GitObject{
			SHA: ref.Object.SHA,
		},
	}
	
	_, _, err = g.client.Git.CreateRef(g.ctx, owner, repo, newRef)
	if err != nil {
		return err
	}
	
	// Add files to the branch
	for path, content := range files {
		fileContent := &github.RepositoryContentFileOptions{
			Message: github.String(commitMsg),
			Content: []byte(content),
			Branch:  github.String(branchName),
		}
		
		_, _, err = g.client.Repositories.CreateFile(g.ctx, owner, repo, path, fileContent)
		if err != nil {
			log.Printf("Failed to create file %s: %v", path, err)
		}
	}
	
	return nil
}

func parseGitHubURL(url string) (owner, repo string, err error) {
	// Handle various GitHub URL formats
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "github.com/")
	url = strings.TrimSuffix(url, ".git")
	
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub URL format")
	}
	
	return parts[0], parts[1], nil
}

func detectLanguageFromPath(path string) string {
	extensions := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "typescript",
		".java": "java",
		".rb":   "ruby",
		".php":  "php",
		".cs":   "csharp",
		".cpp":  "cpp",
		".c":    "c",
		".rs":   "rust",
		".swift": "swift",
		".kt":   "kotlin",
		".scala": "scala",
		".r":    "r",
		".jl":   "julia",
		".dart": "dart",
		".lua":  "lua",
		".sh":   "bash",
		".yaml": "yaml",
		".json": "json",
		".xml":  "xml",
		".html": "html",
		".css":  "css",
		".sql":  "sql",
	}
	
	for ext, lang := range extensions {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return lang
		}
	}
	
	return "unknown"
}