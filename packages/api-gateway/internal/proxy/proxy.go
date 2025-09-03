package proxy

import (
    "bytes"
    "io"
    "net/http"
    "os"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)

var logger = logrus.New()

// ServiceURLs holds the URLs for backend services
type ServiceURLs struct {
    WorkflowAPI      string
    LLMRouter        string
    AgentOrchestrator string
    MetaPromptEngine  string
    Parser           string
}

// ProxyHandler handles proxying requests to backend services
type ProxyHandler struct {
    urls       ServiceURLs
    httpClient *http.Client
}

// NewProxyHandler creates a new proxy handler with service URLs from environment
func NewProxyHandler() *ProxyHandler {
    urls := ServiceURLs{
        WorkflowAPI:      getEnvOrDefault("WORKFLOW_API_URL", "http://workflow-api.temporal.svc.cluster.local:8080"),
        LLMRouter:        getEnvOrDefault("LLM_ROUTER_URL", "http://llm-router.quantumlayer.svc.cluster.local:8080"),
        AgentOrchestrator: getEnvOrDefault("AGENT_ORCHESTRATOR_URL", "http://agent-orchestrator.quantumlayer.svc.cluster.local:8083"),
        MetaPromptEngine:  getEnvOrDefault("META_PROMPT_ENGINE_URL", "http://meta-prompt-engine.quantumlayer.svc.cluster.local:8085"),
        Parser:           getEnvOrDefault("PARSER_URL", "http://parser.quantumlayer.svc.cluster.local:8086"),
    }

    // Create HTTP client with timeouts
    client := &http.Client{
        Timeout: 60 * time.Second, // 60 second timeout for workflow operations
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
        },
    }

    logger.WithFields(logrus.Fields{
        "workflow_api":       urls.WorkflowAPI,
        "llm_router":        urls.LLMRouter,
        "agent_orchestrator": urls.AgentOrchestrator,
        "meta_prompt_engine": urls.MetaPromptEngine,
        "parser":            urls.Parser,
    }).Info("Initialized proxy handler with service URLs")

    return &ProxyHandler{
        urls:       urls,
        httpClient: client,
    }
}

// ProxyToWorkflow proxies workflow generation requests
func (p *ProxyHandler) ProxyToWorkflow(c *gin.Context) {
    // Read request body
    body, err := io.ReadAll(c.Request.Body)
    if err != nil {
        logger.WithError(err).Error("Failed to read request body")
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
        return
    }

    // Determine the endpoint based on the path
    endpoint := p.urls.WorkflowAPI + c.Param("path")
    if endpoint == p.urls.WorkflowAPI {
        endpoint = p.urls.WorkflowAPI + "/api/v1/workflows/generate"
    }

    logger.WithFields(logrus.Fields{
        "endpoint": endpoint,
        "method":   c.Request.Method,
    }).Info("Proxying request to workflow API")

    // Create new request
    req, err := http.NewRequest(c.Request.Method, endpoint, bytes.NewReader(body))
    if err != nil {
        logger.WithError(err).Error("Failed to create proxy request")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
        return
    }

    // Copy headers
    for key, values := range c.Request.Header {
        for _, value := range values {
            req.Header.Add(key, value)
        }
    }

    // Execute request
    resp, err := p.httpClient.Do(req)
    if err != nil {
        logger.WithError(err).WithField("endpoint", endpoint).Error("Failed to proxy request")
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "error": "Service unavailable",
            "service": "workflow-api",
            "details": err.Error(),
        })
        return
    }
    defer resp.Body.Close()

    // Read response
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        logger.WithError(err).Error("Failed to read response body")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
        return
    }

    // Forward response headers
    for key, values := range resp.Header {
        for _, value := range values {
            c.Header(key, value)
        }
    }

    // Return response
    c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// ProxyToWorkflowExtended proxies extended workflow generation requests
