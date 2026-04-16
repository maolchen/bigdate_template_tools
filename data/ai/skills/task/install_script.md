用于 install/start/stop/check 脚本 (for install/start/stop/check scripts)：
- 结构清晰：变量区、辅助函数、主流程 (variables/helpers/main flow)
- `run_as_root` 仅用于特权操作
- 关键操作尽量幂等 (keep operations idempotent when possible)
- 若使用 `.Instance.Vars.xxx`，应同步建议 `serverConfig.vars.xxx`
