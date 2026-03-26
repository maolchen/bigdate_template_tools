# 全局变量复用规范

## 全局变量定义 (config.yaml -> global)

| 变量名 | 值 | 用途 |
|--------|-----|------|
| `install_base_dir` | `/data/localization` | 安装目录基础路径 |
| `data_base_dir` | `/data` | 数据目录基础路径 |
| `temp_dir` | `/data/tmp_install_dir` | 临时目录 |
| `software_dir` | `/data/softwares` | 软件包目录 |
| `java_home` | `/data/jdk/` | JAVA_HOME 路径 |
| `user` | `bigdata` | 运行用户 |
| `group` | `bigdata` | 运行用户组 |
| `timezone` | `Asia/Shanghai` | 时区 |

## 服务配置规范

### 1. 目录配置规范

**数据目录**：使用 `xxx_data_subdir` 后缀，模板中拼接
```yaml
# config.yaml 中
vars:
  data_subdir: "service_name/data"

# 模板中
{{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}
```

**安装目录**：使用 `install_subdir` 后缀，模板中拼接
```yaml
# config.yaml 中
vars:
  install_subdir: "service_name"

# 模板中
{{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}
```

### 2. 用户/组配置规范

**删除服务级别的 run_user/run_group**，模板中直接使用全局变量：
```yaml
# 模板中
RUN_USER="{{ .Global.user }}"
RUN_GROUP="{{ .Global.group }}"
```

### 3. JAVA_HOME 配置规范

**删除服务级别的 java_home**，模板中直接使用全局变量：
```yaml
# 模板中
JAVA_HOME="{{ .Global.java_home }}"
```

### 4. 环境变量配置规范 
环境变量的配置，根据{{ .Global.user }}判断是否为root用户，如果是root用户，则配置到/etc/profile，如果是非root用户，则配置到~/.bash_profile 和/etc/profile 

### 5. 脚本编写规范  
所有的shell脚本，默认使用{{ .Global.user }}这个用户进行执行，如果该用户为非root用户，所有需要进行root用户的操作，均在脚本中使用sudo 提权。
脚本需要尽可能的具有幂等性-----重要  

### 6. 服务部署和启动规范  
所有的服务非必要不使用root用户启动，使用{{ .Global.user }}启动，systemctl 管理的或者有特殊说明的除外，如果有systemctl 启动的或者类似docker用户执行的，或者强制只能安装到/usr/local目录中的服务，均需要配置{{ .Global.user }}能访问和运维的权限，例如chown -R {{ .Global.user }}:{{ .Global.group }} xxx/



## 模板修改规范

所有模板文件需要：

1. **路径拼接**：
   ```bash
   # 旧写法
   DATA_DIR="{{ .Instance.Vars.data_dir }}"
   
   # 新写法
   DATA_DIR="{{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}"
   ```

2. **用户/组**：
   ```bash
   # 旧写法
   RUN_USER="{{ .Instance.Vars.run_user }}"
   
   # 新写法
   RUN_USER="{{ .Global.user }}"
   ```

3. **JAVA_HOME**：
   ```bash
   # 旧写法
   JAVA_HOME="{{ .Instance.Vars.java_home }}"
   
   # 新写法
   JAVA_HOME="{{ .Global.java_home }}"
   ```

## 注意事项

1. 特殊情况：某些服务（如 SSDB）安装目录固定，不可修改
2. 多实例服务：每个实例的 data_subdir 应该不同
3. package_subdir 保持相对路径，与 software_dir 拼接
