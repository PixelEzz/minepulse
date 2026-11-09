package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/PixelEzz/minepulse/internal/slp"
	"github.com/PixelEzz/minepulse/internal/terminal"
	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping <host[:port]>",
	Short: "Perform protocol-level Server List Ping (SLP) inspection",
	Long: `Dispatches standard Minecraft Handshake (0x00) and Status Request (0x00) packets
over a direct TCP stream. Accurately profiles TCP connect latency, Handshake round-trip time,
and Ping/Pong RTT while decoding modern chat component MOTD, player counts, and protocol flags.`,
	Args: cobra.ExactArgs(1),
	RunE: runPing,
}

func runPing(cmd *cobra.Command, args []string) error {
	target := args[0]
	ui := terminal.New(os.Stdout, os.Stderr, flagJSON, flagVerbose)

	ctx, cancel := context.WithTimeout(context.Background(), flagTimeout)
	defer cancel()

	client := slp.NewClient()
	result, err := client.Ping(ctx, target)
	if err != nil {
		if flagJSON {
			_ = ui.OutputJSON(map[string]interface{}{
				"target": target,
				"error":  err.Error(),
			})
			return nil
		}
		return fmt.Errorf("ping failed for %s: %w", target, err)
	}

	if flagJSON {
		return ui.OutputJSON(result)
	}

	ui.Header(fmt.Sprintf("SLP Diagnostics: %s:%d (%s)", result.Host, result.Port, result.ResolvedIP))

	ui.KeyValue("Server Version", fmt.Sprintf("%s (Protocol %d)", ui.Bold(result.ServerVersion), result.ProtocolVersion))
	ui.KeyValue("Players", fmt.Sprintf("%s / %d", ui.Green(fmt.Sprintf("%d", result.PlayersOnline)), result.PlayersMax))
	ui.KeyValue("MOTD", result.MOTD)
	ui.KeyValue("Secure Chat", result.SecureChatStatus)

	ui.Header("Network Latency Breakdown")
	ui.KeyValue("TCP Dial Latency", fmt.Sprintf("%v", result.TCPConnectTime.Round(100*1000))) // round to 0.1ms
	ui.KeyValue("Handshake RTT", fmt.Sprintf("%v", result.HandshakeRTT.Round(100*1000)))
	ui.KeyValue("Ping/Pong RTT", fmt.Sprintf("%v", result.PingPongRTT.Round(100*1000)))
	ui.KeyValue("Total Round-Trip", ui.Cyan(fmt.Sprintf("%v", result.TotalRTT.Round(100*1000))))

	if len(result.Status.Players.Sample) > 0 {
		ui.Header("Player Sample")
		for _, p := range result.Status.Players.Sample {
			fmt.Fprintf(os.Stdout, "  • %-20s %s\n", p.Name, ui.Dim(p.ID))
		}
	}

	ui.PrintFooter(Version)
	return nil
}

func init() {
	RootCmd.AddCommand(pingCmd)
}
