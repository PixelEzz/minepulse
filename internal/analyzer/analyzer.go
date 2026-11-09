package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PixelEzz/minepulse/internal/terminal"
)

// Finding represents a single security, performance, or resource health item.
type Finding struct {
	ConfigFile     string         `json:"config_file"`
	Key            string         `json:"key"`
	CurrentValue   string         `json:"current_value"`
	Recommended    string         `json:"recommended"`
	Severity       terminal.Level `json:"severity"`
	Category       string         `json:"category"`
	Impact         string         `json:"impact"`
	Recommendation string         `json:"recommendation"`
}

// AuditReport aggregates all findings and computes an overall health score (0-100).
type AuditReport struct {
	Directory   string    `json:"directory"`
	HealthScore int       `json:"health_score"`
	FilesScanned []string `json:"files_scanned"`
	Findings    []Finding `json:"findings"`
	Summary     string    `json:"summary"`
}

// Auditor handles scanning and baseline validation.
type Auditor struct {
	BaseDir string
}

// New creates a new Auditor instance.
func New(baseDir string) *Auditor {
	return &Auditor{BaseDir: baseDir}
}

// Run performs the complete audit across supported configurations.
func (a *Auditor) Run() (*AuditReport, error) {
	info, err := os.Stat(a.BaseDir)
	if err != nil {
		return nil, fmt.Errorf("target directory inaccessible: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("target path %q is not a directory", a.BaseDir)
	}

	report := &AuditReport{
		Directory: a.BaseDir,
		Findings:  make([]Finding, 0),
	}

	// 1. Audit server.properties
	spPath := filepath.Join(a.BaseDir, "server.properties")
	if fileExists(spPath) {
		report.FilesScanned = append(report.FilesScanned, "server.properties")
		findings, err := auditServerProperties(spPath)
		if err == nil {
			report.Findings = append(report.Findings, findings...)
		}
	}

	// 2. Audit paper-world-defaults.yml or paper-global.yml
	paperPaths := []string{
		filepath.Join(a.BaseDir, "config", "paper-world-defaults.yml"),
		filepath.Join(a.BaseDir, "config", "paper-global.yml"),
		filepath.Join(a.BaseDir, "paper.yml"),
	}
	for _, p := range paperPaths {
		if fileExists(p) {
			rel, _ := filepath.Rel(a.BaseDir, p)
			report.FilesScanned = append(report.FilesScanned, rel)
			findings, err := auditPaperConfig(p, rel)
			if err == nil {
				report.Findings = append(report.Findings, findings...)
			}
		}
	}

	// 3. Audit velocity.toml
	velPath := filepath.Join(a.BaseDir, "velocity.toml")
	if fileExists(velPath) {
		report.FilesScanned = append(report.FilesScanned, "velocity.toml")
		findings, err := auditVelocityConfig(velPath)
		if err == nil {
			report.Findings = append(report.Findings, findings...)
		}
	}

	// Compute health score (baseline 100, CRIT = -20, WARN = -10, INFO = -2)
	score := 100
	for _, f := range report.Findings {
		switch f.Severity {
		case terminal.LevelCrit:
			score -= 20
		case terminal.LevelWarn:
			score -= 10
		case terminal.LevelInfo:
			score -= 2
		}
	}
	if score < 0 {
		score = 0
	}
	report.HealthScore = score

	if score >= 90 {
		report.Summary = "Excellent. Configurations adhere to modern production baselines."
	} else if score >= 70 {
		report.Summary = "Good with minor warnings. Optimization recommended before peak load."
	} else if score >= 50 {
		report.Summary = "Suboptimal. Contains potential performance bottlenecks or memory hazards."
	} else {
		report.Summary = "Critical attention required. Severe stability or security misconfigurations detected."
	}

	return report, nil
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// auditServerProperties inspects Vanilla/Paper standard properties.
func auditServerProperties(path string) ([]Finding, error) {
	props, err := parseProperties(path)
	if err != nil {
		return nil, err
	}

	var findings []Finding

	// Check view-distance
	if vdStr, ok := props["view-distance"]; ok {
		vd, err := strconv.Atoi(vdStr)
		if err == nil {
			if vd > 12 {
				findings = append(findings, Finding{
					ConfigFile:     "server.properties",
					Key:            "view-distance",
					CurrentValue:   vdStr,
					Recommended:    "6 - 10",
					Severity:       terminal.LevelWarn,
					Category:       "Performance & Memory",
					Impact:         "Severe chunk packet serialization load, memory pressure, and tick spikes.",
					Recommendation: "Reduce view-distance to 8 or 10, pair with No-Tick-View-Distance if on Paper.",
				})
			}
		}
	}

	// Check simulation-distance vs view-distance
	if sdStr, ok := props["simulation-distance"]; ok {
		sd, err := strconv.Atoi(sdStr)
		vdStr := props["view-distance"]
		vd, errVd := strconv.Atoi(vdStr)
		if err == nil && errVd == nil {
			if sd >= vd && sd > 6 {
				findings = append(findings, Finding{
					ConfigFile:     "server.properties",
					Key:            "simulation-distance",
					CurrentValue:   sdStr,
					Recommended:    "4 - 6",
					Severity:       terminal.LevelWarn,
					Category:       "CPU Bottleneck",
					Impact:         "Excessive entity ticking and redstone updates across wide chunk radii.",
					Recommendation: "Decouple simulation-distance (set to 4-6) while keeping view-distance higher.",
				})
			}
		}
	}

	// Check network-compression-threshold
	if nctStr, ok := props["network-compression-threshold"]; ok {
		nct, err := strconv.Atoi(nctStr)
		if err == nil {
			if nct < 64 && nct > 0 {
				findings = append(findings, Finding{
					ConfigFile:     "server.properties",
					Key:            "network-compression-threshold",
					CurrentValue:   nctStr,
					Recommended:    "256 - 512 (or -1 behind Velocity)",
					Severity:       terminal.LevelCrit,
					Category:       "CPU Overhead",
					Impact:         "Compressing tiny packets generates extreme Netty CPU overhead without bandwidth benefit.",
					Recommendation: "Set network-compression-threshold to 256, or disable (-1) if behind a proxy like Velocity.",
				})
			} else if nct == 0 {
				findings = append(findings, Finding{
					ConfigFile:     "server.properties",
					Key:            "network-compression-threshold",
					CurrentValue:   nctStr,
					Recommended:    "256",
					Severity:       terminal.LevelCrit,
					Category:       "Bandwidth Hazard",
					Impact:         "Compressing every packet (including keep-alive) destroys Netty event loop throughput.",
					Recommendation: "Set network-compression-threshold to 256.",
				})
			}
		}
	}

	// Check sync-chunk-writes
	if scw, ok := props["sync-chunk-writes"]; ok {
		if strings.EqualFold(scw, "true") {
			findings = append(findings, Finding{
				ConfigFile:     "server.properties",
				Key:            "sync-chunk-writes",
				CurrentValue:   scw,
				Recommended:    "false",
				Severity:       terminal.LevelCrit,
				Category:       "I/O Blocking",
				Impact:         "Synchronous chunk writes cause full main-thread stalls during auto-save ticks.",
				Recommendation: "Set sync-chunk-writes=false to allow asynchronous I/O flushes.",
			})
		}
	}

	// Check online-mode
	if om, ok := props["online-mode"]; ok {
		if strings.EqualFold(om, "false") {
			findings = append(findings, Finding{
				ConfigFile:     "server.properties",
				Key:            "online-mode",
				CurrentValue:   om,
				Recommended:    "true (or modern forwarding behind proxy)",
				Severity:       terminal.LevelWarn,
				Category:       "Security Risk",
				Impact:         "Offline mode allows arbitrary UUID spoofing and unauthorized operator logins.",
				Recommendation: "Enable online-mode=true or enforce Velocity modern forwarding with firewall isolation.",
			})
		}
	}

	return findings, nil
}

// auditPaperConfig inspects Paper configuration keys.
func auditPaperConfig(path, displayPath string) ([]Finding, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var findings []Finding
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || !strings.Contains(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if key == "max-auto-save-chunks-per-tick" {
			n, _ := strconv.Atoi(val)
			if n > 24 {
				findings = append(findings, Finding{
					ConfigFile:     displayPath,
					Key:            key,
					CurrentValue:   val,
					Recommended:    "6 - 12",
					Severity:       terminal.LevelWarn,
					Category:       "Tick Spikes",
					Impact:         "Too many chunk writes queued per tick triggers noticeable frame stutter during saves.",
					Recommendation: "Tune max-auto-save-chunks-per-tick down to 8 or 10.",
				})
			}
		}

		if key == "optimize-explosions" && strings.EqualFold(val, "false") {
			findings = append(findings, Finding{
				ConfigFile:     displayPath,
				Key:            key,
				CurrentValue:   val,
				Recommended:    "true",
				Severity:       terminal.LevelInfo,
				Category:       "Algorithm Optimization",
				Impact:         "Default explosion algorithm is O(n^3) and causes severe lag under mass TNT detonation.",
				Recommendation: "Enable optimize-explosions: true for optimized vector raytracing.",
			})
		}

		if key == "prevent-moving-into-unloaded-chunks" && strings.EqualFold(val, "false") {
			findings = append(findings, Finding{
				ConfigFile:     displayPath,
				Key:            key,
				CurrentValue:   val,
				Recommended:    "true",
				Severity:       terminal.LevelCrit,
				Category:       "Main-Thread Stall",
				Impact:         "Sync-loading chunks when entities move into unloaded regions freezes server ticks.",
				Recommendation: "Set prevent-moving-into-unloaded-chunks: true.",
			})
		}
	}

	return findings, nil
}

// auditVelocityConfig inspects Velocity proxy configuration.
func auditVelocityConfig(path string) ([]Finding, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var findings []Finding
	scanner := bufio.NewScanner(file)
	hasForwardingSecret := false
	playerInfoForwarding := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "player-info-forwarding-mode") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				playerInfoForwarding = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			}
		}

		if strings.HasPrefix(line, "forwarding-secret-file") || strings.HasPrefix(line, "forwarding-secret") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 && strings.Trim(strings.TrimSpace(parts[1]), `"'`) != "" {
				hasForwardingSecret = true
			}
		}
	}

	if playerInfoForwarding == "none" || playerInfoForwarding == "legacy" {
		findings = append(findings, Finding{
			ConfigFile:     "velocity.toml",
			Key:            "player-info-forwarding-mode",
			CurrentValue:   playerInfoForwarding,
			Recommended:    `"modern"`,
			Severity:       terminal.LevelCrit,
			Category:       "Security Risk",
			Impact:         "BungeeCord legacy forwarding or none is vulnerable to spoofing if backend ports are accessible.",
			Recommendation: "Use modern player-info forwarding with a shared secret and backend firewall rules.",
		})
	}

	if playerInfoForwarding == "modern" && !hasForwardingSecret {
		findings = append(findings, Finding{
			ConfigFile:     "velocity.toml",
			Key:            "forwarding-secret-file",
			CurrentValue:   "empty",
			Recommended:    `"forwarding.secret"`,
			Severity:       terminal.LevelCrit,
			Category:       "Security Failure",
			Impact:         "Modern forwarding configured without a secret will fail backend handshakes or leave trust open.",
			Recommendation: "Set forwarding-secret-file = \"forwarding.secret\" and deploy the token to Paper backends.",
		})
	}

	return findings, nil
}

// parseProperties parses a Java properties file into a key-value map.
func parseProperties(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	props := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") || len(line) == 0 {
			continue
		}
		eqIdx := strings.Index(line, "=")
		if eqIdx != -1 {
			k := strings.TrimSpace(line[:eqIdx])
			v := strings.TrimSpace(line[eqIdx+1:])
			props[k] = v
		}
	}
	return props, nil
}
