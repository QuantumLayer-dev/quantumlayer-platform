package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type DeploymentRequest struct {
	WorkflowID  string            `json:"workflow_id" binding:"required"`
	CapsuleID   string            `json:"capsule_id" binding:"required"`
	Name        string            `json:"name" binding:"required"`
	Image       string            `json:"image" binding:"required"`
	Port        int32             `json:"port"`
	TTLMinutes  int               `json:"ttl_minutes"`
	Environment map[string]string `json:"environment"`
	Resources   ResourceRequirements `json:"resources"`
}

type ResourceRequirements struct {
	Memory string `json:"memory"`
	CPU    string `json:"cpu"`
}

type DeploymentResponse struct {
	ID         string    `json:"id"`
	WorkflowID string    `json:"workflow_id"`
	CapsuleID  string    `json:"capsule_id"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	Status     string    `json:"status"`
	TTL        int       `json:"ttl_minutes"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type DeploymentManager struct {
	clientset     *kubernetes.Clientset
	namespace     string
	baseURL       string
	deployments   map[string]*DeploymentResponse
}

func NewDeploymentManager() (*DeploymentManager, error) {
	// In-cluster configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig for local development
		log.Printf("Failed to get in-cluster config, trying kubeconfig: %v", err)
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	namespace := os.Getenv("DEPLOYMENT_NAMESPACE")
	if namespace == "" {
		namespace = "quantumlayer-apps"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "apps.quantumlayer.io"
	}

	return &DeploymentManager{
		clientset:   clientset,
		namespace:   namespace,
		baseURL:     baseURL,
		deployments: make(map[string]*DeploymentResponse),
	}, nil
}

func (dm *DeploymentManager) CreateDeployment(ctx context.Context, req DeploymentRequest) (*DeploymentResponse, error) {
	deploymentID := fmt.Sprintf("app-%s", uuid.New().String()[:8])
	
	// Set defaults
	if req.Port == 0 {
		req.Port = 8080
	}
	if req.TTLMinutes == 0 {
		req.TTLMinutes = 60 // Default 1 hour
	}

	// Create namespace if it doesn't exist
	_, err := dm.clientset.CoreV1().Namespaces().Get(ctx, dm.namespace, metav1.GetOptions{})
	if err != nil {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: dm.namespace,
			},
		}
		_, err = dm.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("failed to create namespace: %w", err)
		}
	}

	// Prepare labels
	labels := map[string]string{
		"app":         deploymentID,
		"workflow-id": req.WorkflowID,
		"capsule-id":  req.CapsuleID,
		"managed-by":  "deployment-manager",
	}

	// Prepare environment variables
	envVars := []corev1.EnvVar{}
	for k, v := range req.Environment {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	// Set resource defaults
	memoryLimit := "256Mi"
	cpuLimit := "200m"
	if req.Resources.Memory != "" {
		memoryLimit = req.Resources.Memory
	}
	if req.Resources.CPU != "" {
		cpuLimit = req.Resources.CPU
	}

	// Create Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentID,
			Namespace: dm.namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"ttl":        fmt.Sprintf("%d", req.TTLMinutes),
				"expires-at": time.Now().Add(time.Duration(req.TTLMinutes) * time.Minute).Format(time.RFC3339),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: req.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: req.Port,
									Name:          "http",
								},
							},
							Env: envVars,
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(memoryLimit),
									corev1.ResourceCPU:    resource.MustParse(cpuLimit),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("128Mi"),
									corev1.ResourceCPU:    resource.MustParse("100m"),
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = dm.clientset.AppsV1().Deployments(dm.namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentID,
			Namespace: dm.namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(int(req.Port)),
					Name:       "http",
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	_, err = dm.clientset.CoreV1().Services(dm.namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Create Ingress
	subdomain := fmt.Sprintf("%s.%s", deploymentID, dm.baseURL)
	pathType := networkingv1.PathTypePrefix
	
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentID,
			Namespace: dm.namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
				"kubernetes.io/ingress.class":                "nginx",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: subdomain,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: deploymentID,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = dm.clientset.NetworkingV1().Ingresses(dm.namespace).Create(ctx, ingress, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Warning: Failed to create ingress: %v", err)
		// Continue without ingress - can still use NodePort
	}

	// Create response
	response := &DeploymentResponse{
		ID:         deploymentID,
		WorkflowID: req.WorkflowID,
		CapsuleID:  req.CapsuleID,
		Name:       req.Name,
		URL:        fmt.Sprintf("http://%s", subdomain),
		Status:     "deploying",
		TTL:        req.TTLMinutes,
		ExpiresAt:  time.Now().Add(time.Duration(req.TTLMinutes) * time.Minute),
		CreatedAt:  time.Now(),
	}

	dm.deployments[deploymentID] = response
	
	return response, nil
}

func (dm *DeploymentManager) GetDeployment(ctx context.Context, id string) (*DeploymentResponse, error) {
	if dep, exists := dm.deployments[id]; exists {
		// Update status from kubernetes
		deployment, err := dm.clientset.AppsV1().Deployments(dm.namespace).Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			dep.Status = "unknown"
		} else {
			if deployment.Status.ReadyReplicas > 0 {
				dep.Status = "running"
			} else {
				dep.Status = "pending"
			}
		}
		return dep, nil
	}
	return nil, fmt.Errorf("deployment not found")
}

