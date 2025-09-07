package digest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceSecrets(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string // only one file per test for simplicity
	}{
		{
			name:     "Redact AWS Access Key",
			input:    "AWS Access Key: AKIA1234567890ABCD",
			expected: "AWS Access Key: [REDACTED]",
		},
		{
			name:     "Redact generic API key",
			input:    "API_KEY=abcd1234efgh5678",
			expected: "API_KEY=[REDACTED]",
		},
		{
			name: "Redact AWS Secret Key in context",
			input: `# AWS Configuration
aws_access_key_id = AKIACKN7JHWLFFLRIAYK
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG+bPxRfiCYsRybJpK+9f
region = us-west-2`,
			expected: `# AWS Configuration
aws_access_key_id = [REDACTED]
aws_secret_access_key = [REDACTED]
region = us-west-2`,
		},
		{
			name: "Redact Slack token inline",
			input: `#!/bin/bash
export SLACK_TOKEN=xoxb-123456789012-ABCDEFG1234567
echo "Token set"`,
			expected: `#!/bin/bash
export SLACK_TOKEN=[REDACTED]
echo "Token set"`,
		},
		{
			name: "Redact GitHub personal access token in script",
			input: `# Deploy script
curl -H "Authorization: token ghp_thei5ieveewohneiw1Si0luo0boo5wei2eiM" https://api.github.com/user/repos`,
			expected: `# Deploy script
curl -H "Authorization: token [REDACTED]" https://api.github.com/user/repos`,
		},
		{
			name: "Multiple secrets in one file",
			input: `DB_PASSWORD=mydbpassword
API_KEY=sk_test_4eC39HqLyjWDarjtT1zdp7dc
SECRET_TOKEN=abcdef123456`,
			expected: `DB_PASSWORD=mydbpassword
API_KEY=[REDACTED]
SECRET_TOKEN=[REDACTED]`,
		},

		{
			name:     "No secrets",
			input:    "Nothing to see here.",
			expected: "Nothing to see here.",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, ReplaceSecrets(c.input, "[REDACTED]"))
		})
	}
}
