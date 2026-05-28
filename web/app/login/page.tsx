"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import { GoogleLogin } from "@react-oauth/google";
import { Eye, EyeOff, KeyRound, LogIn, Shield, ShieldCheck, Trophy, Users, Zap } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/utils";

type Tab = "player" | "admin";

export default function LoginPage() {
  const router = useRouter();
  const { user, login, adminLogin } = useAuth();
  const [tab, setTab] = useState<Tab>("player");
  const [error, setError] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPw, setShowPw] = useState(false);
  const [loading, setLoading] = useState(false);
  const googleConfigured = Boolean(process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID);

  useEffect(() => {
    if (user) router.replace("/");
  }, [router, user]);

  const features = [
    { icon: Trophy,      text: "Участвуйте в лигах" },
    { icon: Zap,         text: "Следите за рейтингом ELO" },
    { icon: Users,       text: "Управляйте своими матчами" },
    { icon: ShieldCheck, text: "Уведомления в Telegram-боте" },
  ];

  async function handleAdminSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await adminLogin(username.trim(), password);
      router.replace("/admin");
    } catch {
      setError("Неверный логин или пароль.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-[calc(100vh-200px)] flex items-center justify-center py-8">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="w-full max-w-sm"
      >
        <div className="flex flex-col items-center mb-8">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-yellow-500 text-zinc-950 mb-4 shadow-lg shadow-yellow-500/20">
            <Trophy size={32} strokeWidth={2.5} />
          </div>
          <h1 className="text-2xl font-black text-zinc-100">eFootball Web</h1>
          <p className="text-sm text-zinc-500 mt-1">Лиги · Матчи · Рейтинг</p>
        </div>

        {/* Tabs */}
        <div className="flex gap-1 mb-3 rounded-xl border border-zinc-800 bg-zinc-900 p-1">
          {([
            { key: "player" as Tab, label: "Игрок",         icon: Users  },
            { key: "admin"  as Tab, label: "Администратор", icon: Shield },
          ] as const).map(({ key, label, icon: Icon }) => (
            <button
              key={key}
              onClick={() => { setTab(key); setError(""); }}
              className={cn(
                "flex flex-1 items-center justify-center gap-2 rounded-lg py-2 text-sm font-semibold transition-colors",
                tab === key
                  ? "bg-yellow-500 text-zinc-950"
                  : "text-zinc-400 hover:text-zinc-200"
              )}
            >
              <Icon size={14} />
              {label}
            </button>
          ))}
        </div>

        <div className="rounded-2xl border border-zinc-800 bg-zinc-900 p-6 space-y-5">
          {tab === "player" && (
            <>
              <div>
                <h2 className="text-base font-bold text-zinc-100">Войти в кабинет</h2>
                <p className="text-sm text-zinc-500 mt-1">Один аккаунт для веб-кабинета и Telegram-бота.</p>
              </div>

              <div className="space-y-2">
                {features.map((f) => (
                  <div key={f.text} className="flex items-center gap-2.5 text-sm text-zinc-400">
                    <f.icon size={15} className="text-yellow-500 flex-shrink-0" />
                    {f.text}
                  </div>
                ))}
              </div>

              <div className="pt-1">
                {googleConfigured ? (
                  <GoogleLogin
                    onSuccess={async (cred) => {
                      setError("");
                      try {
                        if (!cred.credential) throw new Error("No credential");
                        await login(cred.credential);
                        router.replace("/");
                      } catch {
                        setError("Не удалось войти. Проверьте Google OAuth и попробуйте снова.");
                      }
                    }}
                    onError={() => setError("Google OAuth вернул ошибку.")}
                    theme="filled_black"
                    shape="rectangular"
                    size="large"
                    text="signin_with"
                    locale="ru"
                  />
                ) : (
                  <div className="flex items-start gap-2.5 rounded-lg border border-amber-500/20 bg-amber-500/5 px-3 py-2.5">
                    <ShieldCheck size={15} className="text-amber-400 flex-shrink-0 mt-0.5" />
                    <p className="text-xs text-amber-300">
                      Укажите NEXT_PUBLIC_GOOGLE_CLIENT_ID в web/.env.local, чтобы включить вход.
                    </p>
                  </div>
                )}
              </div>
            </>
          )}

          {tab === "admin" && (
            <>
              <div className="flex items-center gap-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-yellow-500/10">
                  <KeyRound size={15} className="text-yellow-400" />
                </div>
                <div>
                  <h2 className="text-base font-bold text-zinc-100">Вход для администратора</h2>
                  <p className="text-xs text-zinc-500">Доступ только для уполномоченных лиц</p>
                </div>
              </div>

              <form onSubmit={handleAdminSubmit} className="space-y-3">
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-zinc-400">Логин</label>
                  <input
                    type="text"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    placeholder="Имя пользователя"
                    autoComplete="username"
                    required
                    className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 outline-none focus:border-yellow-500 focus:ring-1 focus:ring-yellow-500/30 transition-colors"
                  />
                </div>

                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-zinc-400">Пароль</label>
                  <div className="relative">
                    <input
                      type={showPw ? "text" : "password"}
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      placeholder="••••••••"
                      autoComplete="current-password"
                      required
                      className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 pr-10 text-sm text-zinc-100 placeholder-zinc-600 outline-none focus:border-yellow-500 focus:ring-1 focus:ring-yellow-500/30 transition-colors"
                    />
                    <button
                      type="button"
                      onClick={() => setShowPw((v) => !v)}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-500 hover:text-zinc-300 transition-colors"
                    >
                      {showPw ? <EyeOff size={15} /> : <Eye size={15} />}
                    </button>
                  </div>
                </div>

                <button
                  type="submit"
                  disabled={loading}
                  className="flex w-full items-center justify-center gap-2 rounded-lg bg-yellow-500 py-2.5 text-sm font-bold text-zinc-950 transition-opacity hover:opacity-90 disabled:opacity-50"
                >
                  <LogIn size={15} />
                  {loading ? "Проверяем..." : "Войти"}
                </button>
              </form>
            </>
          )}

          {error && (
            <div className="rounded-lg border border-red-500/20 bg-red-500/5 px-3 py-2 text-xs text-red-400">
              {error}
            </div>
          )}
        </div>
      </motion.div>
    </div>
  );
}
