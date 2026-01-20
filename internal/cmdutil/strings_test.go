package cmdutil

import "testing"

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string unchanged",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length unchanged",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string truncated with ellipsis",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "maxLen 3 with ellipsis",
			input:    "hello",
			maxLen:   3,
			expected: "hel",
		},
		{
			name:     "maxLen 2 no ellipsis",
			input:    "hello",
			maxLen:   2,
			expected: "he",
		},
		{
			name:     "maxLen 1",
			input:    "hello",
			maxLen:   1,
			expected: "h",
		},
		{
			name:     "maxLen 0",
			input:    "hello",
			maxLen:   0,
			expected: "",
		},
		{
			name:     "unicode string (byte-based)",
			input:    "héllo wörld",
			maxLen:   8,
			expected: "héll...", // 'é' is 2 bytes, so 8 bytes = "héll" + "..."
		},
		{
			name:     "long description truncated",
			input:    "This is a very long task description that needs to be truncated",
			maxLen:   30,
			expected: "This is a very long task de...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestTruncate_LengthConstraint(t *testing.T) {
	// Verify truncated strings never exceed maxLen
	inputs := []string{
		"short",
		"medium length string",
		"this is a very long string that definitely needs truncation",
	}

	for _, input := range inputs {
		for maxLen := 0; maxLen <= 20; maxLen++ {
			result := Truncate(input, maxLen)
			if len(result) > maxLen {
				t.Errorf("Truncate(%q, %d) = %q (len %d), exceeds maxLen",
					input, maxLen, result, len(result))
			}
		}
	}
}
