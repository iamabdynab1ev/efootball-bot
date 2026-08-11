"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { m } from "framer-motion";
import { GoogleLogin, GoogleOAuthProvider } from "@react-oauth/google";
import { Eye, EyeOff, KeyRound, LogIn, Shield, ShieldCheck, Trophy, Users, Zap, FlaskConical } from "lucide-react";
import { BrandLogo } from "@/components/BrandLogo";
import { useAuth } from "@/lib/auth";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

type Tab = "player" | "admin" | "dev";

const isDev = process.env.NEXT_PUBLIC_DEV_LOGIN === "true";

/**
 * GoogleOAuthProvider живёт только здесь, а не в корневых providers:
 * GSI-скрипт Google (~100KB + сторонние cookie) грузится исключительно
 * на странице логина, не замедляя остальные страницы.
 */
export default function LoginPage() {
  return (
    <GoogleOAuthProvider clientId={process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID || ""}>
      <LoginContent />
    </GoogleOAuthProvider>
  );
}

function LoginContent() {
  const router = useRouter();
  const { user, login, adminLogin, devLogin } = useAuth();
  const { t } = useLang();
  const [tab, setTab] = useState<Tab>(isDev ? "dev" : "player");
  const [error, setError] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPw, setShowPw] = useState(false);
  const [loading, setLoading] = useState(false);
  const [devUsers, setDevUsers] = useState<{ id: number; display_name: string; rating: number; rank: string }[]>([]);
  const [devUsersLoading, setDevUsersLoading] = useState(true);
  const googleConfigured = Boolean(process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID);

  useEffect(() => {
    if (!isDev) return;
    setDevUsersLoading(true);
    fetch("/auth/dev-users")
      .then(r => r.json())
      .then(data => { setDevUsers(data); setDevUsersLoading(false); })
      .catch(() => setDevUsersLoading(false));
  }, []);

  useEffect(() => {
    if (user) router.replace("/");
  }, [router, user]);

  const features = [
    { icon: Trophy,      text: t("auth.feature1") },
    { icon: Zap,         text: t("auth.feature2") },
    { icon: Users,       text: t("auth.feature3") },
    { icon: ShieldCheck, text: t("auth.feature4") },
  ];

  async function handleAdminSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await adminLogin(username.trim(), password);
      router.replace("/admin");
    } catch {
      setError(t("auth.errorCreds"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-[calc(100vh-200px)] flex items-center justify-center py-8">
      <m.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="w-full max-w-sm"
      >
        <div className="flex flex-col items-center mb-8">
          <BrandLogo size={88} className="mb-4" />
          <h1 className="font-display text-2xl font-black text-zinc-100">eFoot<span className="text-gradient-brand">League</span></h1>
          <p className="text-sm text-zinc-500 mt-1">{t("nav.leagues")} · {t("leagueDetail.schedule")} · {t("players.title")}</p>
        </div>

        {/* Tabs */}
        <div className="flex gap-1 mb-3 rounded-xl card-premium p-1">
          {([
            { key: "player" as Tab, label: t("auth.playerTab"), icon: Users,         show: true },
            { key: "admin"  as Tab, label: t("auth.adminTab"),  icon: Shield,        show: true },
            { key: "dev"    as Tab, label: "Dev",               icon: FlaskConical,  show: isDev },
          ] as const).filter(x => x.show).map(({ key, label, icon: Icon }) => (
            <button
              key={key}
              role="tab"
              aria-selected={tab === key}
              onClick={() => { setTab(key); setError(""); }}
              className={cn(
                "flex flex-1 items-center justify-center gap-2 rounded-lg py-2 text-sm font-semibold transition-colors",
                tab === key ? "bg-yellow-400 text-zinc-950" : "text-zinc-400 hover:text-zinc-200"
              )}
            >
              <Icon size={14} aria-hidden="true" />
              {label}
            </button>
          ))}
        </div>

        <div className="rounded-2xl card-premium p-6 space-y-5">
          {tab === "player" && (
            <>
              <div>
                <h2 className="text-base font-bold text-zinc-100">{t("auth.loginToAccount")}</h2>
                <p className="text-sm text-zinc-500 mt-1">{t("auth.loginDesc")}</p>
              </div>

              <div className="space-y-2">
                {features.map((f) => (
                  <div key={f.text} className="flex items-center gap-2.5 text-sm text-zinc-400">
                    <f.icon size={15} className="text-yellow-400 flex-shrink-0" />
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
                        setError(t("auth.errorGoogle"));
                      }
                    }}
                    onError={() => setError(t("auth.errorGoogleOAuth"))}
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
                      {t("auth.googleClientIdMissing")}
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
                  <h2 className="text-base font-bold text-zinc-100">{t("auth.adminLoginTitle")}</h2>
                  <p className="text-xs text-zinc-500">{t("auth.adminDesc")}</p>
                </div>
              </div>

              <form onSubmit={handleAdminSubmit} className="space-y-3">
                <div className="space-y-1.5">
                  <label htmlFor="admin-username" className="text-xs font-medium text-zinc-400">{t("auth.username")}</label>
                  <input
                    id="admin-username"
                    type="text"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    placeholder={t("auth.usernamePlaceholder")}
                    autoComplete="username"
                    autoCapitalize="none"
                    autoCorrect="off"
                    spellCheck={false}
                    required
                    aria-required="true"
                    aria-invalid={error ? true : undefined}
                    className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 outline-none transition-[border-color,box-shadow] duration-150 focus:border-yellow-400/70 focus:shadow-[0_0_0_3px_var(--volt-glow-soft)]"
                  />
                </div>

                <div className="space-y-1.5">
                  <label htmlFor="admin-password" className="text-xs font-medium text-zinc-400">{t("auth.password")}</label>
                  <div className="relative">
                    <input
                      id="admin-password"
                      type={showPw ? "text" : "password"}
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      placeholder="••••••••"
                      autoComplete="current-password"
                      required
                      aria-required="true"
                      aria-invalid={error ? true : undefined}
                      className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 pr-10 text-sm text-zinc-100 placeholder-zinc-600 outline-none transition-[border-color,box-shadow] duration-150 focus:border-yellow-400/70 focus:shadow-[0_0_0_3px_var(--volt-glow-soft)]"
                    />
                    <button
                      type="button"
                      aria-label={showPw ? t("auth.hidePassword") : t("auth.showPassword")}
                      aria-pressed={showPw}
                      onClick={() => setShowPw((v) => !v)}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-500 hover:text-zinc-300 transition-colors"
                    >
                      {showPw ? <EyeOff size={15} aria-hidden="true" /> : <Eye size={15} aria-hidden="true" />}
                    </button>
                  </div>
                </div>

                <button
                  type="submit"
                  disabled={loading}
                  className="flex w-full items-center justify-center gap-2 rounded-lg bg-yellow-400 py-2.5 text-sm font-bold text-zinc-950 transition-opacity hover:opacity-90 disabled:opacity-50"
                >
                  <LogIn size={15} />
                  {loading ? t("auth.loggingIn") : t("auth.adminLogin")}
                </button>
              </form>
            </>
          )}

          {tab === "dev" && isDev && (
            <>
              <div className="flex items-center gap-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-green-500/10">
                  <FlaskConical size={15} className="text-green-400" />
                </div>
                <div>
                  <h2 className="text-base font-bold text-zinc-100">Dev Login</h2>
                  <p className="text-xs text-zinc-500">Войти как тестовый игрок</p>
                </div>
              </div>
              <div className="space-y-1.5">
                {devUsers.map((u) => (
                  <button
                    key={u.id}
                    onClick={async () => {
                      setLoading(true);
                      setError("");
                      try {
                        await devLogin(u.id);
                        router.replace("/");
                      } catch {
                        setError(t("auth.errorCreds"));
                      } finally {
                        setLoading(false);
                      }
                    }}
                    disabled={loading}
                    className="flex w-full items-center justify-between rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2.5 text-sm hover:border-green-500/50 hover:bg-zinc-750 transition-colors disabled:opacity-50"
                  >
                    <span className="font-semibold text-zinc-100">{u.display_name}</span>
                    <span className="text-xs text-zinc-500">{u.rank} · {u.rating} ELO</span>
                  </button>
                ))}
                {devUsersLoading && (
                  <p className="text-xs text-zinc-500 text-center py-4">{t("common.loading")}</p>
                )}
                {!devUsersLoading && devUsers.length === 0 && (
                  <p className="text-xs text-red-400 text-center py-4">Не удалось загрузить. Перезапусти сервер.</p>
                )}
              </div>
            </>
          )}

          {error && (
            <div className="rounded-lg border border-red-500/20 bg-red-500/5 px-3 py-2 text-xs text-red-400">
              {error}
            </div>
          )}
        </div>

        {/* Кинематографичное интро — пересмотр в любой момент (гости видят
            его автоматически при первом заходе). */}
        <Link
          href="/story"
          className="mt-4 flex items-center justify-center gap-1.5 py-2 text-xs font-semibold text-zinc-500 transition-colors hover:text-yellow-400"
        >
          ▶ {t("auth.watchIntro")}
        </Link>
      </m.div>
    </div>
  );
}
