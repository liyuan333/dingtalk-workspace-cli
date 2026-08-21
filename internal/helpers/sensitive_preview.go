// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"strings"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/logging"
)

// maskedPreviewValue is the fixed mask that replaces credential-bearing
// argument values in dry-run previews. It mirrors the internal/output
// envelope redaction placeholder so masked previews read consistently.
const maskedPreviewValue = "[REDACTED]"

// MaskSensitivePreviewArgs returns a deep copy of args in which every value
// under a credential-bearing key (logging.IsSensitiveKey: password / secret /
// token / credential variants) is replaced by a fixed mask. Dry-run previews
// must never echo secrets such as document access passwords; the input map is
// left untouched so the real MCP call path keeps authentic values. Pagination
// cursors (next_token / nextToken) are opaque resumption handles, not
// credentials, and stay visible — the same boundary as internal/output
// envelope redaction.
func MaskSensitivePreviewArgs(args map[string]any) map[string]any {
	if args == nil {
		return nil
	}
	masked := make(map[string]any, len(args))
	for key, value := range args {
		masked[key] = maskSensitivePreviewValue(key, value)
	}
	return masked
}

func maskSensitivePreviewValue(key string, value any) any {
	if isSensitivePreviewKey(key) {
		return maskedPreviewValue
	}
	switch typed := value.(type) {
	case map[string]any:
		return MaskSensitivePreviewArgs(typed)
	case []any:
		masked := make([]any, len(typed))
		for index, item := range typed {
			masked[index] = maskSensitivePreviewValue("", item)
		}
		return masked
	}
	return value
}

func isSensitivePreviewKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}
	// Pagination cursors are resumption handles an Agent must be able to read
	// back from a preview, not credentials (same exemption as internal/output).
	if normalized == "next_token" || normalized == "nexttoken" {
		return false
	}
	return logging.IsSensitiveKey(normalized)
}
