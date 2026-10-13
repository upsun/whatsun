package digest

import (
	"strings"
	"sync"

	"github.com/zricethezav/gitleaks/v8/detect"
)

// secretDetector is built on first use, as it is expensive to initialize.
var secretDetector = sync.OnceValue(func() *detect.Detector {
	d, _ := detect.NewDetectorDefaultConfig()
	return d
})

// ReplaceSecrets detects and replaces secrets in the subject string, e.g. with "[REDACTED]".
func ReplaceSecrets(s, replacement string) string {
	findings := secretDetector().DetectString(s)
	for _, f := range findings {
		s = strings.ReplaceAll(s, f.Secret, replacement)
	}
	return s
}
