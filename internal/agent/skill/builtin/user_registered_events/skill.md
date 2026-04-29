---
name: user_registered_events
display_name: 已报名活动查询
category: user
auth_required: true
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
    event_status:
      type: string
      description: "按活动状态筛选，可选值：未开始、正在进行、已结束"
  required: []
---

查询当前登录用户已报名的活动列表。

## 使用场景

- 用户想查看自己报名了哪些活动
- 用户问"我报名了什么活动"、"我的活动"等
- 需要确认用户是否已报名某活动

## 返回内容

返回用户已报名的活动列表，每个活动包含：
- id: 活动ID
- title: 活动标题
- event_start_time / event_end_time: 活动时间
- event_address: 活动地址
- registration_fee: 报名费用
- status: 活动状态

## 注意事项

- 必须先登录
- 无需传入user_id，自动获取当前用户
