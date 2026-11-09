package cmd

import (
	"fmt"
	"os"

	"github.com/PixelEzz/minepulse/internal/analyzer"
	"github.com/PixelEzz/minepulse/internal/terminal"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit [server-path]",
	Short: "Audit Minecraft and Proxy configuration files for safety and performance",
	Long: `Inspects server.properties, paper configurations, and velocity.toml against known
production baselines. Flags dangerous memory leaks, asynchronous chunk write hazards,
insecure network forwarding setups, and unoptimized compression thresholds.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAudit,
}

func runAudit(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	ui := terminal.New(os.Stdout, os.Stderr, flagJSON, flagVerbose)
	aud := analyzer.New(dir)

	report, err := aud.Run()
	if err != nil {
		if flagJSON {
			_ = ui.OutputJSON(map[string]interface{}{
				"directory": dir,
				"error":     err.Error(),
			})
			return nil
		}
		return fmt.Errorf("audit failed: %w", err)
	}

	if flagJSON {
		return ui.OutputJSON(report)
	}

	ui.Header(fmt.Sprintf("Configuration Audit: %s", report.Directory))
	ui.KeyValue("Files Inspected", fmt.Sprintf("%d (%v)", len(report.FilesScanned), report.FilesScanned))

	scoreStr := fmt.Sprintf("%d/100", report.HealthScore)
	if report.HealthScore >= 90 {
		scoreStr = ui.Green(scoreStr)
	} else if report.HealthScore >= 70 {
		scoreStr = ui.Yellow(scoreStr)
	} else {
		scoreStr = ui.Red(scoreStr)
	}
	ui.KeyValue("Health Score", scoreStr)
	ui.KeyValue("Status", report.Summary)

	if len(report.Findings) == 0 {
		fmt.Fprintf(os.Stdout, "\n  %s No performance or security anti-patterns detected.\n", ui.Chip(terminal.LevelOK))
	} else {
		ui.Header(fmt.Sprintf("Findings & Action Items (%d detected)", len(report.Findings)))
		for _, f := range report.Findings {
			chip := ui.Chip(f.Severity)
			fmt.Fprintf(os.Stdout, "\n  %s %s [%s]\n", chip, ui.Bold(f.Key), ui.Dim(f.ConfigFile))
			fmt.Fprintf(os.Stdout, "      Current:     %s\n", f.CurrentValue)
			fmt.Fprintf(os.Stdout, "      Target:      %s\n", ui.Green(f.Recommended))
			fmt.Fprintf(os.Stdout, "      Impact:      %s\n", f.Impact)
			fmt.Fprintf(os.Stdout, "      Remediation: %s\n", f.Recommendation)
		}
	}

	ui.PrintFooter(Version)
	return nil
}

func init() {
	RootCmd.AddCommand(auditCmd)
}
