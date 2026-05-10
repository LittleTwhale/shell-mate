package search

import (
	"testing"
)

func TestSearchBing(t *testing.T) {
	results, err := searchBing("kubernetes kubectl get pods sort by restart count", 3)
	if err != nil {
		t.Logf("必应搜索返回错误（可能是网络原因）: %v", err)
		t.Skip("跳过：网络不可达")
		return
	}

	if len(results) == 0 {
		t.Error("必应搜索应返回至少一条结果")
	}

	for i, r := range results {
		t.Logf("结果 #%d: Title=%q Snippet=%q URL=%q", i+1, r.Title, r.Snippet, r.URL)
		if r.Snippet == "" {
			t.Errorf("结果 #%d 的 Snippet 为空", i+1)
		}
	}
}

func TestFlattenResults(t *testing.T) {
	results := []SearchResult{
		{Title: "测试标题", Snippet: "这是一段测试摘要文本", URL: "https://example.com"},
		{Snippet: "无标题摘要"},
	}

	flattened := FlattenResults(results, "zh")
	if flattened == "" {
		t.Error("FlattenResults 不应返回空字符串")
	}
	t.Logf("扁平化结果:\n%s", flattened)
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<b>Hello</b> World", "Hello World"},
		{"<a href=\"url\">Click &amp; Go</a>", "Click & Go"},
		{"plain text", "plain text"},
		{"<div>nested <span>tags</span></div>", "nested tags"},
	}

	for _, tt := range tests {
		got := stripHTMLTags(tt.input)
		if got != tt.expected {
			t.Errorf("stripHTMLTags(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
