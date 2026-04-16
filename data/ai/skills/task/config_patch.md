当模板变更引入或依赖变量时，返回最小可用的配置补丁建议。

目标：
- 为模板中用到的 `.Instance.Vars.*` 补齐 `serverConfig.<service>.vars`
- 为新服务补齐最基本的 `serviceTop.<service>` 草案
- 必要时补充 `global` 下新增共享变量

输出原则：
- patch 要最小化
- 只补当前模板真正依赖的配置
- 不覆盖无关已有配置
- 新服务优先增量追加

`serverConfig` 建议：
- 若模板引入了 `.Instance.Vars.xxx`，就应返回对应 `vars.xxx`
- `description` 应简洁说明用途
- 不要凭空补很多未被模板使用的 key

`serviceTop` 建议：
- 新服务需要返回最基本的 `nodes`、`description`、`id_auto_derive`
- 若无法确定节点分布，在 `warnings` 提示，不要编造不存在的节点别名

`global` 建议：
- 仅当变量适合做全局共享，才建议放到 `global`
- 例如 `software_dir`、`log_base_dir`、通用用户名/用户组、JDK 路径
- 不要把明显服务专属参数放到 `global`
