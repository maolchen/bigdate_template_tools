Kerberos 模板补充约束 (Kerberos-specific guidance)：
- realm 与 hostname 相关值要清晰、可审计 (explicit and auditable)
- 未给定上下文时不要编造 principal/realm 值
- 涉及域名映射时优先按现有配置变量输出
