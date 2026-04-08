package ecs

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// ClusterInfo holds summary data for an ECS cluster.
type ClusterInfo struct {
	Name         string
	ARN          string
	Status       string
	RunningTasks int
	PendingTasks int
}

// ServiceInfo holds summary data for an ECS service.
type ServiceInfo struct {
	Name           string
	ARN            string
	Status         string
	DesiredCount   int
	RunningCount   int
	LastDeployment string // relative time, e.g. "2h ago"
}

// ClientOptions configures the ECS client.
type ClientOptions struct {
	Profile string
	Region  string
}

// Client wraps the AWS ECS API.
type Client struct {
	api *ecs.Client
}

// NewClient creates a new ECS client configured with the given options.
func NewClient(ctx context.Context, opts ClientOptions) (*Client, error) {
	var cfgOpts []func(*awsconfig.LoadOptions) error

	if opts.Region != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithRegion(opts.Region))
	}
	if opts.Profile != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithSharedConfigProfile(opts.Profile))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Client{api: ecs.NewFromConfig(cfg)}, nil
}

// ListClusters returns all ECS clusters with their details.
func (c *Client) ListClusters(ctx context.Context) ([]ClusterInfo, error) {
	listOut, err := c.api.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, fmt.Errorf("list clusters: %w", err)
	}

	if len(listOut.ClusterArns) == 0 {
		return nil, nil
	}

	descOut, err := c.api.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: listOut.ClusterArns,
	})
	if err != nil {
		return nil, fmt.Errorf("describe clusters: %w", err)
	}

	var clusters []ClusterInfo
	for _, cl := range descOut.Clusters {
		clusters = append(clusters, ClusterInfo{
			Name:         deref(cl.ClusterName),
			ARN:          deref(cl.ClusterArn),
			Status:       deref(cl.Status),
			RunningTasks: int(cl.RunningTasksCount),
			PendingTasks: int(cl.PendingTasksCount),
		})
	}
	return clusters, nil
}

// ListServices returns all services in the given cluster.
func (c *Client) ListServices(ctx context.Context, cluster string) ([]ServiceInfo, error) {
	var serviceArns []string
	var nextToken *string

	for {
		listOut, err := c.api.ListServices(ctx, &ecs.ListServicesInput{
			Cluster:   &cluster,
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("list services: %w", err)
		}
		serviceArns = append(serviceArns, listOut.ServiceArns...)
		nextToken = listOut.NextToken
		if nextToken == nil {
			break
		}
	}

	if len(serviceArns) == 0 {
		return nil, nil
	}

	// DescribeServices supports max 10 at a time
	var services []ServiceInfo
	for i := 0; i < len(serviceArns); i += 10 {
		end := i + 10
		if end > len(serviceArns) {
			end = len(serviceArns)
		}
		descOut, err := c.api.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  &cluster,
			Services: serviceArns[i:end],
		})
		if err != nil {
			return nil, fmt.Errorf("describe services: %w", err)
		}
		for _, svc := range descOut.Services {
			info := ServiceInfo{
				Name:         deref(svc.ServiceName),
				ARN:          deref(svc.ServiceArn),
				Status:       deref(svc.Status),
				DesiredCount: int(svc.DesiredCount),
				RunningCount: int(svc.RunningCount),
			}
			if len(svc.Deployments) > 0 {
				info.LastDeployment = relativeTime(svc.Deployments[0].UpdatedAt)
			}
			services = append(services, info)
		}
	}
	return services, nil
}

// TaskDefInfo holds summary data for a task definition revision.
type TaskDefInfo struct {
	ARN      string
	Family   string
	Revision int
	Status   string
	CPU      string
	Memory   string
}

// GetServiceTaskDefFamily returns the task definition family used by a service.
func (c *Client) GetServiceTaskDefFamily(ctx context.Context, cluster, service string) (string, string, error) {
	descOut, err := c.api.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &cluster,
		Services: []string{service},
	})
	if err != nil {
		return "", "", fmt.Errorf("describe service: %w", err)
	}
	if len(descOut.Services) == 0 {
		return "", "", fmt.Errorf("service %q not found", service)
	}
	taskDefARN := deref(descOut.Services[0].TaskDefinition)
	// Extract family from ARN (arn:aws:ecs:region:account:task-definition/family:revision)
	return taskDefARN, extractFamily(taskDefARN), nil
}

// ListTaskDefRevisions returns recent task definition revisions for a family.
func (c *Client) ListTaskDefRevisions(ctx context.Context, family string, maxResults int) ([]TaskDefInfo, error) {
	listOut, err := c.api.ListTaskDefinitions(ctx, &ecs.ListTaskDefinitionsInput{
		FamilyPrefix: &family,
		Sort:         ecstypes.SortOrderDesc,
		MaxResults:   intPtr(int32(maxResults)),
	})
	if err != nil {
		return nil, fmt.Errorf("list task definitions: %w", err)
	}

	if len(listOut.TaskDefinitionArns) == 0 {
		return nil, nil
	}

	var defs []TaskDefInfo
	for _, arn := range listOut.TaskDefinitionArns {
		descOut, err := c.api.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: &arn,
		})
		if err != nil {
			continue
		}
		td := descOut.TaskDefinition
		defs = append(defs, TaskDefInfo{
			ARN:      deref(td.TaskDefinitionArn),
			Family:   deref(td.Family),
			Revision: int(td.Revision),
			Status:   string(td.Status),
			CPU:      deref(td.Cpu),
			Memory:   deref(td.Memory),
		})
	}
	return defs, nil
}

// DeployWithTaskDef updates the service with a specific task definition.
func (c *Client) DeployWithTaskDef(ctx context.Context, cluster, service, taskDefARN string) error {
	_, err := c.api.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:            &cluster,
		Service:            &service,
		TaskDefinition:     &taskDefARN,
		ForceNewDeployment: true,
	})
	if err != nil {
		return fmt.Errorf("deploy with task def: %w", err)
	}
	return nil
}

// ForceNewDeployment triggers a force new deployment on the given service.
func (c *Client) ForceNewDeployment(ctx context.Context, cluster, service string) error {
	_, err := c.api.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:            &cluster,
		Service:            &service,
		ForceNewDeployment: true,
	})
	if err != nil {
		return fmt.Errorf("force new deployment: %w", err)
	}
	return nil
}

func extractFamily(arn string) string {
	// ARN format: arn:aws:ecs:region:account:task-definition/family:revision
	parts := strings.Split(arn, "/")
	if len(parts) < 2 {
		return arn
	}
	familyRev := parts[len(parts)-1]
	colonIdx := strings.LastIndex(familyRev, ":")
	if colonIdx < 0 {
		return familyRev
	}
	return familyRev[:colonIdx]
}

func intPtr(i int32) *int32 { return &i }

// WaitForStable waits until the service reaches a stable state.
func (c *Client) WaitForStable(ctx context.Context, cluster, service string, timeout time.Duration) error {
	waiter := ecs.NewServicesStableWaiter(c.api)
	return waiter.Wait(ctx, &ecs.DescribeServicesInput{
		Cluster:  &cluster,
		Services: []string{service},
	}, timeout)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func relativeTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	d := time.Since(*t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		days := int(math.Floor(d.Hours() / 24))
		return fmt.Sprintf("%dd ago", days)
	}
}

// ServiceEvent represents an ECS service event.
type ServiceEvent = ecstypes.ServiceEvent
