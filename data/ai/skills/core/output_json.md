请只返回一个 JSON object (reply with exactly one JSON object)，必须包含：
- `assistantMessage`
- `draftFiles[]`: `path`, `content`, `reason`
- `plannedActions[]`: `type`, `path`, `reason`
- `warnings[]`
- `followUpQuestions[]`
- `configPatch`: `serviceTop`, `serverConfig`

信息不足时，请在 `followUpQuestions` 和 `warnings` 说明 (if information is missing, do not invent values)。
