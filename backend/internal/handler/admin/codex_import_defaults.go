package admin

import "github.com/Wei-Shaw/sub2api/internal/service"

const codexFingerprintModeExtraKey = "codex_fingerprint_mode"
const codexFingerprintModeSession = "session"

// forceImportedCodexFingerprintSession applies the import policy only to
// OpenAI OAuth accounts. A fresh copy avoids mutating the decoded request map.
func forceImportedCodexFingerprintSession(platform, accountType string, extra map[string]any) map[string]any {
	if platform != service.PlatformOpenAI || accountType != service.AccountTypeOAuth {
		return extra
	}

	out := make(map[string]any, len(extra)+1)
	for key, value := range extra {
		out[key] = value
	}
	out[codexFingerprintModeExtraKey] = codexFingerprintModeSession
	return out
}