func (dm *DeploymentManager) DeleteDeployment(ctx context.Context, id string) error {
	// Delete Kubernetes resources
	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	// Delete deployment
	err := dm.clientset.AppsV1().Deployments(dm.namespace).Delete(ctx, id, deleteOptions)
	if err != nil {
		log.Printf("Failed to delete deployment: %v", err)
	}

	// Delete service
	err = dm.clientset.CoreV1().Services(dm.namespace).Delete(ctx, id, deleteOptions)
	if err != nil {
		log.Printf("Failed to delete service: %v", err)
	}

	// Delete ingress
	err = dm.clientset.NetworkingV1().Ingresses(dm.namespace).Delete(ctx, id, deleteOptions)
	if err != nil {
		log.Printf("Failed to delete ingress: %v", err)
	}

	delete(dm.deployments, id)
	return nil
}

// TTL Cleanup worker
func (dm *DeploymentManager) StartTTLCleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				dm.cleanupExpiredDeployments(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (dm *DeploymentManager) cleanupExpiredDeployments(ctx context.Context) {
	for id, dep := range dm.deployments {
		if time.Now().After(dep.ExpiresAt) {
			log.Printf("Cleaning up expired deployment: %s", id)
			err := dm.DeleteDeployment(ctx, id)
			if err != nil {
				log.Printf("Failed to cleanup deployment %s: %v", id, err)
			}
		}
	}
}

func int32Ptr(i int32) *int32 { return &i }

func main() {
	dm, err := NewDeploymentManager()
	if err != nil {
		log.Fatal("Failed to create deployment manager:", err)
	}

	// Start TTL cleanup worker
	ctx := context.Background()
	dm.StartTTLCleanup(ctx)

	// Setup Gin router
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Deploy application
	r.POST("/api/v1/deploy", func(c *gin.Context) {
		var req DeploymentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		response, err := dm.CreateDeployment(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	// Get deployment status
	r.GET("/api/v1/deployments/:id", func(c *gin.Context) {
		id := c.Param("id")
		
		response, err := dm.GetDeployment(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "deployment not found"})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	// Delete deployment
	r.DELETE("/api/v1/deployments/:id", func(c *gin.Context) {
		id := c.Param("id")
		
		err := dm.DeleteDeployment(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "deployment deleted"})
	})

	// List all deployments
	r.GET("/api/v1/deployments", func(c *gin.Context) {
		deployments := []DeploymentResponse{}
		for _, dep := range dm.deployments {
			deployments = append(deployments, *dep)
		}
		c.JSON(http.StatusOK, gin.H{"deployments": deployments})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8087"
	}

	log.Printf("Starting Deployment Manager on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}