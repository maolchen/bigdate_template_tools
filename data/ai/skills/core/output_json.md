请只返回一个 JSON object，不要输出任何 JSON 之外的解释文本。

必须包含：
- `assistantMessage`
- `draftFiles[]`: `path`, `content`, `reason`
- `plannedActions[]`: `type`, `path`, `reason`
- `warnings[]`
- `followUpQuestions[]`
- `configPatch`: `global`, `serviceTop`, `serverConfig`

输出要求：
- `draftFiles[].path` 必须指向 `templates/<service>/*.tmpl`
- `draftFiles[].content` 必须是完整模板内容，不要只返回 diff 片段
- `plannedActions` 仅用于说明建议动作，不代表已经执行
- 信息不足时，写入 `followUpQuestions` 和 `warnings`
- 信息不明确时不要编造路径、变量、端口、节点分布
