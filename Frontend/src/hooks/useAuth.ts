import { useState } from "react";

export const TOKEN_KEY = "grace_token";
const USERNAME_KEY = "grace_username";

export function useAuth() {
  const [token, setToken] = useState<string>(() => localStorage.getItem(TOKEN_KEY) || "");
  const [username, setUsername] = useState<string>(
    () => localStorage.getItem(USERNAME_KEY) || "管理员"
  );

  const login = (t: string, user?: string) => {
    localStorage.setItem(TOKEN_KEY, t);
    if (user) localStorage.setItem(USERNAME_KEY, user);
    setToken(t);
    if (user) setUsername(user);
  };

  const logout = () => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USERNAME_KEY);
    setToken("");
    setUsername("管理员");
  };

  /**
   * Returns the redirect target from the ?from= query param.
   * Used by LoginPage to navigate back after successful login.
   */
  const getRedirectTarget = (): string => {
    const sp = new URLSearchParams(window.location.search);
    const from = sp.get("from");
    if (from) {
      try {
        const decoded = decodeURIComponent(from);
        // Only allow relative paths to prevent open redirect
        if (decoded.startsWith("/") && !decoded.startsWith("//")) return decoded;
      } catch {
        // ignore malformed
      }
    }
    return "/dashboard";
  };

  return { token, username, isAuthenticated: !!token, login, logout, getRedirectTarget };
}
