package pipeline

import (
	"strings"
	"testing"
)

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "纯文本",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "简单标签",
			input:    "<p>Hello</p>",
			expected: "Hello",
		},
		{
			name:     "带属性标签",
			input:    `<div class="content">Text</div>`,
			expected: "Text",
		},
		{
			name:     "br标签转换行",
			input:    "Line1<br>Line2<br/>Line3",
			expected: "Line1\nLine2\nLine3",
		},
		{
			name:     "p标签转换行",
			input:    "<p>Para1</p><p>Para2</p>",
			expected: "Para1\nPara2",
		},
		{
			name:     "去除script块",
			input:    "<script>var x=1;</script>Text",
			expected: "Text",
		},
		{
			name:     "去除style块",
			input:    "<style>.a{color:red}</style>Text",
			expected: "Text",
		},
		{
			name:     "HTML实体",
			input:    "A&amp;B &lt;C&gt; &nbsp;D",
			expected: "A&B <C>  D",
		},
		{
			name:     "复杂HTML",
			input:    `<h1>标题</h1><p>段落一</p><div>段落二</div>`,
			expected: "标题\n段落一\n段落二",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripHTMLTags(tt.input)
			result = strings.TrimSpace(result)
			if result != tt.expected {
				t.Errorf("StripHTMLTags() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "连续空行",
			input:    "A\n\n\n\nB",
			expected: "A\n\nB",
		},
		{
			name:     "行内多空格",
			input:    "A   B   C",
			expected: "A B C",
		},
		{
			name:     "行首尾空格",
			input:    "  A  \n  B  ",
			expected: "A\nB",
		},
		{
			name:     "Tab转空格",
			input:    "A\tB",
			expected: "A B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWhitespace(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeWhitespace() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestProcessDocument(t *testing.T) {
	tp := NewTextProcessor()

	t.Run("空内容", func(t *testing.T) {
		chunks := tp.ProcessDocument("标题", "", "")
		if len(chunks) != 0 {
			t.Errorf("空内容应返回0个chunk，实际%d", len(chunks))
		}
	})

	t.Run("仅摘要", func(t *testing.T) {
		chunks := tp.ProcessDocument("标题", "", "这是摘要内容")
		if len(chunks) != 1 {
			t.Fatalf("应返回1个chunk，实际%d", len(chunks))
		}
		if chunks[0].Index != 0 {
			t.Errorf("摘要chunk的Index应为0，实际%d", chunks[0].Index)
		}
		if !strings.Contains(chunks[0].Content, "标题") {
			t.Error("摘要chunk应包含标题")
		}
		if !strings.Contains(chunks[0].Content, "这是摘要内容") {
			t.Error("摘要chunk应包含摘要")
		}
	})

	t.Run("仅正文_短文本", func(t *testing.T) {
		chunks := tp.ProcessDocument("标题", "<p>正文内容</p>", "")
		if len(chunks) != 1 {
			t.Fatalf("短正文应返回1个chunk，实际%d", len(chunks))
		}
	})

	t.Run("摘要+正文", func(t *testing.T) {
		chunks := tp.ProcessDocument("标题", "<p>正文内容</p>", "摘要")
		if len(chunks) != 2 {
			t.Fatalf("摘要+正文应返回2个chunk，实际%d", len(chunks))
		}
		if chunks[0].Index != 0 {
			t.Errorf("第一个chunk的Index应为0，实际%d", chunks[0].Index)
		}
		if chunks[1].Index != 1 {
			t.Errorf("第二个chunk的Index应为1，实际%d", chunks[1].Index)
		}
	})

	t.Run("超长正文自动分块", func(t *testing.T) {
		longText := strings.Repeat("这是一个很长的段落。", 200) // 约2000字
		chunks := tp.ProcessDocument("标题", longText, "")
		if len(chunks) <= 1 {
			t.Errorf("超长正文应分为多个chunk，实际%d", len(chunks))
		}
		// 验证每个chunk不超过maxChunkSize的1.5倍（允许句子边界超出）
		for i, c := range chunks {
			runeCount := len([]rune(c.Content))
			if runeCount > 1500 {
				t.Errorf("chunk[%d] 长度 %d 超出合理范围", i, runeCount)
			}
		}
	})

	t.Run("多段落分块", func(t *testing.T) {
		paragraphs := make([]string, 10)
		for i := range paragraphs {
			paragraphs[i] = strings.Repeat("段落内容。", 30) // 每段约150字
		}
		html := strings.Join(paragraphs, "\n\n")
		chunks := tp.ProcessDocument("标题", html, "")
		if len(chunks) < 3 {
			t.Errorf("多段落应分为多个chunk，实际%d", len(chunks))
		}
	})

	t.Run("HTML清洗与分块", func(t *testing.T) {
		html := `<h1>标题</h1><p>第一段内容，包含<strong>加粗</strong>文字。</p><p>第二段内容，包含<a href="#">链接</a>。</p>`
		chunks := tp.ProcessDocument("标题", html, "")
		if len(chunks) == 0 {
			t.Fatal("应返回至少1个chunk")
		}
		for _, c := range chunks {
			if strings.Contains(c.Content, "<") || strings.Contains(c.Content, ">") {
				t.Errorf("chunk中不应包含HTML标签: %q", c.Content)
			}
		}
	})
}
