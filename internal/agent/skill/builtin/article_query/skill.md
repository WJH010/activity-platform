---
name: article_query
display_name: 文章查询
category: article
auth_required: false
parameters:
  type: object
  properties:
    page:
      type: integer
      description: 页码，默认1
      default: 1
    page_size:
      type: integer
      description: 每页数量，默认10
      default: 10
    article_title:
      type: string
      description: 按文章标题关键词搜索
    article_type:
      type: string
      description: 按文章类型筛选
    field_type:
      type: string
      description: 按领域类型筛选
  required: []
---

查询文章列表，支持按标题、类型、领域筛选。返回文章的ID、标题、摘要、类型等基本信息。

## 使用场景

- 用户想浏览文章列表
- 用户想搜索特定主题的文章
- 用户问"有什么文章"、"推荐一些文章"

## 返回内容

返回文章列表，每篇包含：
- article_id: 文章ID
- article_title: 文章标题
- article_type: 文章类型
- field_name: 领域名称
- release_time: 发布时间
- brief_content: 摘要
- is_selection: 是否精选
- cover_image_url: 封面图

## 注意事项

- 查看文章完整内容请使用article_detail
