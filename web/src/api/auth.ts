const API_BASE = '/api';

export interface AuthUser {
  username: string;
  role: 'admin' | 'user';
  mustChangePassword: boolean;
}

export interface LoginResponse {
  success: boolean;
  user: AuthUser;
  sync: {
    addedServices: string[];
    serviceTopAdded: number;
    serverConfigAdded: number;
  };
}

export interface MeResponse {
  user: AuthUser;
  activeTemplateId: string;
}

export interface UserSummary {
  username: string;
  role: 'admin' | 'user';
  enabled: boolean;
  mustChangePassword: boolean;
  createdAt: string;
  updatedAt: string;
}

async function parseError(response: Response, fallback: string): Promise<Error> {
  const text = (await response.text()).trim();
  return new Error(text || fallback);
}

export async function login(username: string, password: string): Promise<LoginResponse> {
  const response = await fetch(`${API_BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });
  if (!response.ok) throw await parseError(response, '登录失败');
  return response.json();
}

export async function logout(): Promise<void> {
  const response = await fetch(`${API_BASE}/auth/logout`, {
    method: 'POST',
  });
  if (!response.ok) throw await parseError(response, '退出登录失败');
}

export async function fetchMe(): Promise<MeResponse> {
  const response = await fetch(`${API_BASE}/auth/me`);
  if (!response.ok) throw await parseError(response, '获取用户信息失败');
  return response.json();
}

export async function changePassword(oldPassword: string, newPassword: string): Promise<{ success: boolean; message: string }> {
  const response = await fetch(`${API_BASE}/auth/change-password`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ oldPassword, newPassword }),
  });
  if (!response.ok) throw await parseError(response, '修改密码失败');
  return response.json();
}

export async function fetchUsers(): Promise<UserSummary[]> {
  const response = await fetch(`${API_BASE}/users`);
  if (!response.ok) throw await parseError(response, '获取用户列表失败');
  return response.json();
}

export async function createUser(payload: {
  username: string;
  password: string;
  role: 'admin' | 'user';
  enabled?: boolean;
}): Promise<UserSummary> {
  const response = await fetch(`${API_BASE}/users`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw await parseError(response, '创建用户失败');
  return response.json();
}

export async function updateUser(username: string, payload: {
  role?: 'admin' | 'user';
  enabled?: boolean;
  resetPassword?: string;
  mustChangePassword?: boolean;
}): Promise<UserSummary> {
  const response = await fetch(`${API_BASE}/users/${encodeURIComponent(username)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw await parseError(response, '更新用户失败');
  return response.json();
}

