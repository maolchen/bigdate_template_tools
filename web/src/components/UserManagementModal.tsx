import { useEffect, useState } from 'react';
import { LoaderCircle, Plus, X } from 'lucide-react';
import { createUser, fetchUsers, updateUser, type UserSummary } from '../api/auth';

interface UserManagementModalProps {
  open: boolean;
  onClose: () => void;
}

export function UserManagementModal({ open, onClose }: UserManagementModalProps) {
  const [users, setUsers] = useState<UserSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [newUsername, setNewUsername] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [newRole, setNewRole] = useState<'admin' | 'user'>('user');

  const loadUsers = async () => {
    setLoading(true);
    setError(null);
    try {
      setUsers(await fetchUsers());
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载用户失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!open) return;
    void loadUsers();
  }, [open]);

  const handleCreate = async () => {
    if (!newUsername.trim() || !newPassword) {
      setError('请输入用户名和密码');
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await createUser({
        username: newUsername.trim(),
        password: newPassword,
        role: newRole,
        enabled: true,
      });
      setNewUsername('');
      setNewPassword('');
      setNewRole('user');
      await loadUsers();
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建用户失败');
    } finally {
      setSaving(false);
    }
  };

  const handleToggleEnabled = async (user: UserSummary) => {
    setSaving(true);
    setError(null);
    try {
      await updateUser(user.username, { enabled: !user.enabled });
      await loadUsers();
    } catch (err) {
      setError(err instanceof Error ? err.message : '更新用户失败');
    } finally {
      setSaving(false);
    }
  };

  const handleToggleRole = async (user: UserSummary) => {
    setSaving(true);
    setError(null);
    try {
      await updateUser(user.username, { role: user.role === 'admin' ? 'user' : 'admin' });
      await loadUsers();
    } catch (err) {
      setError(err instanceof Error ? err.message : '更新用户失败');
    } finally {
      setSaving(false);
    }
  };

  const handleResetPassword = async (user: UserSummary) => {
    if (!confirm(`确认将用户 ${user.username} 的密码重置为 Admin@123 吗？`)) return;
    setSaving(true);
    setError(null);
    try {
      await updateUser(user.username, { resetPassword: 'Admin@123' });
      await loadUsers();
    } catch (err) {
      setError(err instanceof Error ? err.message : '重置密码失败');
    } finally {
      setSaving(false);
    }
  };

  if (!open) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal modal-form-card" style={{ maxWidth: 980 }} onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <div>
            <h3 className="font-bold text-gray-800 text-lg m-0">用户管理</h3>
            <p className="text-sm text-gray-500 mt-1 m-0">仅管理员可见。支持创建用户、启停账号、切换角色和重置密码。</p>
          </div>
          <button className="modal-close" onClick={onClose} aria-label="关闭">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="modal-body space-y-4">
          <div className="card">
            <div className="card-body">
              <div className="grid grid-cols-3 gap-3">
                <input className="input" placeholder="用户名" value={newUsername} onChange={(event) => setNewUsername(event.target.value)} />
                <input className="input" placeholder="初始密码（>=8位）" type="password" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} />
                <select className="select" value={newRole} onChange={(event) => setNewRole(event.target.value as 'admin' | 'user')}>
                  <option value="user">普通用户</option>
                  <option value="admin">管理员</option>
                </select>
              </div>
              <div className="mt-3">
                <button className="btn btn-primary" onClick={() => void handleCreate()} disabled={saving}>
                  <Plus className="w-4 h-4" />
                  新建用户
                </button>
              </div>
            </div>
          </div>

          {error && <div className="text-sm text-danger">{error}</div>}

          <div className="card">
            <div className="table-container">
              <table className="table">
                <thead>
                  <tr>
                    <th>用户名</th>
                    <th>角色</th>
                    <th>状态</th>
                    <th>改密要求</th>
                    <th style={{ width: 320 }}>操作</th>
                  </tr>
                </thead>
                <tbody>
                  {users.map((user) => (
                    <tr key={user.username}>
                      <td>{user.username}</td>
                      <td>{user.role === 'admin' ? '管理员' : '普通用户'}</td>
                      <td>{user.enabled ? '启用' : '停用'}</td>
                      <td>{user.mustChangePassword ? '是' : '否'}</td>
                      <td>
                        <div className="flex items-center gap-2">
                          <button className="btn btn-sm btn-secondary" onClick={() => void handleToggleRole(user)} disabled={saving}>
                            切换角色
                          </button>
                          <button className="btn btn-sm btn-secondary" onClick={() => void handleToggleEnabled(user)} disabled={saving}>
                            {user.enabled ? '停用' : '启用'}
                          </button>
                          <button className="btn btn-sm btn-secondary" onClick={() => void handleResetPassword(user)} disabled={saving}>
                            重置密码
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {loading && (
              <div className="card-body text-sm text-gray-500 flex items-center gap-2">
                <LoaderCircle className="w-4 h-4 animate-spin" />
                加载中...
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

