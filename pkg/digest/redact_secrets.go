package digest

import (
	"strings"

	"github.com/zricethezav/gitleaks/v8/detect"
)

var secretDetector *detect.Detector

func init() {
	secretDetector, _ = detect.NewDetectorDefaultConfig()
}

// ReplaceSecrets detects and replaces secrets in the subject string, e.g. with "[REDACTED]".
func ReplaceSecrets(s, replacement string) string {
	findings := secretDetector.DetectString(s)
	for _, f := range findings {
		s = strings.ReplaceAll(s, f.Secret, replacement)
	}
	return s
}
