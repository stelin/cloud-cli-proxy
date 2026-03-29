export function isAuthenticated(): boolean {
  return !!localStorage.getItem("admin_token");
}

export function getToken(): string | null {
  return localStorage.getItem("admin_token");
}

export function setToken(token: string): void {
  localStorage.setItem("admin_token", token);
}

export function clearToken(): void {
  localStorage.removeItem("admin_token");
}

export function logout(): void {
  clearToken();
  window.location.href = "/login";
}

export function parseTokenPayload(): {
  user_id: string;
  role: string;
  sub: string;
  exp: number;
} | null {
  const token = getToken();
  if (!token) return null;
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

export function getRole(): string | null {
  return parseTokenPayload()?.role ?? null;
}

export function isAdmin(): boolean {
  return getRole() === "admin";
}
