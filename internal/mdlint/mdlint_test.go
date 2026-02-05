package mdlint

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAnalyzeFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("with H2 sections", func(t *testing.T) {
		content := `# Title

Some intro text.

## First Section

Content of first section.
More content.

## Second Section

Content here.

## Third Section

Last section.
`
		path := writeTempFile(t, dir, "sections.md", content)

		totalLines, sections, err := AnalyzeFile(path)
		if err != nil {
			t.Fatal(err)
		}

		if totalLines != 16 {
			t.Errorf("expected 16 total lines, got %d", totalLines)
		}

		if len(sections) != 3 {
			t.Fatalf("expected 3 sections, got %d", len(sections))
		}

		if sections[0].Name != "First Section" {
			t.Errorf("expected first section name 'First Section', got %q", sections[0].Name)
		}
		if sections[1].Name != "Second Section" {
			t.Errorf("expected second section name 'Second Section', got %q", sections[1].Name)
		}
		if sections[2].Name != "Third Section" {
			t.Errorf("expected third section name 'Third Section', got %q", sections[2].Name)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		path := writeTempFile(t, dir, "empty.md", "")

		totalLines, sections, err := AnalyzeFile(path)
		if err != nil {
			t.Fatal(err)
		}

		if totalLines != 0 {
			t.Errorf("expected 0 total lines, got %d", totalLines)
		}
		if len(sections) != 0 {
			t.Errorf("expected 0 sections, got %d", len(sections))
		}
	})

	t.Run("no sections", func(t *testing.T) {
		content := `# Title

Just some text with no H2 sections.
Another line.
`
		path := writeTempFile(t, dir, "nosections.md", content)

		totalLines, sections, err := AnalyzeFile(path)
		if err != nil {
			t.Fatal(err)
		}

		if totalLines != 4 {
			t.Errorf("expected 4 total lines, got %d", totalLines)
		}
		if len(sections) != 0 {
			t.Errorf("expected 0 sections, got %d", len(sections))
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, _, err := AnalyzeFile(filepath.Join(dir, "nonexistent.md"))
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})
}

func TestCountFileLines(t *testing.T) {
	dir := t.TempDir()

	t.Run("basic count", func(t *testing.T) {
		path := writeTempFile(t, dir, "lines.txt", "line1\nline2\nline3\n")
		got := CountFileLines(path)
		if got != 3 {
			t.Errorf("expected 3 lines, got %d", got)
		}
	})

	t.Run("nonexistent file returns 0", func(t *testing.T) {
		got := CountFileLines(filepath.Join(dir, "nope.txt"))
		if got != 0 {
			t.Errorf("expected 0 for nonexistent file, got %d", got)
		}
	})
}

func TestListRulesFiles(t *testing.T) {
	t.Run("with rules directory", func(t *testing.T) {
		dir := t.TempDir()
		rulesDir := filepath.Join(dir, "rules")
		os.MkdirAll(rulesDir, 0755)

		writeTempFile(t, rulesDir, "testing.md", "# Testing\n")
		writeTempFile(t, rulesDir, "development.md", "# Dev\nLine 2\n")
		writeTempFile(t, rulesDir, "readme.txt", "not markdown")

		files := ListRulesFiles(rulesDir)
		if len(files) != 2 {
			t.Fatalf("expected 2 rules files, got %d", len(files))
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		files := ListRulesFiles("/nonexistent/path")
		if files != nil {
			t.Errorf("expected nil for nonexistent dir, got %v", files)
		}
	})

	t.Run("empty rules dir string", func(t *testing.T) {
		files := ListRulesFiles("")
		if files != nil {
			t.Errorf("expected nil for empty dir string, got %v", files)
		}
	})
}

func TestLint(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.FileName = "TEST.md"
	cfg.RulesDir = ".test/rules"
	cfg.StatsCmd = "tk test stats"

	t.Run("file not found returns ok", func(t *testing.T) {
		result, err := Lint(filepath.Join(dir, "missing.md"), cfg)
		if err != nil {
			t.Fatal(err)
		}
		if result.Status != "ok" {
			t.Errorf("expected status 'ok', got %q", result.Status)
		}
	})

	t.Run("below thresholds returns ok", func(t *testing.T) {
		// Create a small file (10 lines)
		var lines string
		for i := 0; i < 10; i++ {
			lines += "line\n"
		}
		path := writeTempFile(t, dir, "small.md", lines)

		result, err := Lint(path, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if result.Status != "ok" {
			t.Errorf("expected status 'ok', got %q", result.Status)
		}
		if result.TotalLines != 10 {
			t.Errorf("expected 10 lines, got %d", result.TotalLines)
		}
	})

	t.Run("above warn threshold returns warning", func(t *testing.T) {
		var lines string
		for i := 0; i < 160; i++ {
			lines += "line\n"
		}
		path := writeTempFile(t, dir, "warn.md", lines)

		result, err := Lint(path, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if result.Status != "warning" {
			t.Errorf("expected status 'warning', got %q", result.Status)
		}
	})

	t.Run("above max threshold returns error", func(t *testing.T) {
		var lines string
		for i := 0; i < 210; i++ {
			lines += "line\n"
		}
		path := writeTempFile(t, dir, "big.md", lines)

		result, err := Lint(path, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if result.Status != "error" {
			t.Errorf("expected status 'error', got %q", result.Status)
		}
	})

	t.Run("large sections detected", func(t *testing.T) {
		content := "# Title\n\n## Big Section\n\n"
		for i := 0; i < 60; i++ {
			content += "line\n"
		}
		content += "\n## Small Section\n\nline\n"
		path := writeTempFile(t, dir, "large-sections.md", content)

		result, err := Lint(path, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.LargeSections) != 1 {
			t.Fatalf("expected 1 large section, got %d", len(result.LargeSections))
		}
		if result.LargeSections[0].Name != "Big Section" {
			t.Errorf("expected large section 'Big Section', got %q", result.LargeSections[0].Name)
		}
	})

	t.Run("recommendations include FileName and RulesDir", func(t *testing.T) {
		var lines string
		for i := 0; i < 160; i++ {
			lines += "line\n"
		}
		path := writeTempFile(t, dir, "recs.md", lines)

		result, err := Lint(path, cfg)
		if err != nil {
			t.Fatal(err)
		}

		hasRulesDir := false
		hasFileName := false
		hasStatsCmd := false
		for _, r := range result.Recommendations {
			if contains(r, cfg.RulesDir) {
				hasRulesDir = true
			}
			if contains(r, cfg.FileName) {
				hasFileName = true
			}
			if contains(r, cfg.StatsCmd) {
				hasStatsCmd = true
			}
		}

		if !hasRulesDir {
			t.Error("expected recommendation mentioning RulesDir")
		}
		if !hasFileName {
			t.Error("expected recommendation mentioning FileName")
		}
		if !hasStatsCmd {
			t.Error("expected recommendation mentioning StatsCmd")
		}
	})

	t.Run("no RulesDir omits rules recommendation", func(t *testing.T) {
		noRulesCfg := cfg
		noRulesCfg.RulesDir = ""

		var lines string
		for i := 0; i < 160; i++ {
			lines += "line\n"
		}
		path := writeTempFile(t, dir, "norules.md", lines)

		result, err := Lint(path, noRulesCfg)
		if err != nil {
			t.Fatal(err)
		}

		for _, r := range result.Recommendations {
			if contains(r, "modules") && contains(r, "/") {
				t.Errorf("did not expect rules dir recommendation, got: %s", r)
			}
		}
	})
}

func TestStats(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.FileName = "TEST.md"

	t.Run("section percentages", func(t *testing.T) {
		content := "## A\n\nline\nline\n\n## B\n\nline\nline\nline\nline\n"
		path := writeTempFile(t, dir, "stats.md", content)

		result, err := Stats(path, cfg)
		if err != nil {
			t.Fatal(err)
		}

		if len(result.Sections) != 2 {
			t.Fatalf("expected 2 sections, got %d", len(result.Sections))
		}

		// Percentages should add up to ~100%
		totalPct := 0.0
		for _, s := range result.Sections {
			totalPct += s.Percentage
		}

		// Sections may not cover the entire file (pre-section content),
		// but they should be reasonable
		if totalPct < 50 || totalPct > 100 {
			t.Errorf("expected percentages roughly totaling 100%%, got %.1f%%", totalPct)
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := Stats(filepath.Join(dir, "nope.md"), cfg)
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})

	t.Run("with rules files", func(t *testing.T) {
		rulesDir := filepath.Join(dir, "rules")
		os.MkdirAll(rulesDir, 0755)
		writeTempFile(t, rulesDir, "test.md", "# Test\n")

		cfgWithRules := cfg
		cfgWithRules.RulesDir = rulesDir

		content := "## Section\n\nline\n"
		path := writeTempFile(t, dir, "withrules.md", content)

		result, err := Stats(path, cfgWithRules)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.RulesFiles) != 1 {
			t.Errorf("expected 1 rules file, got %d", len(result.RulesFiles))
		}
	})
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
