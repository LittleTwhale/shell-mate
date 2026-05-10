package search

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SearchResult 单条搜索结果摘要
type SearchResult struct {
	Title   string // 标题
	Snippet string // 内容摘要
	URL     string // 来源 URL
}

// duckDuckGoResponse DuckDuckGo Instant Answer API 响应格式
type duckDuckGoResponse struct {
	AbstractText  string `json:"AbstractText"`
	AbstractURL   string `json:"AbstractURL"`
	Heading       string `json:"Heading"`
	RelatedTopics []struct {
		FirstURL string `json:"FirstURL"`
		Text     string `json:"Text"`
	} `json:"RelatedTopics"`
}

// ========== 多语言格式化标签 ==========

// flattenLabels 按语言存储 FlattenResults 的格式化标签
var flattenLabels = map[string]struct {
	resultN  string // "搜索结果 #%d:\n" / "Search result #%d:\n"
	title    string // "标题: %s\n" / "Title: %s\n"
	snippet  string // "摘要: %s\n" / "Snippet: %s\n"
	source   string // "来源: %s\n" / "Source: %s\n"
	noResult string // "（无搜索结果）" / "(no search results)"
}{
	"zh": {
		resultN:  "搜索结果 #%d:\n",
		title:    "标题: %s\n",
		snippet:  "摘要: %s\n",
		source:   "来源: %s\n",
		noResult: "（无搜索结果）",
	},
	"en": {
		resultN:  "Search result #%d:\n",
		title:    "Title: %s\n",
		snippet:  "Snippet: %s\n",
		source:   "Source: %s\n",
		noResult: "(no search results)",
	},
}

// getFlattenLabels 根据语言获取格式化标签，默认为中文
func getFlattenLabels(lang string) struct {
	resultN  string
	title    string
	snippet  string
	source   string
	noResult string
} {
	if l, ok := flattenLabels[lang]; ok {
		return l
	}
	return flattenLabels["zh"]
}

// Search 使用无 Key 搜索 API 搜索，返回前 N 条结果摘要
// 优先尝试 DuckDuckGo（国际），失败后回退到 Bing（国内可达）
func Search(query string, maxResults int) ([]SearchResult, error) {
	// 先尝试 DuckDuckGo Instant Answer API
	results, err := searchDuckDuckGo(query, maxResults)
	if err == nil && len(results) > 0 {
		return results, nil
	}

	// 回退到 Bing HTML 搜索
	return searchBing(query, maxResults)
}

// searchDuckDuckGo 使用 DuckDuckGo Instant Answer API（免费，无需 Key）
func searchDuckDuckGo(query string, maxResults int) ([]SearchResult, error) {
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1",
		url.QueryEscape(query))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create DuckDuckGo request failed: %w", err)
	}
	req.Header.Set("User-Agent", "shell-mate/1.0 (CLI AI assistant; github.com/whalechen/shell-mate)")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("DuckDuckGo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DuckDuckGo returned status: %d", resp.StatusCode)
	}

	var ddgResp duckDuckGoResponse
	if err := json.NewDecoder(resp.Body).Decode(&ddgResp); err != nil {
		return nil, fmt.Errorf("parse DuckDuckGo response failed: %w", err)
	}

	var results []SearchResult

	if ddgResp.AbstractText != "" {
		title := ddgResp.Heading
		if title == "" {
			title = query
		}
		results = append(results, SearchResult{
			Title:   title,
			Snippet: ddgResp.AbstractText,
			URL:     ddgResp.AbstractURL,
		})
	}

	for _, topic := range ddgResp.RelatedTopics {
		if len(results) >= maxResults {
			break
		}
		if topic.Text == "" {
			continue
		}
		results = append(results, SearchResult{
			Title:   "",
			Snippet: topic.Text,
			URL:     topic.FirstURL,
		})
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("DuckDuckGo returned no relevant results")
	}

	return results, nil
}

