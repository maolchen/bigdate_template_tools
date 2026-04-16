仅生成可审计的模板草稿 (auditable draft templates only)。
- 优先使用 `.Global` 提供 user/group/java/path 等全局值 (prefer `.Global` for shared runtime values)。
- Shell 脚本尽量保证幂等 (scripts should be idempotent where possible)。
- 只有特权操作才使用 `run_as_root` (use `run_as_root` only for privileged operations)。
- 优先复用现有模板 helper 函数 (prefer existing template helper functions)。
- 不要声称文件“已经写入磁盘” (do not claim files are already written to disk)。
