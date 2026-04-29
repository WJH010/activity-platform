---
name: user_info
display_name: 用户信息查询
category: user
auth_required: true
parameters:
  type: object
  properties: {}
  required: []
---

查询当前登录用户的个人信息。

## 使用场景

- 用户想查看自己的个人信息
- 需要获取用户信息来辅助其他操作
- 用户问"我的信息是什么"、"我叫什么"等

## 返回内容

返回用户详细信息：
- user_id: 用户ID
- nickname: 昵称
- name: 真实姓名
- phone_number: 手机号
- email: 邮箱
- unit: 单位
- department: 部门
- position: 职位
- industry: 行业
- role: 角色

## 注意事项

- 必须先登录
- 无需传入参数，自动获取当前用户信息
