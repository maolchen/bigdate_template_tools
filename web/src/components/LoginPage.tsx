import { useState } from 'react';
import { LoaderCircle, LockKeyhole } from 'lucide-react';

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
    <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
      <div className="card w-full max-w-md">
        <div className="card-body">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
              <LockKeyhole className="w-5 h-5 text-primary" />
            </div>
            <div>
              <h2 className="text-xl font-semibold text-gray-800">登录配置工具</h2>
              <p className="text-sm text-gray-500">按用户隔离配置与输出目录</p>
            </div>
          </div>

          <form onSubmit={(event) => void handleSubmit(event)} className="space-y-4">
            <div>
              <label className="form-label">用户名</label>
              <input
                className="input"
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                placeholder="请输入用户名"
                autoComplete="username"
              />
            </div>
            <div>
              <label className="form-label">密码</label>
              <input
                className="input"
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder="请输入密码"
                autoComplete="current-password"
              />
            </div>

            {error && <div className="text-sm text-danger">{error}</div>}

            <button className="btn btn-primary w-full" disabled={loading || !username.trim() || !password}>
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

          <div className="text-xs text-gray-500 mt-4">
            首次启动默认管理员账号：`admin` / `Admin@123`
          </div>
        </div>
      </div>
    </div>
  );
}

