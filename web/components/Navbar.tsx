"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useMemo } from "react";
import { AnimatePresence, m } from "framer-motion";
import {
  Home, Trophy, ListOrdered, User, Shield,
  LogIn, ChevronLeft, ChevronRight, Medal, Settings, RefreshCw, MessageSquare,
} from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { fetchLeagues } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { BrandLogo } from "@/components/BrandLogo";
import { openLogoShowcase } from "@/components/LogoShowcase";
import { NotificationBell } from "@/components/NotificationBell";
import { useUnreadTotal } from "@/lib/chat";
import { useLang } from "@/lib/i18n";
import { useUIStore } from "@/lib/store";
import { cn } from "@/lib/utils";

export function Navbar() {
  const pathname = usePathname();
  const { user } = useAuth();
  const { sidebarCollapsed, toggleSidebar } = useUIStore();
  const { t } = useLang();

  const unreadDM = useUnreadTotal(!!user);

  const { data: leagues = [] } = useQuery({ queryKey: ["leagues"], queryFn: fetchLeagues, staleTime: 60000 });
  const openLeaguesCount = useMemo(
    () => leagues.filter((l) => l.status === "registration").length,
    [leagues]
  );

  const NAV = useMemo(() => [
    { href: "/",             label: t("nav.home"),        icon: Home,        dot: false              },
    { href: "/leagues",      label: t("nav.leagues"),     icon: Trophy,      dot: openLeaguesCount > 0 },
    { href: "/players",      label: t("nav.players"),     icon: ListOrdered, dot: false              },
    { href: "/hall-of-fame", label: t("nav.hallOfFame"),  icon: Medal,       dot: false              },
    { href: "/profile",      label: t("nav.profile"),     icon: User,        dot: false, auth: true  },
    { href: "/settings",     label: t("nav.settings"),    icon: Settings,    dot: false, auth: true  },
  ], [t, openLeaguesCount]);

  const isActive = (href: string) =>
    href === "/" ? pathname === "/" : pathname.startsWith(href);

  return (
    <>
      {/* ─── Skip to main content ─────────────────────────────────── */}
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-50 focus:rounded-lg focus:bg-yellow-400 focus:px-4 focus:py-2 focus:text-sm focus:font-bold focus:text-zinc-900"
      >
        Перейти к содержимому
      </a>

      {/* ─── Desktop sidebar ─────────────────────────────────────── */}
      <aside
        className={cn(
          "hidden lg:flex fixed top-0 left-0 bottom-0 z-40 flex-col",
          "bg-zinc-950 border-r border-zinc-800 overflow-hidden",
          "transition-[width] duration-200 ease-in-out",
          sidebarCollapsed ? "w-16" : "w-[220px]"
        )}
      >
        {/* Brand */}
        <div className={cn(
          "flex items-center gap-2 border-b border-zinc-800 h-14 flex-shrink-0",
          sidebarCollapsed ? "justify-center px-0" : "px-4"
        )}>
          <button onClick={openLogoShowcase} aria-label="Логотип eFootLeague" className="flex-shrink-0 transition-transform active:scale-90">
            <BrandLogo size={34} />
          </button>
          <AnimatePresence>
            {!sidebarCollapsed && (
              <m.div
                initial={{ opacity: 0, x: -8 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -8 }}
                transition={{ duration: 0.15 }}
                className="min-w-0 overflow-hidden"
              >
                <p className="font-display text-xs font-bold text-zinc-100 leading-none whitespace-nowrap">eFootLeague</p>
                <p className="text-[10px] text-zinc-400 leading-none mt-0.5 uppercase tracking-wider">Web League</p>
              </m.div>
            )}
          </AnimatePresence>
          {user && !sidebarCollapsed && (
            <div className="ml-auto flex-shrink-0">
              <NotificationBell align="left" />
            </div>
          )}
        </div>

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto py-3 px-2 space-y-0.5">
          {NAV.map((item) => {
            if (item.auth && !user) return null;
            const Icon = item.icon;
            const active = isActive(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                title={sidebarCollapsed ? item.label : undefined}
                className={cn(
                  "relative flex items-center gap-3 rounded-lg py-2 text-sm font-medium transition-all duration-150",
                  sidebarCollapsed ? "justify-center px-2" : "px-3",
                  active
                    ? "bg-yellow-500/10 text-yellow-400"
                    : "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
                )}
              >
                <div className="relative flex-shrink-0">
                  <Icon size={17} />
                  {item.dot && (
                    <span className="absolute -top-1 -right-1 flex h-2 w-2">
                      <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75" />
                      <span className="relative inline-flex rounded-full h-2 w-2 bg-red-500" />
                    </span>
                  )}
                </div>
                <AnimatePresence>
                  {!sidebarCollapsed && (
                    <m.span
                      initial={{ opacity: 0 }}
                      animate={{ opacity: 1 }}
                      exit={{ opacity: 0 }}
                      transition={{ duration: 0.12 }}
                      className="whitespace-nowrap overflow-hidden"
                    >
                      {item.label}
                    </m.span>
                  )}
                </AnimatePresence>
              </Link>
            );
          })}

          {user && (
            <Link
              href="/messages"
              title={sidebarCollapsed ? "Сообщения" : undefined}
              className={cn(
                "flex items-center gap-3 rounded-lg py-2 text-sm font-medium transition-all duration-150",
                sidebarCollapsed ? "justify-center px-2" : "px-3",
                isActive("/messages")
                  ? "bg-yellow-500/10 text-yellow-400"
                  : "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
              )}
            >
              <div className="relative flex-shrink-0">
                <MessageSquare size={17} />
                {unreadDM > 0 && (
                  <span className="absolute -top-1.5 -right-1.5 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-yellow-400 px-1 text-[9px] font-bold text-zinc-900">
                    {unreadDM > 99 ? "99+" : unreadDM}
                  </span>
                )}
              </div>
              <AnimatePresence>
                {!sidebarCollapsed && (
                  <m.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                    className="whitespace-nowrap overflow-hidden"
                  >
                    Сообщения
                  </m.span>
                )}
              </AnimatePresence>
            </Link>
          )}

          {user?.is_admin && (
            <Link
              href="/admin"
              title={sidebarCollapsed ? t("nav.admin") : undefined}
              className={cn(
                "flex items-center gap-3 rounded-lg py-2 text-sm font-medium transition-all duration-150",
                sidebarCollapsed ? "justify-center px-2" : "px-3",
                isActive("/admin")
                  ? "bg-yellow-500/10 text-yellow-400"
                  : "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
              )}
            >
              <Shield size={17} className="flex-shrink-0" />
              <AnimatePresence>
                {!sidebarCollapsed && (
                  <m.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                    className="whitespace-nowrap overflow-hidden"
                  >
                    {t("nav.admin")}
                  </m.span>
                )}
              </AnimatePresence>
            </Link>
          )}
        </nav>

        {/* User footer */}
        <div className={cn("border-t border-zinc-800 p-2", sidebarCollapsed && "flex justify-center")}>
          {user ? (
            <div className={cn("flex items-center gap-2.5 rounded-lg p-2", !sidebarCollapsed && "pr-1")}>
              <PlayerAvatar
                displayName={user.display_name}
                favoriteClub={user.favorite_club}
                size={28}
                bgClassName="bg-yellow-400"
              />
              <AnimatePresence>
                {!sidebarCollapsed && (
                  <m.div
                    initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                    className="flex-1 min-w-0 overflow-hidden"
                  >
                    <p className="text-xs font-semibold text-zinc-200 truncate">{user.display_name}</p>
                    <p className="text-[10px] text-zinc-400">{user.rating} ELO</p>
                  </m.div>
                )}
              </AnimatePresence>
            </div>
          ) : (
            <Link
              href="/login"
              title={sidebarCollapsed ? t("nav.login") : undefined}
              className={cn(
                "flex items-center gap-3 rounded-lg py-2 text-sm text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100 transition-colors",
                sidebarCollapsed ? "justify-center px-2" : "px-3"
              )}
            >
              <LogIn size={17} />
              <AnimatePresence>
                {!sidebarCollapsed && (
                  <m.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                    className="whitespace-nowrap"
                  >
                    {t("nav.login")}
                  </m.span>
                )}
              </AnimatePresence>
            </Link>
          )}
        </div>

        {/* Collapse toggle */}
        <button
          onClick={toggleSidebar}
          aria-label={sidebarCollapsed ? "Развернуть меню" : "Свернуть меню"}
          className="absolute -right-3 top-[52px] flex h-6 w-6 items-center justify-center rounded-full border border-zinc-700 bg-zinc-900 text-zinc-400 hover:text-zinc-100 transition-colors z-10"
        >
          {sidebarCollapsed ? <ChevronRight size={12} /> : <ChevronLeft size={12} />}
        </button>
      </aside>

      {/* ─── Mobile top bar ──────────────────────────────────────── */}
      <header className="lg:hidden fixed top-0 left-0 right-0 z-40 flex h-14 items-center gap-3 border-b border-zinc-800 bg-zinc-950/95 backdrop-blur-sm px-4 pt-[env(safe-area-inset-top)] box-content">
        <button onClick={openLogoShowcase} aria-label="Логотип eFootLeague" className="flex-shrink-0 transition-transform active:scale-90">
          <BrandLogo size={30} />
        </button>
        <span className="font-display text-sm font-bold text-zinc-100">eFootLeague</span>
        <div className="ml-auto flex items-center gap-2">
          {user && (
            <Link
              href="/messages"
              aria-label="Сообщения"
              className={cn(
                "relative rounded-md p-2 transition-colors",
                isActive("/messages") ? "text-yellow-400" : "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
              )}
            >
              <MessageSquare size={16} />
              {unreadDM > 0 && (
                <span className="absolute -top-0.5 -right-0.5 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-yellow-400 px-1 text-[10px] font-bold text-zinc-900">
                  {unreadDM > 99 ? "99+" : unreadDM}
                </span>
              )}
            </Link>
          )}
          {user && <NotificationBell align="right" />}
          <button
            onClick={() => window.location.reload()}
            aria-label="Обновить"
            className="rounded-md p-2 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100 transition-colors"
          >
            <RefreshCw size={16} />
          </button>
        </div>
      </header>

      {/* ─── Mobile bottom nav ───────────────────────────────────── */}
      <nav className="lg:hidden fixed bottom-0 left-0 right-0 z-40 flex items-stretch border-t border-zinc-800 bg-zinc-950/95 backdrop-blur-sm pb-[env(safe-area-inset-bottom)]">
        {NAV.map((item) => {
          if (item.auth && !user) return null;
          const Icon = item.icon;
          const active = isActive(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "group flex flex-1 select-none flex-col items-center justify-start gap-0.5 pt-1.5 pb-1.5 px-0.5 text-[9px] font-medium uppercase tracking-tight transition-colors",
                active ? "text-yellow-400" : "text-zinc-400"
              )}
            >
              <div className="relative flex h-7 w-12 items-center justify-center transition-transform duration-200 group-active:scale-90">
                {/* Пилюля активного таба перетекает между иконками (layoutId) */}
                {active && (
                  <m.div
                    layoutId="bottomnav-pill"
                    className="absolute inset-0 rounded-full bg-yellow-400/15"
                    transition={{ type: "spring", stiffness: 420, damping: 32 }}
                  />
                )}
                <Icon size={20} className={cn("relative z-10 transition-transform duration-200", active && "scale-110")} />
                {item.dot && (
                  <span className="absolute top-0.5 right-2 flex h-2 w-2">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75" />
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-red-500" />
                  </span>
                )}
              </div>
              <span className="text-center leading-[1.1] line-clamp-2">{item.label}</span>
            </Link>
          );
        })}
        {!user && (
          <Link href="/login" className="group flex flex-1 select-none flex-col items-center justify-start gap-0.5 pt-1.5 pb-1.5 px-0.5 text-[9px] font-medium uppercase tracking-tight text-zinc-400">
            <div className="flex h-7 w-12 items-center justify-center rounded-full transition-transform duration-200 group-active:scale-90">
              <LogIn size={20} />
            </div>
            <span className="text-center leading-[1.1] line-clamp-2">{t("nav.login")}</span>
          </Link>
        )}
        {user?.is_admin && (
          <Link href="/admin" className={cn(
            "group flex flex-1 select-none flex-col items-center justify-start gap-0.5 pt-1.5 pb-1.5 px-0.5 text-[9px] font-medium uppercase tracking-tight transition-colors",
            isActive("/admin") ? "text-yellow-400" : "text-zinc-400"
          )}>
            <div className="relative flex h-7 w-12 items-center justify-center transition-transform duration-200 group-active:scale-90">
              {isActive("/admin") && (
                <m.div
                  layoutId="bottomnav-pill"
                  className="absolute inset-0 rounded-full bg-yellow-400/15"
                  transition={{ type: "spring", stiffness: 420, damping: 32 }}
                />
              )}
              <Shield size={20} className={cn("relative z-10 transition-transform duration-200", isActive("/admin") && "scale-110")} />
            </div>
            <span className="text-center leading-[1.1] line-clamp-2">{t("nav.adminShort")}</span>
          </Link>
        )}
      </nav>
    </>
  );
}
