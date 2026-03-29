import { ApiError } from "./api";

const PORTAL_API_BASE = "/v1/user";

export async function portalApiFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const token = localStorage.getItem("admin_token");
  const res = await fetch(`${PORTAL_API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init?.headers,
    },
  });

  if (res.status === 401) {
    localStorage.removeItem("admin_token");
    window.location.href = "/login";
    throw new ApiError(401, "Unauthorized");
  }

  if (!res.ok) {
    const body = await res.text();
    throw new ApiError(res.status, body);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return res.json();
}
