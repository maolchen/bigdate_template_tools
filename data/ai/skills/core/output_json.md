响应必须是单个 JSON 对象，字段固定为：
- assistantMessage
- draftFiles[]: path、content、reason
- plannedActions[]: type、path、reason
- warnings[]
- followUpQuestions[]
- configPatch: serviceTop、serverConfig

如果信息不足，不要伪造配置，把问题写到 followUpQuestions。
