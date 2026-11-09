package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	// Version can be populated at build time via -ldflags "-X github.com/PixelEzz/minepulse/cmd.Version=vX.Y.Z"
	Version = "v0.1.0"

	// Global Flags
	flagTimeout time.Duration
	flagJSON    bool
	flagVerbose bool
)

// RootCmd represents the base command when called without any subcommands.
var RootCmd = &cobra.Command{
	Use:   "minepulse",
	Short: "High-performance Minecraft infrastructure diagnostics and SLP engine",
	Long: `minepulse is a production-grade, zero-dependency diagnostic utility designed for
Minecraft server operators, infrastructure engineers, and hosting providers.

Capabilities:
  • Protocol-level Server List Ping (SLP) breakdown (TCP dial, Handshake, Ping/Pong)
  • Non-destructive configuration audit for Paper, Purpur, Velocity, and Vanilla
  • High-resolution network latency benchmarking, percentiles (p50/p95/p99), and jitter analysis
  • Automation-ready JSON output with clean fallback for CI/CD pipelines`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().DurationVarP(&flagTimeout, "timeout", "t", 5*time.Second, "Timeout for network calls and socket deadlines")
	RootCmd.PersistentFlags().BoolVarP(&flagJSON, "json", "j", false, "Output machine-readable JSON format")
	RootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose diagnostic logs")
}
