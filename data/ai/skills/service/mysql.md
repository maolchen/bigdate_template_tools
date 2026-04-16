MySQL 模板补充约束 (MySQL-specific guidance)：
- 目录、端口等参数应保持可配置 (parameterized)
- 可来自 `.Global` / `.Instance.Vars` 的值不要硬编码
- 涉及数据目录时保持幂等创建逻辑
