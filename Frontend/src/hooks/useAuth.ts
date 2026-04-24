import { useState } from "react";

const TOKEN_KEY = "grace_token";

export function useAuth() {
  const [token, setToken] = useState<string>(() => localStorage.getItem(TOKEN_KEY) || "");

  const login = (t: string) => {
    localStorage.setItem(TOKEN_KEY, t);
    setToken(t);
  };

  const logout = () => {
    localStorage.removeItem(TOKEN_KEY);
    setToken("");
  };

  return { token, isAuthenticated: !!token, login, logout };
}
