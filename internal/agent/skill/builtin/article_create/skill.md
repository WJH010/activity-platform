---
name: article_create
display_name: 创建文章
category: article
auth_required: true
parameters:
  type: object
  properties:
    article_title:
      type: string
      description: 文章标题
    article_type:
      type: string
      description: 文章类型
    article_content:
      type: string
      description: 文章正文内容（超文本格式）
    brief_content:
      type: string
      description: 文章摘要（如未提供，将从正文截取）
    field_type:
      type: string
      description: 领域类型代码
    article_source:
      type: string
      description: 文章来源
    cover_image_url:
      type: string
      description: 封面图片URL
    is_selection:
      type: integer
      description: 是否精选，1=精选，0=非精选，默认0
      default: 0
  required:
    - article_title
    - article_type
    - article_content
---

创建一篇新文章。

## 使用场景

- 用户想创建/发布一篇文章
- AI辅助编辑完成后，用户确认发布
- 用户说"帮我创建文章"、"把这段内容发布为文章"等

## 使用流程

1. 确认文章标题、类型和内容
2. 如果用户未提供摘要，可从正文自动提取
3. 调用此Skill创建文章
4. 创建成功后告知用户

## 注意事项

- article_title、article_type、article_content 为必填项
- article_type 常见值：新闻、公告、资讯、教程等
- 必须先登录才能创建文章
