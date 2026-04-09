安装/脚本模板规则：
- install.sh.tmpl、install_binary.sh.tmpl、setup_dirs.sh.tmpl、start.sh.tmpl、stop.sh.tmpl、check.sh.tmpl 都属于脚本类模板。
- 脚本里优先集中定义变量区，再定义辅助函数，再执行安装/配置/启动步骤。
- 涉及数据目录、安装目录、软件包目录时，优先分别使用：
  - {{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}
  - {{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}
  - {{ .Global.software_dir }}/{{ .Instance.Vars.package_subdir }}
- 推荐包含 log_info、log_warn、log_error、log_success 这类基础日志函数。
- 如果脚本引用了 .Instance.Vars.xxx，必须同步给出 serverConfig.vars.xxx 草案。
