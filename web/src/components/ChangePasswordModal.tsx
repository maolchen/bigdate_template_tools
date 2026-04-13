import { useState } from 'react';
import { LoaderCircle, X } from 'lucide-react';
import { changePassword } from '../api/auth';

interface ChangePasswordModalProps {
  open: boolean;
  force: boolean;
  onClose: () => void;
  onChanged: () => void;
}

export function ChangePasswordModal({ open, force, onClose, onChanged }: ChangePasswordModalProps) {
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!open) return null;

  const handleSubmit = async () => {
    if (!oldPassword || !newPassword || !confirmPassword) {
      setError('请完整填写密码');
      return;
    }
    if (newPassword !== confirmPassword) {
      setError('两次输入的新密码不一致');
      return;
    }

    setSaving(true);
    setError(null);
    try {
      await changePassword(oldPassword, newPassword);
      setOldPassword('');
      setNewPassword('');
      setConfirmPassword('');
      onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : '修改密码失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={() => { if (!force) onClose(); }}>
      <div className="modal modal-form-card" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <div>
            <h3 className="font-bold text-gray-800 text-lg m-0">修改密码</h3>
            <p className="text-sm text-gray-500 mt-1 m-0">
              {force ? '首次登录必须先修改密码后再继续使用。' : '建议定期更新密码。'}
            </p>
          </div>
          {!force && (
            <button className="modal-close" onClick={onClose} aria-label="关闭">
              <X className="w-4 h-4" />
            </button>
          )}
        </div>
        <div className="modal-body space-y-3">
          <input className="input" type="password" placeholder="当前密码" value={oldPassword} onChange={(event) => setOldPassword(event.target.value)} />
          <input className="input" type="password" placeholder="新密码（>=8位）" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} />
          <input className="input" type="password" placeholder="确认新密码" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} />
          {error && <div className="text-sm text-danger">{error}</div>}
        </div>
        <div className="modal-footer modal-footer-actions">
          {!force && <button className="btn btn-secondary" onClick={onClose}>取消</button>}
          <button className="btn btn-primary" onClick={() => void handleSubmit()} disabled={saving}>
            {saving ? <LoaderCircle className="w-4 h-4 animate-spin" /> : null}
            保存新密码
          </button>
        </div>
      </div>
    </div>
  );
}

