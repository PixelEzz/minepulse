package analyzer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PixelEzz/minepulse/internal/analyzer"
	"github.com/PixelEzz/minepulse/internal/terminal"
)

func TestAuditorServerProperties(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "minepulse-audit-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	spContent := `
view-distance=16
simulation-distance=16
network-compression-threshold=32
sync-chunk-writes=true
online-mode=false
`
	if err := os.WriteFile(filepath.Join(tmpDir, "server.properties"), []byte(spContent), 0644); err != nil {
		t.Fatalf("failed to write server.properties: %v", err)
	}

	auditor := analyzer.New(tmpDir)
	report, err := auditor.Run()
	if err != nil {
		t.Fatalf("auditor Run() failed: %v", err)
	}

	if len(report.Findings) < 4 {
		t.Errorf("expected at least 4 findings, got %d", len(report.Findings))
	}

	// Verify critical findings exist
	hasSyncChunkWriteCrit := false
	hasCompressionCrit := false
	for _, f := range report.Findings {
		if f.Key == "sync-chunk-writes" && f.Severity == terminal.LevelCrit {
			hasSyncChunkWriteCrit = true
		}
		if f.Key == "network-compression-threshold" && f.Severity == terminal.LevelCrit {
			hasCompressionCrit = true
		}
	}

	if !hasSyncChunkWriteCrit {
		t.Errorf("expected sync-chunk-writes critical finding")
	}
	if !hasCompressionCrit {
		t.Errorf("expected network-compression-threshold critical finding")
	}

	if report.HealthScore >= 80 {
		t.Errorf("health score should be low for bad config, got %d", report.HealthScore)
	}
}

func TestAuditorVelocityConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "minepulse-vel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	velContent := `
player-info-forwarding-mode = "legacy"
forwarding-secret-file = ""
`
	if err := os.WriteFile(filepath.Join(tmpDir, "velocity.toml"), []byte(velContent), 0644); err != nil {
		t.Fatalf("failed to write velocity.toml: %v", err)
	}

	auditor := analyzer.New(tmpDir)
	report, err := auditor.Run()
	if err != nil {
		t.Fatalf("auditor Run() failed: %v", err)
	}

	foundLegacyCrit := false
	for _, f := range report.Findings {
		if f.Key == "player-info-forwarding-mode" && f.Severity == terminal.LevelCrit {
			foundLegacyCrit = true
		}
	}
	if !foundLegacyCrit {
		t.Errorf("expected player-info-forwarding-mode critical alert")
	}
}
