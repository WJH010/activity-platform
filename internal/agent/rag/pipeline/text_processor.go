package pipeline

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// 预编译正则表达式，避免每次调用都重新编译
var (
	reScript    = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle     = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reBr        = regexp.MustCompile(`(?i)<br\s*/?\s*>`)
	reBlockEnd  = regexp.MustCompile(`(?i)</(?:p|div|h[1-6]|li|tr|blockquote|section|article)>`)
	reTags      = regexp.MustCompile(`<[^>]+>`)
	reMultiNewl = regexp.MustCompile(`\n{3,}`)
	reSpaces    = regexp.MustCompile(`[ \t]+`)
)

// Chunk 文本分块
type Chunk struct {
	Index   int    // 分块序号（0=摘要块）
	Content string // 清洗后的纯文本
}

// TextProcessor 文本处理器：HTML清洗 + 分块
type TextProcessor struct {
	chunkSize    int // 目标分块大小（字符数）
	maxChunkSize int // 单块最大大小（超过则按句子切分）
	minChunkSize int // 最小块大小（低于此值合并到前一块）
}

// NewTextProcessor 创建文本处理器
func NewTextProcessor() *TextProcessor {
	return &TextProcessor{
		chunkSize:    500,
		maxChunkSize: 1000,
		minChunkSize: 200,
	}
}

// ProcessDocument 处理单个文档（文章或活动）
// title: 标题, htmlContent: HTML正文, brief: 摘要
func (p *TextProcessor) ProcessDocument(title string, htmlContent string, brief string) []Chunk {
	var chunks []Chunk
	chunkIndex := 0

	// 1. 摘要块（chunk_index=0）：标题 + 摘要
	if brief != "" {
		cleanBrief := StripHTMLTags(brief)
		cleanBrief = NormalizeWhitespace(cleanBrief)
		if strings.TrimSpace(cleanBrief) != "" {
			content := title
			if cleanBrief != "" {
				content += "\n" + cleanBrief
			}
			chunks = append(chunks, Chunk{
				Index:   chunkIndex,
				Content: content,
			})
			chunkIndex++
		}
	}

	// 2. 正文分块
	if htmlContent != "" {
		plainText := StripHTMLTags(htmlContent)
		plainText = NormalizeWhitespace(plainText)
		plainText = strings.TrimSpace(plainText)

		if plainText != "" {
			bodyChunks := p.chunkByParagraphThenLength(plainText)
			for _, c := range bodyChunks {
				chunks = append(chunks, Chunk{
					Index:   chunkIndex,
					Content: c,
				})
				chunkIndex++
			}
		}
	}

	return chunks
}

// StripHTMLTags 去除HTML标签，保留纯文本
func StripHTMLTags(html string) string {
	// 去除<script>和<style>整个块
	html = reScript.ReplaceAllString(html, "")
	html = reStyle.ReplaceAllString(html, "")

	// <br>, <br/>, <br /> → 换行
	html = reBr.ReplaceAllString(html, "\n")

	// </p>, </div>, </h1>~</h6>, </li>, </tr> → 换行
	html = reBlockEnd.ReplaceAllString(html, "\n")

	// 去除所有剩余HTML标签
	html = reTags.ReplaceAllString(html, "")

	// HTML实体解码
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&quot;", `"`)
	html = strings.ReplaceAll(html, "&#39;", "'")
	html = strings.ReplaceAll(html, "&nbsp;", " ")
	html = strings.ReplaceAll(html, "&#8220;", `"`)
	html = strings.ReplaceAll(html, "&#8221;", `"`)
	html = strings.ReplaceAll(html, "&mdash;", "——")
	html = strings.ReplaceAll(html, "&ndash;", "–")
	html = strings.ReplaceAll(html, "&hellip;", "……")

	return html
}

// NormalizeWhitespace 规范化空白字符
func NormalizeWhitespace(text string) string {
	// 连续空行→双换行
	text = reMultiNewl.ReplaceAllString(text, "\n\n")

	// 每行内连续空格→单空格
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
		// 行内连续空白→单空格
		lines[i] = reSpaces.ReplaceAllString(lines[i], " ")
	}
	text = strings.Join(lines, "\n")

	return strings.TrimSpace(text)
}

// chunkByParagraphThenLength 段落优先 + 长度兜底分块
func (p *TextProcessor) chunkByParagraphThenLength(text string) []string {
	// 按 "\n\n" 拆分段落
	paragraphs := strings.Split(text, "\n\n")

	var chunks []string
	var currentChunk strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// 如果当前chunk + 这个段落不超过目标大小，合并
		if currentChunk.Len() > 0 && utf8.RuneCountInString(currentChunk.String())+utf8.RuneCountInString(para)+1 <= p.chunkSize {
			currentChunk.WriteString("\n")
			currentChunk.WriteString(para)
		} else {
			// 保存当前chunk
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
			}

			// 如果单个段落超长，按句子切分
			if utf8.RuneCountInString(para) > p.maxChunkSize {
				subChunks := p.splitBySentence(para)
				chunks = append(chunks, subChunks...)
			} else {
				currentChunk.WriteString(para)
			}
		}
	}

	// 保存最后一个chunk
	if currentChunk.Len() > 0 {
		lastChunk := currentChunk.String()
		// 如果最后一个chunk太短，合并到前一个
		if utf8.RuneCountInString(lastChunk) < p.minChunkSize && len(chunks) > 0 {
			chunks[len(chunks)-1] += "\n" + lastChunk
		} else {
			chunks = append(chunks, lastChunk)
		}
	}

	return chunks
}

// splitBySentence 按句子边界切分超长段落
func (p *TextProcessor) splitBySentence(text string) []string {
	runes := []rune(text)
	totalLen := len(runes)

	var chunks []string
	start := 0

	for start < totalLen {
		end := start + p.chunkSize
		if end >= totalLen {
			// 剩余部分不足一个chunk，直接作为最后一块
			remaining := string(runes[start:])
			if len(chunks) > 0 && utf8.RuneCountInString(remaining) < p.minChunkSize {
				chunks[len(chunks)-1] += remaining
			} else {
				chunks = append(chunks, remaining)
			}
			break
		}

		// 在chunkSize附近向后搜索最近的句子结尾
		splitPos := p.findSentenceBoundary(runes, end, totalLen)
		chunk := string(runes[start:splitPos])
		chunks = append(chunks, chunk)
		start = splitPos
	}

	return chunks
}

// findSentenceBoundary 从pos位置附近寻找最近的句子边界
// 优先在pos之后搜索（最多多100个字符），如果找不到则在pos之前搜索
func (p *TextProcessor) findSentenceBoundary(runes []rune, pos int, totalLen int) int {
	sentenceEnds := []rune{'。', '！', '？', '；', '.', '!', '?', ';', '\n'}

	// 先向后搜索（允许超出chunkSize最多100字符）
	searchEnd := pos + 100
	if searchEnd > totalLen {
		searchEnd = totalLen
	}

	bestPos := -1
	for i := pos; i < searchEnd; i++ {
		for _, sep := range sentenceEnds {
			if runes[i] == sep {
				bestPos = i + 1 // 包含分隔符
				break
			}
		}
		if bestPos != -1 {
			return bestPos
		}
	}

	// 向后找不到，从pos向前搜索
	searchStart := pos - p.minChunkSize
	if searchStart < 0 {
		searchStart = 0
	}
	for i := pos - 1; i >= searchStart; i-- {
		for _, sep := range sentenceEnds {
			if runes[i] == sep {
				return i + 1
			}
		}
	}

	// 前后都找不到，强制在pos处断开
	return pos
}
