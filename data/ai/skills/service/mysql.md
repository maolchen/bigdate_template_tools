MySQL 特殊规则：
- 配置文件通常命名为 my.cnf.tmpl。
- 数据目录、日志目录优先参数化为 .Instance.Vars，再与 .Global.data_base_dir 拼接。
- 如果需要初始化库或用户，优先拆分成额外脚本模板，不要把所有逻辑塞进一个大脚本。
