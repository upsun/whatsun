package digest

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveComments(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		input    string
		wantNot  []string // strings that should be removed
		wantKeep []string // strings that should remain
	}{
		{
			name:     "Makefile with comments",
			filename: "Makefile",
			input: `
# This is a comment
build:
	echo "Building..." # Inline comment
`,
			wantNot:  []string{"# This is a comment", "# Inline comment"},
			wantKeep: []string{`echo "Building..."`},
		},
		{
			name:     "JSON with JS-style comments",
			filename: "config.json",
			input: `
{
  // user info
  "user": "admin"
  /* This block is ignored */
}
`,
			wantNot:  []string{"// user info", "/* This block is ignored */"},
			wantKeep: []string{`"user": "admin"`},
		},
		{
			name:     "YAML with comments",
			filename: "settings.yaml",
			input: `
# Top-level settings
database:
  user: admin # inline user
`,
			wantNot:  []string{"# Top-level settings", "# inline user"},
			wantKeep: []string{"user: admin"},
		},
		{
			name:     "PHP with all types of comments",
			filename: "index.php",
			input: `
// This is a line comment
# Another one
echo "Hello"; /* inline block */
`,
			wantNot:  []string{"//", "#", "/* inline block */"},
			wantKeep: []string{`echo "Hello";`},
		},
		{
			name:     "Markdown with HTML-style comments",
			filename: "README.md",
			input: `
# Project Title

<!-- This comment should not be rendered -->

Some description.
`,
			wantNot:  []string{"<!-- This comment should not be rendered -->"},
			wantKeep: []string{"Project Title", "Some description"},
		},
		{
			name:     "HTML with block comment",
			filename: "index.html",
			input: `
<html>
<!-- TODO: remove debug info -->
<body>Hello</body>
</html>
`,
			wantNot:  []string{"<!-- TODO: remove debug info -->"},
			wantKeep: []string{"<body>Hello</body>"},
		},
		{
			name:     "Python with comments",
			filename: "script.py",
			input: `
# This is a Python script
print("hi")  # print greeting
`,
			wantNot:  []string{"# This is a Python script", "# print greeting"},
			wantKeep: []string{`print("hi")`},
		},
		{
			name:     "HCL with comments",
			filename: "main.hcl",
			input: `
# Terraform config
resource "aws_s3_bucket" "bucket" {
  bucket = "my-bucket" # Inline comment
}
`,
			wantNot:  []string{"# Terraform config", "# Inline comment"},
			wantKeep: []string{`bucket = "my-bucket"`},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ext := strings.ToLower(filepath.Ext(c.filename))
			base := strings.ToLower(filepath.Base(c.filename))
			t.Log(base, ext)

			clean := RemoveComments(c.filename, c.input)

			for _, s := range c.wantNot {
				if strings.Contains(clean, s) {
					t.Errorf("expected comment to be removed: %q", s)
				}
			}
			for _, s := range c.wantKeep {
				if !strings.Contains(clean, s) {
					t.Errorf("expected content to remain: %q", s)
				}
			}
		})
	}
}
