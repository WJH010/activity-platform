---
name: event_detail
display_name: 活动详情
category: event
auth_required: false
parameters:
  type: object
  properties:
    event_id:
      type: integer
      description: 活动ID
  required:
    - event_id
---

查询指定活动的详细信息，包括活动内容、时间、地点、费用、图片等。

## 使用场景

- 用户想了解某个活动的详细信息
- 在报名前需要确认活动详情
- 用户提到某个活动ID或从列表中选择了某个活动

## 返回内容

返回活动的完整详情，包括：
- title: 活动标题
- detail: 活动详细介绍
- event_start_time / event_end_time: 活动时间
- registration_start_time / registration_end_time: 报名时间
- event_address: 活动地址
- registration_fee: 报名费用
- status: 活动状态
- images: 活动图片列表

## 注意事项

- event_id可通过event_query查询获得
- 获取详情后，如用户想报名，使用event_register
