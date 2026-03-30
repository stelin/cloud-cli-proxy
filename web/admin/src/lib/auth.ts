const CURRENT_TOKEN_KEY = "admin_token";
const SESSIONS_KEY = "auth_sessions";
const CURRENT_SESSION_ID_KEY = "auth_current_session_id";
const AUTH_EVENT = "auth:sessions-changed";

export interface AuthSession {
  id: string;
  /** 用户短 ID（curl 入口等）；会话列表展示优先使用 username */
  shortId: string;
  /** 用户名，网页登录账号；旧会话可能为空 */
  username?: string;
  token: string;
  role: string;
  userId: string;
  subject: string;
  exp: number;
  lastUsedAt: number;
}

export interface TokenPayload {
  user_id: string;
  role: string;
  sub: string;
  exp: number;
}

export function parseTokenPayloadFromToken(token: string): TokenPayload | null {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    const payload = JSON.parse(atob(parts[1]));
    return {
      user_id: payload.user_id || "",
      role: payload.role || "",
      sub: payload.sub || "",
      exp: payload.exp || 0,
    };
  } catch {
    return null;
  }
}

function emitAuthChange(): void {
  window.dispatchEvent(new Event(AUTH_EVENT));
}

function readSessions(): AuthSession[] {
  try {
    const raw = localStorage.getItem(SESSIONS_KEY);
    if (!raw) return [];
    const sessions = JSON.parse(raw) as AuthSession[];
    return Array.isArray(sessions) ? sessions : [];
  } catch {
    return [];
  }
}

function writeSessions(sessions: AuthSession[]): void {
  localStorage.setItem(SESSIONS_KEY, JSON.stringify(sessions));
}

function setCurrentSessionId(sessionId: string | null): void {
  if (sessionId) {
    localStorage.setItem(CURRENT_SESSION_ID_KEY, sessionId);
  } else {
    localStorage.removeItem(CURRENT_SESSION_ID_KEY);
  }
}

function syncCurrentToken(session: AuthSession | null): void {
  if (session) {
    localStorage.setItem(CURRENT_TOKEN_KEY, session.token);
  } else {
    localStorage.removeItem(CURRENT_TOKEN_KEY);
  }
}

function getRedirectPathForRole(role: string | null): string {
  return role === "admin" ? "/" : "/portal";
}

export function getSessions(): AuthSession[] {
  return readSessions().sort((a, b) => b.lastUsedAt - a.lastUsedAt);
}

export function getCurrentSession(): AuthSession | null {
  const sessions = readSessions();
  const currentId = localStorage.getItem(CURRENT_SESSION_ID_KEY);
  const current = sessions.find((session) => session.id === currentId) ?? null;
  if (current) {
    syncCurrentToken(current);
    return current;
  }
  const fallback = sessions[0] ?? null;
  if (fallback) {
    setCurrentSessionId(fallback.id);
    syncCurrentToken(fallback);
  } else {
    syncCurrentToken(null);
  }
  return fallback;
}

export function isAuthenticated(): boolean {
  return !!getCurrentSession();
}

export function getToken(): string | null {
  return getCurrentSession()?.token ?? null;
}

export function saveSession(
  shortId: string,
  token: string,
  username?: string,
): AuthSession | null {
  const payload = parseTokenPayloadFromToken(token);
  if (!payload) return null;

  const session: AuthSession = {
    id: `${payload.role}:${payload.user_id}:${shortId}`,
    shortId,
    ...(username ? { username } : {}),
    token,
    role: payload.role,
    userId: payload.user_id,
    subject: payload.sub,
    exp: payload.exp,
    lastUsedAt: Date.now(),
  };

  const sessions = readSessions().filter((item) => item.id !== session.id);
  sessions.unshift(session);
  writeSessions(sessions);
  setCurrentSessionId(session.id);
  syncCurrentToken(session);
  emitAuthChange();
  return session;
}

export function switchSession(sessionId: string): AuthSession | null {
  const sessions = readSessions();
  const target = sessions.find((session) => session.id === sessionId) ?? null;
  if (!target) return null;
  target.lastUsedAt = Date.now();
  writeSessions(
    sessions
      .map((session) => (session.id === sessionId ? target : session))
      .sort((a, b) => b.lastUsedAt - a.lastUsedAt),
  );
  setCurrentSessionId(target.id);
  syncCurrentToken(target);
  emitAuthChange();
  return target;
}

export function clearCurrentSession(): void {
  const current = getCurrentSession();
  if (!current) {
    syncCurrentToken(null);
    setCurrentSessionId(null);
    emitAuthChange();
    return;
  }
  const remaining = readSessions().filter((session) => session.id !== current.id);
  writeSessions(remaining);
  const next = remaining[0] ?? null;
  setCurrentSessionId(next?.id ?? null);
  syncCurrentToken(next);
  emitAuthChange();
}

export function clearAllSessions(): void {
  writeSessions([]);
  setCurrentSessionId(null);
  syncCurrentToken(null);
  emitAuthChange();
}

export function logout(): void {
  clearCurrentSession();
  window.location.href = "/login";
}

export function parseTokenPayload(): TokenPayload | null {
  const token = getToken();
  if (!token) return null;
  return parseTokenPayloadFromToken(token);
}

export function getRole(): string | null {
  return getCurrentSession()?.role ?? null;
}

export function isAdmin(): boolean {
  return getRole() === "admin";
}

export function redirectToCurrentSessionHome(): void {
  window.location.href = getRedirectPathForRole(getRole());
}

export function redirectToSessionHome(sessionId: string): void {
  const session = switchSession(sessionId);
  window.location.href = getRedirectPathForRole(session?.role ?? null);
}

export function removeSession(sessionId: string): void {
  const currentId = getCurrentSession()?.id ?? null;
  const remaining = readSessions().filter((session) => session.id !== sessionId);
  writeSessions(remaining);
  if (currentId === sessionId) {
    const next = remaining[0] ?? null;
    setCurrentSessionId(next?.id ?? null);
    syncCurrentToken(next);
  }
  emitAuthChange();
}

export function subscribeAuthChanges(callback: () => void): () => void {
  const handler = () => callback();
  window.addEventListener(AUTH_EVENT, handler);
  window.addEventListener("storage", handler);
  return () => {
    window.removeEventListener(AUTH_EVENT, handler);
    window.removeEventListener("storage", handler);
  };
}
