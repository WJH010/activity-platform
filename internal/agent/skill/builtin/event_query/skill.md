---
name: event_query
display_name: 活动查询
category: event
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
      description: 每页数量，默认10，最大100
      default: 10
    event_status:
      type: string
      description: "活动状态筛选，可选值：未开始、正在进行、已结束"
    event_title:
      type: string
      description: 按活动标题关键词搜索
  required: []
---

查询活动列表，支持按状态和标题筛选。返回活动的ID、标题、时间、地址、费用等基本信息。

## 使用场景

- 用户想了解有哪些活动可以参加
- 用户想查找特定状态的活动（如"正在进行"的活动）
- 用户想搜索包含某个关键词的活动
- 用户问"最近有什么活动"或"有什么活动可以报名"

## 返回内容

返回活动列表，每个活动包含：
- id: 活动ID
- title: 活动标题
- event_start_time / event_end_time: 活动开始/结束时间
- registration_start_time / registration_end_time: 报名开始/截止时间
- event_address: 活动地址
- registration_fee: 报名费用
- status: 活动状态
- member_count: 已报名人数

## 注意事项

- 不传status时返回所有状态的活动
- 如需报名活动，请先使用event_detail查看详情确认，再使用event_register报名
