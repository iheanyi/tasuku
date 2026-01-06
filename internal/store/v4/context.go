package v4

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/iheanyi/tasuku/internal/task"
)

// LearningsFile represents the parsed learnings.md file.
type LearningsFile struct {
	Learnings []task.Learning
}

// DecisionsFile represents the parsed decisions.md file.
type DecisionsFile struct {
	Decisions []task.Decision
}

var (
	// learningHeaderRegex matches "## <id> - YYYY-MM-DD"
	learningHeaderRegex = regexp.MustCompile(`^##\s+(\S+)\s+-\s+(\d{4}-\d{2}-\d{2})$`)
	// decisionHeaderRegex matches "## <id> - YYYY-MM-DD"
	decisionHeaderRegex = regexp.MustCompile(`^##\s+(\S+)\s+-\s+(\d{4}-\d{2}-\d{2})$`)
	// decisionFieldRegex matches "**Label**: Value"
	decisionFieldRegex = regexp.MustCompile(`^\*\*(\w+)\*\*:\s*(.+)$`)
)

// ParseLearningsFile parses a learnings.md file.
// Format:
//
//	# Learnings
//
//	## <id> - YYYY-MM-DD
//	Learning text with optional code blocks.
//
//	## <id2> - YYYY-MM-DD
//	Another learning.
func ParseLearningsFile(content []byte) (*LearningsFile, error) {
	result := &LearningsFile{
		Learnings: []task.Learning{},
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	var currentLearning *task.Learning
	var currentContent []string

	for scanner.Scan() {
		line := scanner.Text()

		// Skip main title
		if strings.HasPrefix(line, "# ") {
			continue
		}

		// Check for learning header
		if match := learningHeaderRegex.FindStringSubmatch(line); match != nil {
			// Save previous learning
			if currentLearning != nil {
				currentLearning.Text = strings.TrimSpace(strings.Join(currentContent, "\n"))
				currentLearning.IsRule = task.IsRuleLearning(currentLearning.Text)
				result.Learnings = append(result.Learnings, *currentLearning)
			}

			// Start new learning
			id := match[1]
			dateStr := match[2]
			date, _ := time.Parse("2006-01-02", dateStr)

			currentLearning = &task.Learning{
				ID:        id,
				CreatedAt: date,
			}
			currentContent = nil
			continue
		}

		// Accumulate content for current learning
		if currentLearning != nil {
			currentContent = append(currentContent, line)
		}
	}

	// Save final learning
	if currentLearning != nil {
		currentLearning.Text = strings.TrimSpace(strings.Join(currentContent, "\n"))
		currentLearning.IsRule = task.IsRuleLearning(currentLearning.Text)
		result.Learnings = append(result.Learnings, *currentLearning)
	}

	return result, nil
}

// WriteLearningsFile generates Markdown content for learnings.
func WriteLearningsFile(learnings []task.Learning) []byte {
	var buf bytes.Buffer

	buf.WriteString("# Learnings\n\n")

	for _, l := range learnings {
		// Format: ## <id> - YYYY-MM-DD
		dateStr := l.CreatedAt.Format("2006-01-02")
		if l.CreatedAt.IsZero() {
			dateStr = time.Now().Format("2006-01-02")
		}

		buf.WriteString(fmt.Sprintf("## %s - %s\n", l.ID, dateStr))
		buf.WriteString(l.Text)
		buf.WriteString("\n\n")
	}

	return buf.Bytes()
}

// ParseDecisionsFile parses a decisions.md file.
// Format:
//
//	# Decisions
//
//	## <id> - YYYY-MM-DD
//	**Chose**: Option A
//	**Over**: Option B, Option C
//	**Because**: Reasoning text that can span multiple lines.
func ParseDecisionsFile(content []byte) (*DecisionsFile, error) {
	result := &DecisionsFile{
		Decisions: []task.Decision{},
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	var currentDecision *task.Decision
	var inBecause bool
	var becauseContent []string

	for scanner.Scan() {
		line := scanner.Text()

		// Skip main title
		if strings.HasPrefix(line, "# ") && !strings.HasPrefix(line, "## ") {
			continue
		}

		// Check for decision header
		if match := decisionHeaderRegex.FindStringSubmatch(line); match != nil {
			// Save previous decision
			if currentDecision != nil {
				if inBecause && len(becauseContent) > 0 {
					currentDecision.Because = strings.TrimSpace(strings.Join(becauseContent, "\n"))
				}
				result.Decisions = append(result.Decisions, *currentDecision)
			}

			// Start new decision
			id := match[1]
			dateStr := match[2]
			date, _ := time.Parse("2006-01-02", dateStr)

			currentDecision = &task.Decision{
				ID:        id,
				CreatedAt: date,
				Over:      []string{},
			}
			inBecause = false
			becauseContent = nil
			continue
		}

		if currentDecision == nil {
			continue
		}

		// Parse decision fields
		if match := decisionFieldRegex.FindStringSubmatch(line); match != nil {
			field := strings.ToLower(match[1])
			value := strings.TrimSpace(match[2])

			switch field {
			case "chose":
				currentDecision.Chose = value
				inBecause = false
			case "over":
				// Parse comma-separated alternatives
				parts := strings.Split(value, ",")
				for _, p := range parts {
					if trimmed := strings.TrimSpace(p); trimmed != "" {
						currentDecision.Over = append(currentDecision.Over, trimmed)
					}
				}
				inBecause = false
			case "because":
				becauseContent = []string{value}
				inBecause = true
			}
			continue
		}

		// Accumulate multi-line because content
		if inBecause {
			becauseContent = append(becauseContent, line)
		}
	}

	// Save final decision
	if currentDecision != nil {
		if inBecause && len(becauseContent) > 0 {
			currentDecision.Because = strings.TrimSpace(strings.Join(becauseContent, "\n"))
		}
		result.Decisions = append(result.Decisions, *currentDecision)
	}

	return result, nil
}

// WriteDecisionsFile generates Markdown content for decisions.
func WriteDecisionsFile(decisions []task.Decision) []byte {
	var buf bytes.Buffer

	buf.WriteString("# Decisions\n\n")

	for _, d := range decisions {
		// Format: ## <id> - YYYY-MM-DD
		dateStr := d.CreatedAt.Format("2006-01-02")
		if d.CreatedAt.IsZero() {
			dateStr = time.Now().Format("2006-01-02")
		}

		buf.WriteString(fmt.Sprintf("## %s - %s\n", d.ID, dateStr))
		buf.WriteString(fmt.Sprintf("**Chose**: %s\n", d.Chose))

		// Join alternatives with comma
		if len(d.Over) > 0 {
			buf.WriteString(fmt.Sprintf("**Over**: %s\n", strings.Join(d.Over, ", ")))
		}

		buf.WriteString(fmt.Sprintf("**Because**: %s\n", d.Because))
		buf.WriteString("\n")
	}

	return buf.Bytes()
}
