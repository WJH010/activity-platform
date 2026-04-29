---
name: event_cancel
display_name: 取消活动报名
category: event
auth_required: true
parameters:
  type: object
  properties:
    event_id:
      type: integer
      description: 要取消报名的活动ID
  required:
    - event_id
---

取消当前登录用户在指定活动的报名。

## 使用场景

- 用户想取消已报名的活动
- 用户说"我不想去了"、"取消报名"等

## 注意事项

- 必须先登录
- 只能取消自己已报名的活动
- 未报名的活动无法取消
