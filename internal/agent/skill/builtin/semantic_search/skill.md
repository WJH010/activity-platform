---
name: semantic_search
display_name: 语义搜索
category: search
auth_required: false
parameters:
  type: object
  properties:
    query:
      type: string
      description: 搜索查询文本，支持自然语言描述
    search_type:
      type: string
      description: "搜索范围，可选值：article(仅文章)、event(仅活动)、all(全部，默认)"
      default: all
    top_k:
      type: integer
      description: 返回结果数量，默认5，最大20
      default: 5
  required:
    - query
---

基于语义理解搜索文章和活动内容，支持自然语言查询。结合向量语义检索和关键词匹配，比传统搜索更智能。

## 使用场景

- 用户用自然语言描述想找的内容，例如"有哪些关于人工智能的讲座"
- 传统关键词搜索无法满足的模糊语义查询
- 用户想找到内容相关但用词不同的文档
- 用户问"有没有和XX相关的内容"、"帮我找关于XX的文章/活动"

## 搜索方式

该工具使用混合检索策略：
- **语义检索**：理解查询含义，找到语义相关的内容
- **关键词匹配**：确保包含查询关键词的内容也能被召回
- **融合排序**：将两种检索结果智能融合，兼顾语义相关性和关键词匹配度

## 返回内容

每条结果包含：
- id: 文章ID或活动ID
- source_type: 结果类型(article/event)
- text_summary: 匹配的文本片段
- article_title/event_title: 标题
- 其他元数据（类型、状态、时间等）

## 注意事项

- 该工具适用于语义理解式搜索，精确标题匹配建议使用article_query或event_query
- 搜索范围默认为全部（文章+活动），可指定search_type缩小范围
- 返回的是内容片段而非完整文档，如需详情请使用article_detail或event_detail
