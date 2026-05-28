"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { useQueries, useQuery } from "@tanstack/react-query";
import {
  AlertTriangle, Bell, ChevronRight, Clock,
  ListOrdered, LogIn, RefreshCw, Trophy, Users, Zap,
} from "lucide-react";
import { useState } from "react";
import { EmptyState } from "@/components/EmptyState";
import { MatchCard } from "@/components/MatchCard";
import { LeagueStatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import { SkeletonTable } from "@/components/ui/skeleton";
import { fetchLeagues, fetchMyLeagues, fetchMyMatches, fetchPlayers } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/utils";

type Tab = "overview" | "matches" | "rating";

const fadeUp = { hidden: { opacity: 0, y: 12 }, show: { opacity: 1, y: 0 } };
const stagger = { show: { transition: { staggerChildren: 0.06 } } };

function ratingColor(rating: number) {
  if (rating >= 1200) return "text-yellow-400";
  if (rating >= 1100) return "text-blue-400";
  if (rating >= 1050) return "text-purple-400";
  return "text-green-400";
}

export default function HomePage() {
  const { user, loading } = useAuth();
  const [tab, setTab] = useState<Tab>("overview");

  const { data: leagues = [], isLoading: loadingLeagues } = useQuery({ queryKey: ["leagues"], queryFn: fetchLeagues });
  const { data: players = [], isLoading: loadingPlayers } = useQuery({ queryKey: ["players", 50], queryFn: () => fetchPlayers(50) });
  const { data: myLeagues = [] } = useQuery({ queryKey: ["me", "leagues"], queryFn: fetchMyLeagues, enabled: !!user });

  const matchQueries = useQueries({
    queries: myLeagues.map((m) => ({
      queryKey: ["my-matches", m.league?.id],
      queryFn: () => fetchMyMatches(m.league!.id),
      enabled: !!user && !!m.league?.id,
    })),
  });
  const myMatches = matchQueries.flatMap((q) => q.data ?? []);
  const waitingForMe = myMatches.filter((m) => {
    if (!user) return false;
    return (
      (m.home_user_id === user.id && (m.status === "scheduled" || m.status === "disputed")) ||
      (m.away_user_id === user.id && m.status === "pending_confirm")
    );
  });
  const pendingConfirm = myMatches.filter((m) => m.away_user_id === user?.id && m.status === "pending_confirm");
  const activeLeagues = leagues.filter((l) => l.status === "active");
  const openLeagues   = leagues.filter((l) => l.status === "registration");

  const TABS = [
    { key: "overview" as Tab, label: "Обзор" },
    { key: "matches"  as Tab, label: "Мои матчи", count: waitingForMe.length },
    { key: "rating"   as Tab, label: "Рейтинг" },
  ];

  return (
    <div className="space-y-5">
      {/* ── Header ── */}
      <div className="flex items-center justify-between">
        <div>
          <p className="text-[11px] font-bold uppercase tracking-widest text-zinc-500 mb-0.5">Главная</p>
          <h1 className="text-xl font-black text-zinc-100">Дашборд</h1>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => matchQueries.forEach((q) => q.refetch())}>
            <RefreshCw size={13} />
          </Button>
          {!user && !loading && (
            <Button asChild size="sm">
              <Link href="/login"><LogIn size={13} /> Войти</Link>
            </Button>
          )}
        </div>
      </div>

      {/* ── Alert banners ── */}
      {user && waitingForMe.length > 0 && (
        <div className="space-y-2">
          <div className="flex items-center gap-3 rounded-xl border border-red-500/20 bg-red-500/8 px-4 py-3">
            <AlertTriangle size={16} className="text-red-400 flex-shrink-0" />
            <div className="flex-1">
              <span className="text-sm font-bold text-red-300">
                {waitingForMe.length} матч{waitingForMe.length > 1 ? "а" : ""} требуют действия
              </span>
              <span className="ml-2 text-xs text-red-400/70">введите счёт или подтвердите результат</span>
            </div>
            <Button variant="destructive" size="sm" onClick={() => setTab("matches")}>Перейти</Button>
          </div>
          {pendingConfirm.length > 0 && (
            <div className="flex items-center gap-3 rounded-xl border border-amber-500/20 bg-amber-500/8 px-4 py-3">
              <Clock size={16} className="text-amber-400 flex-shrink-0" />
              <span className="text-sm font-bold text-amber-300">
                {pendingConfirm.length} матч{pendingConfirm.length > 1 ? "а" : ""} ждут вашего подтверждения
              </span>
            </div>
          )}
        </div>
      )}

      {/* ── User stats panel ── */}
      {user && (
        <div className="space-y-3">
          <div className="rounded-xl border border-zinc-800 bg-zinc-900 flex items-center gap-3 px-4 py-3">
            <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-yellow-400 text-zinc-900 text-sm font-black">
              {(user.display_name || "?").slice(0, 1).toUpperCase()}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-bold text-zinc-100 truncate">{user.display_name}</p>
              <p className="text-xs text-zinc-500">{user.rank || "Игрок"}</p>
            </div>
            <div className="text-right">
              <p className={cn("text-xl font-black tabular-nums", ratingColor(user.rating ?? 1000))}>{user.rating ?? 1000}</p>
              <p className="text-[9px] text-zinc-600 uppercase">ELO</p>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2">
            {[
              { label: "Лиги",    value: myLeagues.length,                        sub: `${activeLeagues.length} активных`, icon: Trophy, color: "text-yellow-400" },
              { label: "Матчи",   value: myMatches.length,                        sub: `${waitingForMe.length} ждут меня`, icon: Bell,   color: waitingForMe.length > 0 ? "text-red-400" : "text-zinc-400" },
              { label: "Игроков", value: players.length,                          sub: "в системе",                        icon: Users,  color: "text-blue-400" },
              { label: "Сила",    value: (user.team_power || 0).toLocaleString(), sub: "команды",                          icon: Zap,    color: "text-zinc-400" },
            ].map((m) => {
              const Icon = m.icon;
              return (
                <div key={m.label} className="rounded-xl border border-zinc-800 bg-zinc-900 p-3 flex flex-col justify-between">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-[10px] font-bold uppercase tracking-wide text-zinc-500">{m.label}</span>
                    <Icon size={13} className={m.color} />
                  </div>
                  <p className="text-2xl font-black text-zinc-100 leading-none">{m.value}</p>
                  <p className="text-[10px] text-zinc-600 mt-1">{m.sub}</p>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Guest quick stats */}
      {!user && (
        <motion.div variants={stagger} initial="hidden" animate="show"
          className="grid grid-cols-2 gap-3"
        >
          {[
            { label: "Лиг в системе", value: leagues.length, color: "text-yellow-400", icon: Trophy },
            { label: "Игроков",       value: players.length, color: "text-blue-400",   icon: Users  },
          ].map((m) => {
            const Icon = m.icon;
            return (
              <motion.div key={m.label} variants={fadeUp}
                className="rounded-xl border border-zinc-800 bg-zinc-900 p-4"
              >
                <div className="flex items-center gap-2 mb-2">
                  <Icon size={14} className={m.color} />
                  <span className="text-[10px] font-bold uppercase tracking-wide text-zinc-500">{m.label}</span>
                </div>
                <p className="text-3xl font-black text-zinc-100">{m.value}</p>
              </motion.div>
            );
          })}
        </motion.div>
      )}

      {/* ── Tabs ── */}
      <div className="flex items-center gap-1 border-b border-zinc-800">
        {TABS.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={cn(
              "flex items-center gap-2 px-4 py-2.5 text-sm font-semibold transition-colors border-b-2 -mb-px",
              tab === t.key
                ? "border-yellow-400 text-yellow-400"
                : "border-transparent text-zinc-500 hover:text-zinc-200"
            )}
          >
            {t.label}
            {t.count ? (
              <span className="flex h-4 min-w-[16px] items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-black text-white">
                {t.count}
              </span>
            ) : null}
          </button>
        ))}
      </div>

      {/* ── Tab: Overview ── */}
      {tab === "overview" && (
        <div className="space-y-5">
          {/* Top players */}
          <div>
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-sm font-black uppercase tracking-wider text-zinc-400 flex items-center gap-2">
                <Trophy size={14} className="text-yellow-400" /> Топ игроков
              </h2>
              <Link href="/players" className="text-xs text-yellow-400 hover:text-yellow-300 flex items-center gap-1">
                Все <ChevronRight size={12} />
              </Link>
            </div>
            {loadingPlayers ? (
              <SkeletonTable rows={5} />
            ) : players.length === 0 ? (
              <div className="rounded-xl border border-zinc-800 bg-zinc-900">
                <EmptyState icon={Users} title="Нет игроков" text="Рейтинг появится после регистрации." />
              </div>
            ) : (
              <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
                {players.slice(0, 5).map((p, i) => {
                  const isMe = p.id === user?.id;
                  return (
                    <div key={p.id} className={cn(
                      "flex items-center gap-3 px-4 py-2.5 border-b border-zinc-800/50 last:border-0 transition-colors",
                      isMe ? "bg-yellow-500/5 border-l-2 border-l-yellow-500" : "hover:bg-zinc-800/30"
                    )}>
                      <span className="w-6 text-center flex-shrink-0">
                        {i < 3
                          ? <span className="text-sm">{i === 0 ? "🥇" : i === 1 ? "🥈" : "🥉"}</span>
                          : <span className="text-xs font-black text-zinc-600">{i + 1}</span>
                        }
                      </span>
                      <div className="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-full bg-zinc-800 text-xs font-black text-zinc-300">
                        {(p.display_name || "?").slice(0, 1).toUpperCase()}
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className={cn("text-sm font-semibold truncate", isMe ? "text-yellow-300" : "text-zinc-200")}>
                          {p.display_name}
                          {isMe && <span className="ml-1.5 text-[9px] font-bold text-yellow-500 uppercase">вы</span>}
                        </p>
                        <p className="text-xs text-zinc-600">{p.rank || "Игрок"}</p>
                      </div>
                      <span className={cn("text-base font-black tabular-nums", ratingColor(p.rating))}>{p.rating}</span>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Active leagues + My leagues */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
              <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-800">
                <div className="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-zinc-400">
                  <Trophy size={13} className="text-yellow-500" /> Лиги
                </div>
                <Link href="/leagues" className="text-xs text-yellow-400 hover:text-yellow-300 flex items-center gap-1">
                  Все <ChevronRight size={12} />
                </Link>
              </div>
              {loadingLeagues ? <SkeletonTable rows={3} /> :
               leagues.length === 0 ? (
                <EmptyState icon={Trophy} title="Лиг пока нет" text="Администратор создаст их позже." />
              ) : (
                <div>
                  {[...activeLeagues, ...openLeagues].slice(0, 5).map((league) => (
                    <Link key={league.id} href={`/leagues/details?id=${league.id}`}
                      className="flex items-center gap-3 px-4 py-3 border-b border-zinc-800/50 last:border-0 hover:bg-zinc-800/40 transition-colors group"
                    >
                      <div className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl bg-yellow-500/10 text-yellow-500">
                        <Trophy size={15} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-bold text-zinc-200 truncate group-hover:text-yellow-400 transition-colors">{league.name}</p>
                        <p className="text-xs text-zinc-600">{league.rounds_type === "double" ? "Двойной круг" : "Один круг"}</p>
                      </div>
                      <LeagueStatusBadge status={league.status} />
                    </Link>
                  ))}
                </div>
              )}
            </div>

            <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
              <div className="flex items-center gap-2 px-4 py-3 border-b border-zinc-800 text-xs font-bold uppercase tracking-wider text-zinc-400">
                <ListOrdered size={13} className="text-blue-400" /> Мои лиги
              </div>
              {!user ? (
                <div className="p-6 flex flex-col items-center gap-3">
                  <p className="text-sm text-zinc-500 text-center">Войдите чтобы видеть свои лиги</p>
                  <Button asChild size="sm">
                    <Link href="/login"><LogIn size={14} /> Войти</Link>
                  </Button>
                </div>
              ) : myLeagues.length === 0 ? (
                <EmptyState icon={ListOrdered} title="Не в лигах" text="Вступите в лигу." />
              ) : (
                <div>
                  {myLeagues.map((m) => (
                    <Link key={m.league?.id} href={`/leagues/details?id=${m.league?.id}`}
                      className="flex items-center justify-between gap-3 px-4 py-3 border-b border-zinc-800/50 last:border-0 hover:bg-zinc-800/40 transition-colors"
                    >
                      <div className="min-w-0">
                        <p className="text-sm font-bold text-zinc-200 truncate">{m.league?.name}</p>
                        <p className="text-xs text-zinc-500">{m.wins}В · {m.draws}Н · {m.losses}П</p>
                      </div>
                      <div className="text-right flex-shrink-0">
                        <p className="text-xl font-black text-yellow-400">{m.points}</p>
                        <p className="text-[9px] text-zinc-600 uppercase">очков</p>
                      </div>
                    </Link>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* ── Tab: My matches ── */}
      {tab === "matches" && (
        <div className="space-y-3">
          {!user ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900 py-8 flex justify-center">
              <Button asChild><Link href="/login"><LogIn size={14} /> Войти</Link></Button>
            </div>
          ) : waitingForMe.length === 0 ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900">
              <EmptyState icon={Bell} title="Нет активных действий" text="Новые матчи, счета и подтверждения появятся здесь." />
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {waitingForMe.map((match) => (
                <MatchCard key={match.id} match={match} onUpdate={() => matchQueries.forEach((q) => q.refetch())} />
              ))}
            </div>
          )}
        </div>
      )}

      {/* ── Tab: Rating ── */}
      {tab === "rating" && (
        <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
          {players.map((p, i) => (
            <div key={p.id} className={cn(
              "flex items-center gap-3 px-4 py-2.5 border-b border-zinc-800/50 last:border-0",
              p.id === user?.id && "bg-yellow-500/5 border-l-2 border-l-yellow-500"
            )}>
              <span className={cn("w-6 text-center text-xs font-black flex-shrink-0",
                i < 3 ? "text-yellow-400" : "text-zinc-600"
              )}>
                {i + 1}
              </span>
              <div className="flex-1 min-w-0">
                <p className={cn("text-sm font-bold truncate", p.id === user?.id ? "text-yellow-300" : "text-zinc-200")}>
                  {p.display_name}
                  {p.id === user?.id && <span className="ml-1.5 text-[9px] font-bold text-yellow-500 uppercase">вы</span>}
                </p>
                <p className="text-xs text-zinc-500">{p.rank || "Игрок"}</p>
              </div>
              <span className={cn("text-base font-black tabular-nums", ratingColor(p.rating))}>{p.rating}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