// searchBing 使用必应搜索（国内版），通过 HTML 解析提取结果摘要
func searchBing(query string, maxResults int) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("https://cn.bing.com/search?q=%s&setlang=zh-cn",
		url.QueryEscape(query))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create Bing request failed: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Bing request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bing returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Bing response failed: %w", err)
	}

	results := parseBingHTML(string(body), maxResults)
	if len(results) == 0 {
		return nil, fmt.Errorf("Bing search returned no results")
	}

	return results, nil
}

// parseBingHTML 从必应搜索结果页面中提取标题、摘要和 URL
// 使用字符串操作而非 HTML 解析库，避免引入额外依赖
func parseBingHTML(html string, maxResults int) []SearchResult {
	var results []SearchResult

	// 按结果块切分：必应的每条结果都在 <li class="b_algo" ...> 块中
	for _, chunk := range strings.Split(html, "<li class=\"b_algo\"") {
		if len(results) >= maxResults {
			break
		}

		// 提取标题和 URL：<h2 ...><a href="URL">TITLE</a></h2>
		title, url := "", ""
		h2Start := strings.Index(chunk, "<h2")
		if h2Start != -1 {
			// 定位 h2 开标签的结束位置 >
			h2TagClose := strings.Index(chunk[h2Start:], ">")
			if h2TagClose != -1 {
				contentStart := h2Start + h2TagClose + 1
				h2End := strings.Index(chunk[contentStart:], "</h2>")
				if h2End != -1 {
					title = stripHTMLTags(chunk[contentStart : contentStart+h2End])
				}
			}
			// 从 h2 区域内提取 href 属性值
			hrefStart := strings.Index(chunk[h2Start:], "href=\"")
			if hrefStart != -1 {
				absHrefStart := h2Start + hrefStart + 6
				hrefEnd := strings.Index(chunk[absHrefStart:], "\"")
				if hrefEnd != -1 {
					url = chunk[absHrefStart : absHrefStart+hrefEnd]
				}
			}
		}

		// 提取摘要：<div class="b_caption"><p ...>SNIPPET</p></div>
		snippet := ""
		capStart := strings.Index(chunk, "b_caption")
		if capStart != -1 {
			// 查找 <p 标签（可能带有 class 等属性）
			pStart := strings.Index(chunk[capStart:], "<p")
			if pStart != -1 {
				// 定位 p 开标签的结束位置 >
				pTagClose := strings.Index(chunk[capStart+pStart:], ">")
				if pTagClose != -1 {
					contentStart := capStart + pStart + pTagClose + 1
					pEnd := strings.Index(chunk[contentStart:], "</p>")
					if pEnd != -1 {
						snippet = stripHTMLTags(chunk[contentStart : contentStart+pEnd])
					}
				}
			}
		}

		if snippet != "" {
			if title == "" {
				title = snippet
				if len(title) > 40 {
					title = title[:40] + "..."
				}
			}
			results = append(results, SearchResult{
				Title:   title,
				Snippet: snippet,
				URL:     url,
			})
		}
	}

	return results
}

// stripHTMLTags 移除字符串中所有的 HTML 标签，仅保留纯文本
func stripHTMLTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	// 清理常见的 HTML 实体和多余空白
	result := strings.TrimSpace(b.String())
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&#39;", "'")
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	return result
}

// FlattenResults 将多条搜索结果拼接为一段纯文本，供 LLM 二次调用使用
// lang: 当前语言 (zh/en)，决定输出标签的语言
func FlattenResults(results []SearchResult, lang string) string {
	labels := getFlattenLabels(lang)

	if len(results) == 0 {
		return labels.noResult
	}

	var s string
	for i, r := range results {
		s += fmt.Sprintf(labels.resultN, i+1)
		if r.Title != "" {
			s += fmt.Sprintf(labels.title, r.Title)
		}
		s += fmt.Sprintf(labels.snippet, r.Snippet)
		if r.URL != "" {
			s += fmt.Sprintf(labels.source, r.URL)
		}
		s += "\n"
	}
	return s
}
