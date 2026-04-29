---
name: event_register
display_name: 活动报名
category: event
auth_required: true
parameters:
  type: object
  properties:
    event_id:
      type: integer
      description: 要报名的活动ID
  required:
    - event_id
---

为当前登录用户报名指定活动。

## 使用场景

- 用户明确表示要报名某个活动
- 用户说"我要报名这个活动"、"帮我报名"等

## 使用流程

1. 先通过event_query或event_detail确认活动信息
2. 获取用户确认后，调用此Skill完成报名
3. 报名成功后告知用户

## 注意事项

- 必须先登录才能报名
- 重复报名会返回错误提示
- 只能报名状态为"正在进行"（报名中）的活动
- 报名前建议先确认活动详情，告知用户活动名称、时间、费用等信息
