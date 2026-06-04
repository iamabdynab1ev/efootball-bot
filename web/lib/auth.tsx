"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { User } from "@/lib/api";

interface AuthState {
  user: User | null;
  token: string | null;
  loading: boolean;
  login: (idToken: string) => Promise<void>;
  adminLogin: (username: string, password: string) => Promise<void>;
  devLogin: (userId: number) => Promise<void>;
  logout: () => void;
  refreshUser: () => Promise<void>;
}

const TOKEN_KEY = "efootball_jwt";

const AuthContext = createContext<AuthState>({
  user: null,
  token: null,
  loading: true,
  login: async () => {},
  adminLogin: async () => {},
  devLogin: async () => {},
  logout: () => {},
  refreshUser: async () => {},
});

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const qc = useQueryClient();
  const [token, setToken] = useState<string | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const loadUser = useCallback(async (jwt: string) => {
    const res = await fetch("/api/me", { headers: { Authorization: `Bearer ${jwt}` } });
    if (!res.ok) throw new Error("Сессия истекла. Войдите снова.");
    const nextUser = (await res.json()) as User;
    setUser(nextUser);
  }, []);

  useEffect(() => {
    const stored = localStorage.getItem(TOKEN_KEY);
    if (!stored) {
      setLoading(false);
      return;
    }

    setToken(stored);
    loadUser(stored)
      .catch(() => {
        localStorage.removeItem(TOKEN_KEY);
        setToken(null);
        setUser(null);
      })
      .finally(() => setLoading(false));
  }, [loadUser]);

  const login = useCallback(async (idToken: string) => {
    const res = await fetch("/auth/google", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ id_token: idToken }),
    });
    if (!res.ok) throw new Error("Не удалось войти через Google. Попробуйте ещё раз.");
    const data = (await res.json()) as { token: string; user: User };
    qc.clear(); // сбрасываем старый кэш перед установкой нового пользователя
    localStorage.setItem(TOKEN_KEY, data.token);
    setToken(data.token);
    setUser(data.user);
  }, [qc]);

  const adminLogin = useCallback(async (username: string, password: string) => {
    const res = await fetch("/auth/admin-login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    if (!res.ok) throw new Error("Неверный логин или пароль.");
    const data = (await res.json()) as { token: string; user: User };
    qc.clear();
    localStorage.setItem(TOKEN_KEY, data.token);
    setToken(data.token);
    setUser(data.user);
  }, [qc]);

  const devLogin = useCallback(async (userId: number) => {
    const res = await fetch("/auth/dev-login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ user_id: userId }),
    });
    if (!res.ok) throw new Error("Dev login не работает. Проверьте сервер.");
    const data = (await res.json()) as { token: string; user: User };
    qc.clear(); // каждый dev-логин начинает с чистого кэша
    localStorage.setItem(TOKEN_KEY, data.token);
    setToken(data.token);
    setUser(data.user);
  }, [qc]);

  const refreshUser = useCallback(async () => {
    const stored = localStorage.getItem(TOKEN_KEY);
    if (!stored) return;
    await loadUser(stored);
  }, [loadUser]);

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY);
    setToken(null);
    setUser(null);
    qc.clear(); // очищаем весь кэш при выходе
  }, [qc]);

  const value = useMemo(
    () => ({ user, token, loading, login, adminLogin, devLogin, logout, refreshUser }),
    [user, token, loading, login, adminLogin, devLogin, logout, refreshUser]
  );

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => useContext(AuthContext);
