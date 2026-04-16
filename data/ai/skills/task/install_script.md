用于 `install.sh.tmpl` 及少量必要的 start/stop/check 脚本。

安装脚本硬约束：
- 每个服务默认一个 `install.sh.tmpl`
- 若一个安装脚本足够完成任务，不要拆成多个脚本
- 默认按照二进制包安装理解
- 安装脚本应具备幂等性
- 结构建议：变量区 -> 辅助函数 -> 主流程
- `run_as_root` 仅用于特权操作
- 非必要不使用 root 直接执行

`run_as_root` 适用场景：
- 写 `/etc/profile`、`/etc/profile.d`
- 写 systemd unit
- `systemctl daemon-reload / enable / start`
- 修改系统目录权限
- 创建需要 root 的目录
- 安装到 root 才能写入的位置

不建议使用 root 的场景：
- 普通二进制解压
- 当前输出目录内的配置文件复制和调整
- 运行用户目录下的初始化
- 服务用户能完成的维护命令

环境变量规则：
- 如果运行用户是 root，可写 `/etc/profile` 或 `/etc/profile.d/*.sh`
- 如果运行用户是非 root，除了系统级环境变量外，还应考虑写入该用户的 shell 配置文件
- 例如 `.bash_profile`、`.bashrc`、`.profile`
- 若用户已明确指定 shell 环境文件，以用户要求为准

脚本与配置文件协作：
- 默认安装脚本和配置文件位于同一输出目录 `<Node.IP>/<service>/`
- 因此脚本中优先用相对路径操作配置文件
- 除非用户明确指定，否则不要假设配置文件在额外 `config/` 子目录

推荐内容：
- 解压安装包
- 创建目录
- 调整 owner/group
- 复制或覆盖生成后的配置文件到服务真实配置目录
- 必要的初始化数据
- 必要的开机自启动配置

避免：
- 无意义拆分为 `setup_dirs.sh`、`setup_env.sh`、`deploy_config.sh` 多脚本
- 把所有命令都包到 `sudo`
- 把目录、端口、用户、包名全部硬编码

若使用 `.Instance.Vars.xxx`、`.Global.xxx` 或新增变量，应同步建议对应配置补丁。
