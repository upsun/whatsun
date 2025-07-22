package files

import (
	"regexp"
)

// ReplaceEmails tries to detect email addresses in the subject string and then replaces them.
var ReplaceEmails = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`).ReplaceAllString
