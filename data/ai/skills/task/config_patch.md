当模板变更引入或依赖变量时 (when template changes introduce/rely on vars)：
- 返回匹配的 `configPatch.serverConfig` 建议
- patch 要最小化，不覆盖无关配置 (keep patch minimal)
- 新服务优先增量追加 (prefer additive changes)