func (p *ProxyHandler) ProxyToWorkflowExtended(c *gin.Context) {
    // Read request body
    body, err := io.ReadAll(c.Request.Body)
    if err != nil {
        logger.WithError(err).Error("Failed to read request body")
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
        return
    }

    endpoint := p.urls.WorkflowAPI + "/api/v1/workflows/generate-extended"
    
    logger.WithFields(logrus.Fields{
        "endpoint": endpoint,
        "method":   "POST",
    }).Info("Proxying extended workflow request")

    // Create new request
    req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
    if err != nil {
        logger.WithError(err).Error("Failed to create proxy request")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
        return
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    
    // Copy other headers
    for key, values := range c.Request.Header {
        if key != "Content-Length" && key != "Host" {
            for _, value := range values {
                req.Header.Add(key, value)
            }
        }
    }

    // Execute request
    resp, err := p.httpClient.Do(req)
    if err != nil {
        logger.WithError(err).WithField("endpoint", endpoint).Error("Failed to proxy extended workflow request")
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "error": "Service unavailable",
            "service": "workflow-api",
            "details": err.Error(),
        })
        return
    }
    defer resp.Body.Close()

    // Read response
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        logger.WithError(err).Error("Failed to read response body")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
        return
    }

    // Forward response
    c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// ProxyToLLMRouter proxies requests to LLM Router
func (p *ProxyHandler) ProxyToLLMRouter(c *gin.Context) {
    p.proxyToService(c, p.urls.LLMRouter, "llm-router")
}

// ProxyToAgentOrchestrator proxies requests to Agent Orchestrator
func (p *ProxyHandler) ProxyToAgentOrchestrator(c *gin.Context) {
    p.proxyToService(c, p.urls.AgentOrchestrator, "agent-orchestrator")
}

// ProxyToMetaPromptEngine proxies requests to Meta Prompt Engine
func (p *ProxyHandler) ProxyToMetaPromptEngine(c *gin.Context) {
    p.proxyToService(c, p.urls.MetaPromptEngine, "meta-prompt-engine")
}

// ProxyToParser proxies requests to Parser service
func (p *ProxyHandler) ProxyToParser(c *gin.Context) {
    p.proxyToService(c, p.urls.Parser, "parser")
}

// Generic proxy function for services
func (p *ProxyHandler) proxyToService(c *gin.Context, baseURL, serviceName string) {
    // Read request body
    body, err := io.ReadAll(c.Request.Body)
    if err != nil {
        logger.WithError(err).Error("Failed to read request body")
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
        return
    }

    // Build endpoint
    path := c.Param("path")
    endpoint := baseURL + path

    logger.WithFields(logrus.Fields{
        "service":  serviceName,
        "endpoint": endpoint,
        "method":   c.Request.Method,
    }).Info("Proxying request")

    // Create new request
    req, err := http.NewRequest(c.Request.Method, endpoint, bytes.NewReader(body))
    if err != nil {
        logger.WithError(err).Error("Failed to create proxy request")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
        return
    }

    // Copy headers
    for key, values := range c.Request.Header {
        if key != "Content-Length" && key != "Host" {
            for _, value := range values {
                req.Header.Add(key, value)
            }
        }
    }

    // Execute request
    resp, err := p.httpClient.Do(req)
    if err != nil {
        logger.WithError(err).WithFields(logrus.Fields{
            "service":  serviceName,
            "endpoint": endpoint,
        }).Error("Failed to proxy request")
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "error":   "Service unavailable",
            "service": serviceName,
            "details": err.Error(),
        })
        return
    }
    defer resp.Body.Close()

    // Read response
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        logger.WithError(err).Error("Failed to read response body")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
        return
    }

    // Forward response
    c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// CheckServiceHealth checks if a service is healthy
func (p *ProxyHandler) CheckServiceHealth(serviceURL string) bool {
    resp, err := p.httpClient.Get(serviceURL + "/health")
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    return resp.StatusCode == http.StatusOK
}

// GetServiceStatus returns the status of all backend services
func (p *ProxyHandler) GetServiceStatus(c *gin.Context) {
    status := gin.H{
        "platform": "QuantumLayer",
        "version":  "2.0.0",
        "services": gin.H{
            "workflow-api":       p.checkHealth(p.urls.WorkflowAPI),
            "llm-router":        p.checkHealth(p.urls.LLMRouter),
            "agent-orchestrator": p.checkHealth(p.urls.AgentOrchestrator),
            "meta-prompt-engine": p.checkHealth(p.urls.MetaPromptEngine),
            "parser":            p.checkHealth(p.urls.Parser),
        },
    }

    c.JSON(http.StatusOK, status)
}

func (p *ProxyHandler) checkHealth(serviceURL string) string {
    if p.CheckServiceHealth(serviceURL) {
        return "healthy"
    }
    return "unhealthy"
}

func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}