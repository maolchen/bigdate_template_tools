路径约束：
- 模板文件路径必须位于 templates/<service>/... 下。
- 模板文件必须以 .tmpl 结尾。
- 新服务可以先规划 mkdir 动作，再生成具体草稿文件。
- 目录计划和文件计划都只能落在 templates/ 下。
常用变量约束:
- 安装路径： {{ .Global.install_base_dir }}/<service>， 特殊指定路径除外。
- 安装包路径: {{ .Global.data_base_dir }}/{{ .Instance.Vars.install_subdir }} ,特殊指定除外
- 安装包名： {{ .Instance.Vars.pkg_name }},特殊指定除外
- 服务数据目录: {{.Global.data_base_dir}}/{{ .Instance.Vars.data_subdir }}
- 节点: {{ .Instance.Node.HostName }} 
- 节点IP： {{ .Instance.Node.IP }}
