配置同步规则：
- 新服务模板通常需要同时给出 configPatch.serviceTop 和 configPatch.serverConfig。
- 如果模板引用了 .Instance.Vars.xxx，就必须补对应的 serverConfig.vars.xxx。
- 不要把与模板无关的变量塞进 vars。
- 已有服务如果只是补模板，优先补齐缺失项，不要无理由覆盖现有配置。
