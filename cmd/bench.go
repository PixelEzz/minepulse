package cmd

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/PixelEzz/minepulse/internal/terminal"
	"github.com/spf13/cobra"
)

var (
	flagBenchCount    int
	flagBenchInterval time.Duration
)

// BenchMetrics holds computed summary metrics from latency trials.
type BenchMetrics struct {
	Target      string        `json:"target"`
	PacketsSent int           `json:"packets_sent"`
	PacketsRecv int           `json:"packets_recv"`
	PacketLoss  float64       `json:"packet_loss_percent"`
	MinLatency  time.Duration `json:"min_latency_ms"`
	MaxLatency  time.Duration `json:"max_latency_ms"`
	MeanLatency time.Duration `json:"mean_latency_ms"`
	P50         time.Duration `json:"p50_latency_ms"`
	P95         time.Duration `json:"p95_latency_ms"`
	P99         time.Duration `json:"p99_latency_ms"`
	Jitter      time.Duration `json:"jitter_ms"`
}

var benchCmd = &cobra.Command{
	Use:   "bench <host[:port]>",
	Short: "Benchmark network latency, TCP jitter, and packet loss",
	Long: `Dispatches repeated TCP handshake probes to measure connection latency percentiles
(p50, p95, p99), standard jitter variation, and drop rate. Useful for diagnosing routing instability,
datacenter peering issues, and DDoS mitigation scrubbing overhead.`,
	Args: cobra.ExactArgs(1),
	RunE: runBench,
}

func runBench(cmd *cobra.Command, args []string) error {
	target := args[0]
	ui := terminal.New(os.Stdout, os.Stderr, flagJSON, flagVerbose)

	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		host = target
		portStr = "25565"
	}

	portInt, err := strconv.Atoi(portStr)
	if err != nil || portInt <= 0 || portInt > 65535 {
		return fmt.Errorf("invalid port %q: must be between 1 and 65535", portStr)
	}

	address := net.JoinHostPort(host, strconv.Itoa(portInt))

	if !flagJSON {
		fmt.Fprintf(os.Stdout, "Benchmarking TCP handshake against %s (%d trials)...\n", ui.Bold(address), flagBenchCount)
	}

	var latencies []time.Duration
	dialer := net.Dialer{}

	for i := 1; i <= flagBenchCount; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), flagTimeout)
		t0 := time.Now()
		conn, err := dialer.DialContext(ctx, "tcp", address)
		elapsed := time.Since(t0)
		cancel()

		if err == nil {
			_ = conn.Close()
			latencies = append(latencies, elapsed)
			if !flagJSON && flagVerbose {
				fmt.Fprintf(os.Stdout, "  seq=%d target=%s rtt=%v\n", i, address, elapsed.Round(100*time.Microsecond))
			}
		} else {
			if !flagJSON && flagVerbose {
				fmt.Fprintf(os.Stdout, "  seq=%d target=%s error=%v\n", i, address, err)
			}
		}

		if i < flagBenchCount {
			time.Sleep(flagBenchInterval)
		}
	}

	sent := flagBenchCount
	recv := len(latencies)
	loss := float64(sent-recv) / float64(sent) * 100.0

	metrics := BenchMetrics{
		Target:      address,
		PacketsSent: sent,
		PacketsRecv: recv,
		PacketLoss:  loss,
	}

	if recv > 0 {
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		metrics.MinLatency = latencies[0]
		metrics.MaxLatency = latencies[recv-1]

		var total time.Duration
		for _, l := range latencies {
			total += l
		}
		metrics.MeanLatency = total / time.Duration(recv)

		metrics.P50 = latencies[percentileIndex(recv, 50)]
		metrics.P95 = latencies[percentileIndex(recv, 95)]
		metrics.P99 = latencies[percentileIndex(recv, 99)]

		// Calculate RFC 3550 style interarrival jitter or mean absolute deviation
		if recv > 1 {
			var sumDiff float64
			for i := 1; i < recv; i++ {
				diff := math.Abs(float64(latencies[i] - latencies[i-1]))
				sumDiff += diff
			}
			metrics.Jitter = time.Duration(sumDiff / float64(recv-1))
		}
	}

	if flagJSON {
		return ui.OutputJSON(metrics)
	}

	ui.Header(fmt.Sprintf("Benchmark Results: %s", metrics.Target))
	ui.KeyValue("Probes Transmitted", fmt.Sprintf("%d", metrics.PacketsSent))
	ui.KeyValue("Probes Received", fmt.Sprintf("%d", metrics.PacketsRecv))

	lossStr := fmt.Sprintf("%.1f%%", metrics.PacketLoss)
	if metrics.PacketLoss == 0 {
		lossStr = ui.Green(lossStr)
	} else if metrics.PacketLoss < 5 {
		lossStr = ui.Yellow(lossStr)
	} else {
		lossStr = ui.Red(lossStr)
	}
	ui.KeyValue("Packet Loss", lossStr)

	if recv > 0 {
		ui.Header("Latency Percentiles & Jitter")
		ui.KeyValue("Minimum Latency", fmt.Sprintf("%v", metrics.MinLatency.Round(100*time.Microsecond)))
		ui.KeyValue("Mean Latency", fmt.Sprintf("%v", metrics.MeanLatency.Round(100*time.Microsecond)))
		ui.KeyValue("p50 (Median)", fmt.Sprintf("%v", metrics.P50.Round(100*time.Microsecond)))
		ui.KeyValue("p95 Percentile", fmt.Sprintf("%v", metrics.P95.Round(100*time.Microsecond)))
		ui.KeyValue("p99 Percentile", fmt.Sprintf("%v", metrics.P99.Round(100*time.Microsecond)))
		ui.KeyValue("Maximum Latency", fmt.Sprintf("%v", metrics.MaxLatency.Round(100*time.Microsecond)))
		ui.KeyValue("Calculated Jitter", ui.Cyan(fmt.Sprintf("%v", metrics.Jitter.Round(100*time.Microsecond))))
	}

	ui.PrintFooter(Version)
	return nil
}

func percentileIndex(length int, percentile int) int {
	idx := (percentile * length) / 100
	if idx >= length {
		idx = length - 1
	}
	return idx
}

func init() {
	benchCmd.Flags().IntVarP(&flagBenchCount, "count", "c", 10, "Number of probe trials to execute")
	benchCmd.Flags().DurationVarP(&flagBenchInterval, "interval", "i", 100*time.Millisecond, "Interval delay between successive probes")
	RootCmd.AddCommand(benchCmd)
}
