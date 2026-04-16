import { useState } from 'react';
import { LoaderCircle, ShieldCheck, Sparkles, Workflow } from 'lucide-react';

interface LoginPageProps {
  loading: boolean;
  error: string | null;
  onLogin: (username: string, password: string) => Promise<void>;
}

export function LoginPage({ loading, error, onLogin }: LoginPageProps) {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    await onLogin(username.trim(), password);
  };

  return (
    <div className="login-stage">
      <section className="login-brand-wrap">
        <div className="login-brand-bg" aria-hidden="true" />
        <div className="login-brand">
          <div className="login-brand-badge">
            <Sparkles className="w-4 h-4" />
            <span>{'离线交付配置中心'}</span>
          </div>
          <h1 className="login-brand-title">大数据平台脚本生成工具</h1>
          <p className="login-brand-subtitle">
            基于 Go Template 的可视化配置与模板生成平台，面向内网无 SSH 互通场景。
          </p>
          <div className="login-brand-grid">
            <div className="login-brand-card">
              <Workflow className="w-4 h-4" />
              <div>
                <h3>四层配置分离</h3>
                <p>全局、节点、拓扑、服务配置统一管理</p>
              </div>
            </div>
            <div className="login-brand-card">
              <ShieldCheck className="w-4 h-4" />
              <div>
                <h3>角色权限隔离</h3>
                <p>管理员单会话锁定，普通用户按空间独立操作</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="login-panel-wrap">
        <div className="login-panel">
          <div className="login-panel-body">
            <div className="login-panel-head">
              <h2>欢迎使用</h2>
            </div>

            <form onSubmit={(event) => void handleSubmit(event)} className="login-form">
              <div className="form-group">
                <label className="form-label">用户名</label>
                <input
                  className="input login-input"
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  placeholder="请输入用户名"
                  autoComplete="username"
                />
              </div>
              <div className="form-group">
                <label className="form-label">密码</label>
                <input
                  className="input login-input"
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="请输入密码"
                  autoComplete="current-password"
                />
              </div>

              {error && <div className="login-error">{error}</div>}

              <button className="btn btn-primary w-full login-submit" disabled={loading || !username.trim() || !password}>
                {loading ? (
                  <>
                    <LoaderCircle className="w-4 h-4 animate-spin" />
                    登录中...
                  </>
                ) : (
                  '登录'
                )}
              </button>
            </form>

            <div className="login-footnote">
              首次启动默认管理员账号：<code>admin</code> / <code>Admin@123</code>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
