package files

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceEmails(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single email",
			input:    "Contact: john.doe@example.com for support",
			expected: "Contact: redacted@example.org for support",
		},
		{
			name:     "Multiple emails",
			input:    "Email admin@company.org or support@help.net",
			expected: "Email redacted@example.org or redacted@example.org",
		},
		{
			name: "Email in code comment",
			input: `# Contact maintainer at maintainer@project.dev
			func main() {`,
			expected: `# Contact maintainer at redacted@example.org
			func main() {`,
		},
		{
			name: "Email in config file",
			input: `smtp:
  username: noreply@company.com
  password: secret123`,
			expected: `smtp:
  username: redacted@example.org
  password: secret123`,
		},
		{
			name:     "Email with plus sign",
			input:    "test+tag@gmail.com is valid",
			expected: "redacted@example.org is valid",
		},
		{
			name:     "Email with numbers and dots",
			input:    "user.123@sub.domain.co.uk works fine",
			expected: "redacted@example.org works fine",
		},
		{
			name:     "No emails",
			input:    "This text has no email addresses",
			expected: "This text has no email addresses",
		},
		{
			name:     "Almost email but invalid",
			input:    "Not an email: @domain.com or user@ or .com",
			expected: "Not an email: @domain.com or user@ or .com",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, ReplaceEmails(c.input, "redacted@example.org"))
		})
	}
}
