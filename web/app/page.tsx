"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { m } from "framer-motion";
import { useQueries, useQuery } from "@tanstack/react-query";
import {
  AlertTriangle, Bell, ChevronRight, Clock,
  ListOrdered, LogIn, Trophy, Users, Zap,
} from "lucide-react";
import { useState, useMemo } from "react";
import { CountUp } from "@/components/CountUp";
import { EmptyState } from "@/components/EmptyState";
import { MatchCard } from "@/components/MatchCard";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { LeagueStatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import { SkeletonTable } from "@/components/ui/skeleton";
import { fetchLeagues, fetchMyLeagues, fetchMyMatches, fetchPlayers } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useLang } from "@/lib/i18n";
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
  const { t } = useLang();
  const router = useRouter();
  const [tab, setTab] = useState<Tab>("overview");

  const { data: leagues = [], isLoading: loadingLeagues } = useQuery({ queryKey: ["leagues"], queryFn: fetchLeagues, staleTime: 30000 });
  const { data: players = [], isLoading: loadingPlayers } = useQuery({ queryKey: ["players", 50], queryFn: () => fetchPlayers(50), staleTime: 60000 });
  const { data: myLeagues = [] } = useQuery({ queryKey: ["me", "leagues"], queryFn: fetchMyLeagues, enabled: !!user, staleTime: 30000 });

  const joinedIds = useMemo(() => new Set(myLeagues.filter((m) => m.status === "approved").map((m) => m.league?.id).filter(Boolean)), [myLeagues]);

  const matchQueries = useQueries({
    queries: myLeagues.map((m) => ({
      queryKey: ["my-matches", m.league?.id],
      queryFn: () => fetchMyMatches(m.league!.id),
      enabled: !!user && !!m.league?.id,
    })),
  });
  const myMatches = useMemo(
    () => matchQueries.flatMap((q) => q.data ?? []),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [matchQueries.map((q) => q.dataUpdatedAt).join(",")]
  );
  const waitingForMe = useMemo(() => myMatches.filter((m) => {
    if (!user) return false;
    return (
      (m.home_user_id === user.id && (m.status === "scheduled" || m.status === "disputed")) ||
      (m.away_user_id === user.id && m.status === "pending_confirm")
    );
  }), [myMatches, user]);
  const pendingConfirm = useMemo(
    () => myMatches.filter((m) => m.away_user_id === user?.id && m.status === "pending_confirm"),
    [myMatches, user?.id]
  );
  const activeLeagues = useMemo(() => leagues.filter((l) => l.status === "active"), [leagues]);
  const openLeagues   = useMemo(() => leagues.filter((l) => l.status === "registration"), [leagues]);
  const myActiveLeagues = useMemo(() => myLeagues.filter((m) =>
    m.league?.status === "active" || m.league?.status === "registration"
  ), [myLeagues]);

  const TABS = [
    { key: "overview" as Tab, label: t("dashboard.tabs.overview") },
    { key: "matches"  as Tab, label: t("dashboard.tabs.matches"), count: waitingForMe.length },
    { key: "rating"   as Tab, label: t("dashboard.tabs.rating") },
  ];

  return (
    <div className="space-y-5">
      {/* ── Header ── */}
      <div className="flex items-center justify-between">
        <div>
          <p className="text-[11px] font-bold uppercase tracking-widest text-zinc-400 mb-0.5">{t("dashboard.subtitle")}</p>
          <h1 className="font-display text-xl font-black text-zinc-100">{t("dashboard.title")}</h1>
        </div>
        <div className="flex items-center gap-2">
          {!user && !loading && (
            <Button asChild size="sm">
              <Link href="/login"><LogIn size={13} aria-hidden="true" /> {t("nav.login")}</Link>
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
                {waitingForMe.length} {t("dashboard.actionRequired_one")}
              </span>
              <span className="ml-2 text-xs text-red-400/70">{t("dashboard.enterScore")}</span>
            </div>
            <Button variant="destructive" size="sm" onClick={() => {
              const leagueId = waitingForMe[0]?.league_id;
              if (leagueId) router.push(`/leagues/details?id=${leagueId}&tab=my`);
            }}>{t("dashboard.goTo")}</Button>
          </div>
          {pendingConfirm.length > 0 && (
            <div className="flex items-center gap-3 rounded-xl border border-amber-500/20 bg-amber-500/8 px-4 py-3">
              <Clock size={16} className="text-amber-400 flex-shrink-0" />
              <span className="text-sm font-bold text-amber-300">
                {pendingConfirm.length} {t("dashboard.awaitConfirm_one")}
              </span>
            </div>
          )}
        </div>
      )}

      {/* ── User stats panel ── */}
      {user && (
        <div className="space-y-3">
          <div className="rounded-xl border border-zinc-800 bg-zinc-900 flex items-center gap-3 px-4 py-3">
            <PlayerAvatar
              displayName={user.display_name}
              favoriteClub={user.favorite_club}
              size={40}
              bgClassName="bg-yellow-400"
            />
            <div className="flex-1 min-w-0">
              <p className="text-sm font-bold text-zinc-100 truncate">{user.display_name}</p>
              <p className="text-xs text-zinc-400">{user.rank || t("common.rank")}</p>
            </div>
            <div className="text-right">
              <p className={cn("text-xl font-black tabular-nums", ratingColor(user.rating ?? 1000))}><CountUp value={user.rating ?? 1000} /></p>
              <p className="text-[9px] text-zinc-400 uppercase">ELO</p>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2">
            {[
              { id: "leagues",    label: t("nav.leagues"),             value: myActiveLeagues.length,                   sub: `${activeLeagues.length} ${t("dashboard.leaguesCount")}`, icon: Trophy, color: "text-yellow-400" },
              { id: "matches",  label: t("dashboard.tabs.matches"), value: myMatches.length,             sub: `${waitingForMe.length} ${t("dashboard.matchesWaiting")}`, icon: Bell, color: waitingForMe.length > 0 ? "text-red-400" : "text-zinc-400" },
              { id: "players",  label: t("nav.players"),             value: players.length,               sub: t("common.inSystem"),                                       icon: Users, color: "text-blue-400" },
              { id: "power",    label: t("common.teamPower"),        value: (user.team_power || 0).toLocaleString(), sub: "",                                              icon: Zap, color: "text-zinc-400" },
            ].map((m) => {
              const Icon = m.icon;
              return (
                <div key={m.id} className="rounded-xl border border-zinc-800 bg-zinc-900 p-3 flex flex-col justify-between">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-[10px] font-bold uppercase tracking-wide text-zinc-400">{m.label}</span>
                    <Icon size={13} className={m.color} />
                  </div>
                  <p className="text-2xl font-black text-zinc-100 leading-none">{m.value}</p>
                  <p className="text-[10px] text-zinc-400 mt-1">{m.sub}</p>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Guest quick stats */}
      {!user && (
        <m.div variants={stagger} initial="hidden" animate="show"
          className="grid grid-cols-2 gap-3"
        >
          {[
            { id: "leagues", label: t("nav.leagues"), value: leagues.length, color: "text-yellow-400", icon: Trophy },
            { id: "players", label: t("nav.players"), value: players.length, color: "text-blue-400",   icon: Users  },
          ].map((card) => {
            const Icon = card.icon;
            return (
              <m.div key={card.id} variants={fadeUp}
                className="rounded-xl border border-zinc-800 bg-zinc-900 p-4"
              >
                <div className="flex items-center gap-2 mb-2">
                  <Icon size={14} className={card.color} />
                  <span className="text-[10px] font-bold uppercase tracking-wide text-zinc-400">{card.label}</span>
                </div>
                <p className="text-3xl font-black text-zinc-100">{card.value}</p>
              </m.div>
            );
          })}
        </m.div>
      )}

      {/* ── Tabs ── */}
      <div className="flex items-center gap-1 border-b border-zinc-700">
        {TABS.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={cn(
              "flex items-center gap-2 px-4 py-2.5 text-sm font-semibold transition-colors border-b-2 -mb-px",
              tab === t.key
                ? "border-yellow-400 text-yellow-400"
                : "border-transparent text-zinc-300 hover:text-white"
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
                <Trophy size={14} className="text-yellow-400" /> {t("dashboard.topPlayers")}
              </h2>
              <Link href="/players" className="text-xs text-yellow-400 hover:text-yellow-300 flex items-center gap-1">
                Все <ChevronRight size={12} aria-hidden="true" />
              </Link>
            </div>
            {loadingPlayers ? (
              <SkeletonTable rows={5} />
            ) : players.length === 0 ? (
              <div className="rounded-xl border border-zinc-800 bg-zinc-900">
                <EmptyState icon={Users} title={t("dashboard.noPlayers")} text={t("dashboard.noPlayersText")} />
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
                          : <span className="text-xs font-black text-zinc-500">{i + 1}</span>
                        }
                      </span>
                      <PlayerAvatar
                        displayName={p.display_name}
                        favoriteClub={p.favorite_club}
                        size={28}
                        bgClassName="bg-zinc-800"
                      />
                      <div className="flex-1 min-w-0">
                        <p className={cn("text-sm font-semibold truncate", isMe ? "text-yellow-300" : "text-zinc-200")}>
                          {p.display_name}
                          {isMe && <span className="ml-1.5 text-[9px] font-bold text-yellow-400 uppercase">{t("common.you")}</span>}
                        </p>
                        <p className="text-xs text-zinc-400">{p.rank || t("common.rank")}</p>
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
                  <Trophy size={13} className="text-yellow-400" /> {t("nav.leagues")}
                </div>
                <Link href="/leagues" className="text-xs text-yellow-400 hover:text-yellow-300 flex items-center gap-1">
                  Все <ChevronRight size={12} aria-hidden="true" />
                </Link>
              </div>
              {loadingLeagues ? <SkeletonTable rows={3} /> :
               leagues.length === 0 ? (
                <EmptyState icon={Trophy} title={t("dashboard.noLeagues")} text={t("dashboard.noLeaguesText")} />
              ) : (
                <div>
                  {[...activeLeagues, ...openLeagues].slice(0, 5).map((league) => (
                    <Link key={league.id} href={`/leagues/details?id=${league.id}`}
                      className="flex items-center gap-3 px-4 py-3 border-b border-zinc-800/50 last:border-0 hover:bg-zinc-800/40 transition-colors group"
                    >
                      <div className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl bg-yellow-500/10 text-yellow-400">
                        <Trophy size={15} aria-hidden="true" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-bold text-zinc-200 truncate group-hover:text-yellow-400 transition-colors">{league.name}</p>
                        <p className="text-xs text-zinc-400">{league.rounds_type === "double" ? t("common.doubleRound") : t("common.singleRound")}</p>
                      </div>
                      <LeagueStatusBadge status={league.status} />
                    </Link>
                  ))}
                </div>
              )}
            </div>

            <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
              <div className="flex items-center gap-2 px-4 py-3 border-b border-zinc-800 text-xs font-bold uppercase tracking-wider text-zinc-400">
                <ListOrdered size={13} className="text-blue-400" /> {t("dashboard.myLeagues")}
              </div>
              {!user ? (
                <div className="p-6 flex flex-col items-center gap-3">
                  <p className="text-sm text-zinc-400 text-center">{t("dashboard.loginToSeeLeagues")}</p>
                  <Button asChild size="sm">
                    <Link href="/login"><LogIn size={14} aria-hidden="true" /> {t("nav.login")}</Link>
                  </Button>
                </div>
              ) : myLeagues.length === 0 ? (
                <EmptyState icon={ListOrdered} title={t("dashboard.notInLeagues")} text={t("dashboard.joinLeague")} />
              ) : (
                <div>
                  {myLeagues.map((m) => (
                    <Link key={m.league?.id} href={`/leagues/details?id=${m.league?.id}`}
                      className="flex items-center justify-between gap-3 px-4 py-3 border-b border-zinc-800/50 last:border-0 hover:bg-zinc-800/40 transition-colors"
                    >
                      <div className="min-w-0">
                        <p className="text-sm font-bold text-zinc-200 truncate">{m.league?.name}</p>
                        <p className="text-xs text-zinc-400">{m.wins}В · {m.draws}Н · {m.losses}П</p>
                      </div>
                      <div className="text-right flex-shrink-0">
                        <p className="text-xl font-black text-yellow-400 tabular-nums"><CountUp value={m.points} /></p>
                        <p className="text-[9px] text-zinc-400 uppercase">очков</p>
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
              <Button asChild><Link href="/login"><LogIn size={14} aria-hidden="true" /> {t("nav.login")}</Link></Button>
            </div>
          ) : waitingForMe.length === 0 ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900">
              <EmptyState icon={Bell} title={t("dashboard.noActions")} text={t("dashboard.noActionsText")} />
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
                i < 3 ? "text-yellow-400" : "text-zinc-500"
              )}>
                {i + 1}
              </span>
              <PlayerAvatar
                displayName={p.display_name}
                favoriteClub={p.favorite_club}
                size={30}
                bgClassName="bg-zinc-800"
              />
              <div className="flex-1 min-w-0">
                <p className={cn("text-sm font-bold truncate", p.id === user?.id ? "text-yellow-300" : "text-zinc-200")}>
                  {p.display_name}
                  {p.id === user?.id && <span className="ml-1.5 text-[9px] font-bold text-yellow-400 uppercase">{t("common.you")}</span>}
                </p>
                <p className="text-xs text-zinc-400">{p.rank || t("common.rank")}</p>
              </div>
              <span className={cn("text-base font-black tabular-nums", ratingColor(p.rating))}>{p.rating}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
