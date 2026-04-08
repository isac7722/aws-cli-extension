package ecs

import (
	"context"
	"fmt"
	"os"
	"time"

	ecslib "github.com/isac7722/aws-cli-extension/internal/ecs"
	"github.com/isac7722/aws-cli-extension/internal/tui"
	uissm "github.com/isac7722/aws-cli-extension/internal/ui/ssm"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Force a new deployment on an ECS service",
	Long:  `Interactively select a cluster and service, then trigger a force new deployment.`,
	RunE:  runDeploy,
}

var (
	flagCluster string
	flagService string
	flagTaskDef string
	flagWait    bool
)

func init() {
	deployCmd.Flags().StringVar(&flagCluster, "cluster", "", "ECS cluster name (skip selector)")
	deployCmd.Flags().StringVar(&flagService, "service", "", "ECS service name (skip selector)")
	deployCmd.Flags().StringVar(&flagTaskDef, "task-def", "", "Task definition ARN or family:revision (skip selector)")
	deployCmd.Flags().BoolVar(&flagWait, "wait", false, "Wait for service to stabilize after deployment")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	profile := flagProfile
	region := flagRegion

	// 1. Profile selector
	if profile == "" {
		currentProfile := os.Getenv("AWS_PROFILE")
		chosen, err := uissm.RunProfileSelector(currentProfile)
		if err != nil {
			return fmt.Errorf("profile selector: %w", err)
		}
		if chosen == nil {
			return nil // cancelled
		}
		profile = chosen.Name
		if region == "" && chosen.Region != "" {
			region = chosen.Region
		}
	}

	// 2. Region selector
	if region == "" {
		currentRegion := os.Getenv("AWS_REGION")
		if currentRegion == "" {
			currentRegion = os.Getenv("AWS_DEFAULT_REGION")
		}
		chosenRegion, err := uissm.RunRegionSelector(currentRegion)
		if err != nil {
			return fmt.Errorf("region selector: %w", err)
		}
		if chosenRegion == "" {
			return nil // cancelled
		}
		region = chosenRegion
	}

	// Create ECS client
	client, err := ecslib.NewClient(ctx, ecslib.ClientOptions{
		Profile: profile,
		Region:  region,
	})
	if err != nil {
		return fmt.Errorf("ECS client: %w", err)
	}

	cluster := flagCluster
	service := flagService

	// 3. Cluster selector
	if cluster == "" {
		clusters, err := client.ListClusters(ctx)
		if err != nil {
			return fmt.Errorf("list clusters: %w", err)
		}
		if len(clusters) == 0 {
			fmt.Fprintln(os.Stderr, "No ECS clusters found.")
			return nil
		}

		items := make([]tui.SelectorItem, len(clusters))
		for i, cl := range clusters {
			items[i] = tui.SelectorItem{
				Label: cl.Name,
				Value: cl.Name,
				Hint:  fmt.Sprintf("%s · %d running · %d pending", cl.Status, cl.RunningTasks, cl.PendingTasks),
			}
		}

		idx, err := tui.RunSelector(items, "Select ECS Cluster")
		if err != nil {
			return fmt.Errorf("cluster selector: %w", err)
		}
		if idx < 0 {
			return nil // cancelled
		}
		cluster = clusters[idx].Name
	}

	// 4. Service selector
	if service == "" {
		services, err := client.ListServices(ctx, cluster)
		if err != nil {
			return fmt.Errorf("list services: %w", err)
		}
		if len(services) == 0 {
			fmt.Fprintln(os.Stderr, "No services found in cluster.")
			return nil
		}

		items := make([]tui.SelectorItem, len(services))
		for i, svc := range services {
			hint := fmt.Sprintf("%d/%d running", svc.RunningCount, svc.DesiredCount)
			if svc.LastDeployment != "" {
				hint += fmt.Sprintf(" · deployed %s", svc.LastDeployment)
			}
			items[i] = tui.SelectorItem{
				Label: svc.Name,
				Value: svc.Name,
				Hint:  hint,
			}
		}

		idx, err := tui.RunSelector(items, "Select Service")
		if err != nil {
			return fmt.Errorf("service selector: %w", err)
		}
		if idx < 0 {
			return nil // cancelled
		}
		service = services[idx].Name
	}

	// 5. Task definition selector
	taskDefARN := flagTaskDef
	taskDefDisplay := "(current)"

	if taskDefARN == "" {
		// Get current service's task def family
		currentTD, family, err := client.GetServiceTaskDefFamily(ctx, cluster, service)
		if err != nil {
			return fmt.Errorf("get task def family: %w", err)
		}

		// List recent revisions
		defs, err := client.ListTaskDefRevisions(ctx, family, 10)
		if err != nil {
			return fmt.Errorf("list task defs: %w", err)
		}

		if len(defs) > 0 {
			items := make([]tui.SelectorItem, len(defs))
			for i, td := range defs {
				hint := fmt.Sprintf("cpu: %s · mem: %s · %s", td.CPU, td.Memory, td.Status)
				selected := td.ARN == currentTD
				label := fmt.Sprintf("%s:%d", td.Family, td.Revision)
				if selected {
					label += " (current)"
				}
				items[i] = tui.SelectorItem{
					Label:    label,
					Value:    td.ARN,
					Hint:     hint,
					Selected: selected,
				}
			}

			idx, err := tui.RunSelector(items, "Select Task Definition")
			if err != nil {
				return fmt.Errorf("task def selector: %w", err)
			}
			if idx < 0 {
				return nil // cancelled
			}
			taskDefARN = items[idx].Value
			taskDefDisplay = items[idx].Label
		}
	} else {
		taskDefDisplay = taskDefARN
	}

	// 6. Confirmation
	confirmItems := []string{
		fmt.Sprintf("Cluster:  %s", cluster),
		fmt.Sprintf("Service:  %s", service),
		fmt.Sprintf("Task Def: %s", taskDefDisplay),
		fmt.Sprintf("Region:   %s", region),
	}

	ok, err := tui.RunConfirm(
		"Force new deployment?",
		tui.WithDestructive(),
		tui.WithItems(confirmItems),
	)
	if err != nil {
		return fmt.Errorf("confirm: %w", err)
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "Cancelled.")
		return nil
	}

	// 7. Deploy
	fmt.Fprintln(os.Stderr)
	if taskDefARN != "" {
		if err := client.DeployWithTaskDef(ctx, cluster, service, taskDefARN); err != nil {
			return fmt.Errorf("deployment failed: %w", err)
		}
	} else {
		if err := client.ForceNewDeployment(ctx, cluster, service); err != nil {
			return fmt.Errorf("deployment failed: %w", err)
		}
	}
	fmt.Fprintf(os.Stderr, "%s Deployment triggered for %s on %s\n",
		tui.Green.Render("✔"), tui.Bold.Render(service), tui.Bold.Render(cluster))

	// 7. Optional wait
	if flagWait {
		fmt.Fprintf(os.Stderr, "%s Waiting for service to stabilize...\n", tui.Yellow.Render("⏳"))
		if err := client.WaitForStable(ctx, cluster, service, 10*time.Minute); err != nil {
			return fmt.Errorf("wait failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "%s Service stabilized\n", tui.Green.Render("✔"))
	}

	return nil
}
