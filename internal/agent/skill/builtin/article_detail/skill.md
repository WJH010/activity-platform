---
name: article_detail
display_name: 文章详情
category: article
auth_required: false
parameters:
  type: object
  properties:
    article_id:
      type: integer
      description: 文章ID
  required:
    - article_id
---

查询指定文章的完整内容，包括正文、图片等。

## 使用场景

- 用户想阅读某篇文章的完整内容
- 从文章列表中选择后查看详情
- 用户提到某个文章ID

## 返回内容

返回文章完整信息：
- article_id: 文章ID
- article_title: 文章标题
- brief_content: 摘要
- article_content: 正文内容
- article_type: 文章类型
- field_name: 领域名称
- release_time: 发布时间
- article_source: 文章来源
- images: 图片列表

## 注意事项

- article_id可通过article_query查询获得
