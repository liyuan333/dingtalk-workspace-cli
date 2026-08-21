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
	"bytes"
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/pkg/edition"
)

func TestCrossPlatformCoverageMaskSensitivePreviewArgs(t *testing.T) {
	if MaskSensitivePreviewArgs(nil) != nil {
		t.Fatal("nil args must stay nil")
	}

	args := map[string]any{
		"nodeId":   "doc-1",
		"password": "pw-secret",
		"auth":     map[string]any{"clientSecret": "cs-secret", "region": "cn"},
		"items":    []any{map[string]any{"accessToken": "at-secret", "name": "n"}, "plain"},
		"paging":   map[string]any{"nextToken": "cursor-1", "next_token": "cursor-2"},
		"count":    3,
	}
	masked := MaskSensitivePreviewArgs(args)

	want := map[string]any{
		"nodeId":   "doc-1",
		"password": "[REDACTED]",
		"auth":     map[string]any{"clientSecret": "[REDACTED]", "region": "cn"},
		"items":    []any{map[string]any{"accessToken": "[REDACTED]", "name": "n"}, "plain"},
		"paging":   map[string]any{"nextToken": "cursor-1", "next_token": "cursor-2"},
		"count":    3,
	}
	if !reflect.DeepEqual(masked, want) {
		t.Fatalf("masked = %#v, want %#v", masked, want)
	}

	// 真实调用路径必须继续拿到原值：输入 map（含嵌套）不能被掩码修改。
	if args["password"] != "pw-secret" ||
		args["auth"].(map[string]any)["clientSecret"] != "cs-secret" ||
		args["items"].([]any)[0].(map[string]any)["accessToken"] != "at-secret" {
		t.Fatalf("original args mutated: %#v", args)
	}

	// 敏感键下的非字符串值也替换为固定掩码，不保留结构。
	nested := MaskSensitivePreviewArgs(map[string]any{"credentials": map[string]any{"user": "u"}})
	if nested["credentials"] != "[REDACTED]" {
		t.Fatalf("non-string sensitive value = %#v", nested["credentials"])
	}
}

type sensitivePreviewCaller struct {
	format string
	calls  int
}

func (c *sensitivePreviewCaller) CallTool(_ context.Context, _, _ string, _ map[string]any) (*edition.ToolResult, error) {
	c.calls++
	return &edition.ToolResult{Content: []edition.ContentBlock{{Type: "text", Text: "{}"}}}, nil
}

func (c *sensitivePreviewCaller) Format() string { return c.format }
func (*sensitivePreviewCaller) DryRun() bool     { return true }
func (*sensitivePreviewCaller) Fields() string   { return "" }
func (*sensitivePreviewCaller) JQ() string       { return "" }

func runDocReadDryRunPreview(t *testing.T, format string) (string, int) {
	t.Helper()
	caller := &sensitivePreviewCaller{format: format}
	previousDeps := deps
	previousArgs := os.Args
	var out bytes.Buffer
	InitDeps(caller)
	deps.Out.w = &out
	deps.Out.errW = &out
	os.Args = []string{"dws", "doc"}
	t.Cleanup(func() {
		deps = previousDeps
		os.Args = previousArgs
	})
	root := newDocCommand()
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{"read", "--node", "doc-1", "--password", "pw-secret", "--version", "7"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doc read dry-run (%s): %v", format, err)
	}
	return out.String(), caller.calls
}

func TestCrossPlatformCoverageDocReadDryRunMasksPassword(t *testing.T) {
	t.Run("json preview", func(t *testing.T) {
		output, calls := runDocReadDryRunPreview(t, "json")
		if calls != 0 {
			t.Fatalf("dry-run executed %d MCP calls", calls)
		}
		if strings.Contains(output, "pw-secret") {
			t.Fatalf("structured dry-run preview leaked the password: %s", output)
		}
		if !strings.Contains(output, `"password": "[REDACTED]"`) ||
			!strings.Contains(output, `"nodeId": "doc-1"`) ||
			!strings.Contains(output, `"historyVersion": 7`) {
			t.Fatalf("structured dry-run preview shape unexpected: %s", output)
		}
	})

	t.Run("table preview", func(t *testing.T) {
		output, calls := runDocReadDryRunPreview(t, "table")
		if calls != 0 {
			t.Fatalf("dry-run executed %d MCP calls", calls)
		}
		if strings.Contains(output, "pw-secret") {
			t.Fatalf("dry-run preview text leaked the password: %s", output)
		}
		if !strings.Contains(output, "get_document_content") || !strings.Contains(output, "[REDACTED]") {
			t.Fatalf("dry-run preview text shape unexpected: %s", output)
		}
	})
}
