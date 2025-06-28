/*
Copyright © 2025 changheonshin
*/
package cmd

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/devlikebear/fman/internal/daemon"
	"github.com/spf13/cobra"
)

// queueCmd represents the queue command
var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Manage background scan queue",
	Long: `Manage the background scan queue. You can list jobs, check status, 
cancel jobs, and clear completed jobs.

Available subcommands:
  list    - List all jobs in the queue
  status  - Check status of a specific job
  cancel  - Cancel a pending or running job
  clear   - Clear completed and failed jobs`,
}

// queueListCmd represents the queue list command
var queueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all jobs in the queue",
	Long: `List all jobs in the queue with their status, creation time, and other details.
Shows jobs in all states: pending, running, completed, failed, and cancelled.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQueueList(cmd, args)
	},
}

// queueStatusCmd represents the queue status command
var queueStatusCmd = &cobra.Command{
	Use:   "status <job-id>",
	Short: "Check status of a specific job",
	Long: `Check the detailed status of a specific job by its ID.
Shows job progress, statistics, and any error messages.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQueueStatus(cmd, args)
	},
}

// queueCancelCmd represents the queue cancel command
var queueCancelCmd = &cobra.Command{
	Use:   "cancel <job-id>",
	Short: "Cancel a pending or running job",
	Long: `Cancel a job that is currently pending or running.
Completed jobs cannot be cancelled.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQueueCancel(cmd, args)
	},
}

// queueClearCmd represents the queue clear command
var queueClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear completed and failed jobs",
	Long: `Clear all completed and failed jobs from the queue.
This helps keep the queue clean and reduces memory usage.
Pending and running jobs are not affected.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQueueClear(cmd, args)
	},
}

func runQueueList(cmd *cobra.Command, args []string) error {
	// Create daemon client
	client := daemon.NewDaemonClient(nil)

	// Connect to daemon
	err := client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Disconnect()

	// Get all jobs
	jobs, err := client.ListJobs("")
	if err != nil {
		return fmt.Errorf("failed to list jobs: %w", err)
	}

	if len(jobs) == 0 {
		fmt.Println("No jobs in queue.")
		return nil
	}

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Print header
	fmt.Fprintln(w, "Job ID\tStatus\tPath\tCreated\tDuration")
	fmt.Fprintln(w, "------\t------\t----\t-------\t--------")

	// Print jobs
	for _, job := range jobs {
		duration := formatJobDuration(job)
		status := formatJobStatus(job.Status)

		// Truncate path if too long
		path := job.Path
		if len(path) > 50 {
			path = "..." + path[len(path)-47:]
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			job.ID[:8]+"...", // Show first 8 characters of ID
			status,
			path,
			job.CreatedAt.Format("15:04:05"),
			duration,
		)
	}

	return nil
}

func runQueueStatus(cmd *cobra.Command, args []string) error {
	jobID := args[0]

	// Create daemon client
	client := daemon.NewDaemonClient(nil)

	// Connect to daemon
	err := client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Disconnect()

	// Get job details
	job, err := client.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("failed to get job status: %w", err)
	}

	// Print job details
	fmt.Printf("🆔 Job ID: %s\n", job.ID)
	fmt.Printf("📁 Path: %s\n", job.Path)
	fmt.Printf("📊 Status: %s\n", formatJobStatus(job.Status))
	fmt.Printf("⏰ Created: %s\n", job.CreatedAt.Format("2006-01-02 15:04:05"))

	if job.StartedAt != nil {
		fmt.Printf("🚀 Started: %s\n", job.StartedAt.Format("2006-01-02 15:04:05"))
	}

	if job.CompletedAt != nil {
		fmt.Printf("✅ Completed: %s\n", job.CompletedAt.Format("2006-01-02 15:04:05"))
	}

	duration := formatJobDuration(job)
	if duration != "-" {
		fmt.Printf("⏱️  Duration: %s\n", duration)
	}

	// Show progress if available
	if job.Progress != nil {
		fmt.Printf("📈 Progress: %d files processed\n", job.Progress.FilesProcessed)
		if job.Progress.CurrentPath != "" {
			fmt.Printf("📂 Current: %s\n", job.Progress.CurrentPath)
		}
	}

	// Show statistics if available
	if job.Stats != nil {
		fmt.Printf("\n📊 Statistics:\n")
		fmt.Printf("  ✅ Files indexed: %d\n", job.Stats.FilesIndexed)
		fmt.Printf("  ⏭️  Directories skipped: %d\n", job.Stats.DirectoriesSkipped)
		fmt.Printf("  ⚠️  Permission errors: %d\n", job.Stats.PermissionErrors)
	}

	// Show error if any
	if job.Error != "" {
		fmt.Printf("\n❌ Error: %s\n", job.Error)
	}

	// Show options
	if job.Options != nil {
		fmt.Printf("\n⚙️  Options:\n")
		fmt.Printf("  Verbose: %t\n", job.Options.Verbose)
		fmt.Printf("  Force Sudo: %t\n", job.Options.ForceSudo)
	}

	return nil
}

func runQueueCancel(cmd *cobra.Command, args []string) error {
	jobID := args[0]

	// Create daemon client
	client := daemon.NewDaemonClient(nil)

	// Connect to daemon
	err := client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Disconnect()

	// Cancel job
	err = client.CancelJob(jobID)
	if err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	fmt.Printf("✅ Job %s has been cancelled\n", jobID)
	return nil
}

func runQueueClear(cmd *cobra.Command, args []string) error {
	// Create daemon client
	client := daemon.NewDaemonClient(nil)

	// Connect to daemon
	err := client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Disconnect()

	// Clear queue
	err = client.ClearQueue()
	if err != nil {
		return fmt.Errorf("failed to clear queue: %w", err)
	}

	fmt.Println("✅ Queue cleared successfully")
	return nil
}

// Helper functions
func formatJobStatus(status daemon.JobStatus) string {
	switch status {
	case daemon.JobStatusPending:
		return "⏳ Pending"
	case daemon.JobStatusRunning:
		return "🔄 Running"
	case daemon.JobStatusCompleted:
		return "✅ Completed"
	case daemon.JobStatusFailed:
		return "❌ Failed"
	case daemon.JobStatusCancelled:
		return "⛔ Cancelled"
	default:
		return string(status)
	}
}

func formatJobDuration(job *daemon.Job) string {
	if job.StartedAt == nil {
		return "-"
	}

	endTime := time.Now()
	if job.CompletedAt != nil {
		endTime = *job.CompletedAt
	}

	duration := endTime.Sub(*job.StartedAt)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm%ds", int(duration.Minutes()), int(duration.Seconds())%60)
	} else {
		return fmt.Sprintf("%dh%dm", int(duration.Hours()), int(duration.Minutes())%60)
	}
}

func init() {
	rootCmd.AddCommand(queueCmd)

	// Add subcommands
	queueCmd.AddCommand(queueListCmd)
	queueCmd.AddCommand(queueStatusCmd)
	queueCmd.AddCommand(queueCancelCmd)
	queueCmd.AddCommand(queueClearCmd)
}
