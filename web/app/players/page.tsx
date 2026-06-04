"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { Crown, Shirt, Target, Trophy, Users } from "lucide-react";
import { EmptyState } from "@/components/EmptyState";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { SkeletonTable } from "@/components/ui/skeleton";
import { fetchPlayers, fetchTopScorers } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

type Tab = "rating" | "scorers";

function tierColor(rating: number) {
  if (rating >= 1200) return { dot: "bg-yellow-400", text: "text-yellow-400", badge: "bg-yellow-500/10 text-yellow-400 border-yellow-500/30" };
  if (rating >= 1100) return { dot: "bg-blue-400",   text: "text-blue-400",   badge: "bg-blue-400/10 text-blue-400 border-blue-400/30"   };
  if (rating >= 1050) return { dot: "bg-purple-400", text: "text-purple-400", badge: "bg-purple-400/10 text-purple-400 border-purple-400/30" };
  return               { dot: "bg-green-500",  text: "text-green-400",  badge: "bg-green-500/10 text-green-400 border-green-500/30"  };
}

function positionBadge(index: number) {
  if (index === 0) return "bg-yellow-400 text-zinc-900";
  if (index === 1) return "bg-zinc-300 text-zinc-900";
  if (index === 2) return "bg-amber-600 text-white";
  return "bg-zinc-800 text-zinc-400";
}

export default function PlayersPage() {
  const { user } = useAuth();
  const { t } = useLang();
  const [tab, setTab] = useState<Tab>("rating");
  const { data: players = [], isLoading } = useQuery({ queryKey: ["players", 300], queryFn: () => fetchPlayers(300), staleTime: 60000 });
  const { data: topScorers = [] } = useQuery({ queryKey: ["top-scorers"], queryFn: fetchTopScorers, staleTime: 60000 });

  const TABS = [
    { key: "rating"  as Tab, label: t("players.tabElo"),     icon: Crown  },
    { key: "scorers" as Tab, label: t("players.tabScorers"), icon: Target },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-zinc-500 mb-1">{t("players.subtitle")}</p>
          <h1 className="text-2xl font-bold text-zinc-100">{t("players.title")}</h1>
        </div>
        <div className="text-right">
          <p className="text-lg font-black text-yellow-400">{players.length}</p>
          <p className="text-[10px] uppercase text-zinc-600 tracking-wide">{t("common.players")}</p>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1 border-b border-zinc-700">
        {TABS.map((t) => {
          const Icon = t.icon;
          return (
            <button key={t.key} onClick={() => setTab(t.key)}
              className={cn(
                "flex items-center gap-2 px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px",
                tab === t.key ? "border-yellow-400 text-yellow-400" : "border-transparent text-zinc-300 hover:text-white"
              )}
            >
              <Icon size={15} />
              {t.label}
            </button>
          );
        })}
      </div>

      {/* ── ELO Table ── */}
      {tab === "rating" && (
        isLoading ? <SkeletonTable rows={10} /> :
        players.length === 0 ? (
          <div className="rounded-xl border border-zinc-800 bg-zinc-900">
            <EmptyState icon={Users} title={t("players.noPlayers")} text={t("players.noPlayersText")} />
          </div>
        ) : (
          <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
            {/* Table header */}
            <div className="grid grid-cols-[40px_1fr_80px_80px_60px_90px] gap-2 px-4 py-2.5 border-b border-zinc-800 text-[10px] font-bold uppercase tracking-widest text-zinc-600">
              <div className="text-center">{t("players.colNum")}</div>
              <div>{t("players.colPlayer")}</div>
              <div className="text-center hidden sm:block">{t("players.colTier")}</div>
              <div className="text-center hidden sm:block">{t("players.colPower")}</div>
              <div className="text-center hidden sm:block">Win%</div>
              <div className="text-right">{t("players.colElo")}</div>
            </div>

            {/* Rows */}
            {players.map((player, index) => {
              const tier = tierColor(player.rating);
              const isMe = player.id === user?.id;
              const tierLabel = player.rating >= 1200 ? t("players.tierGold")
                : player.rating >= 1100 ? t("players.tierBlue")
                : player.rating >= 1050 ? t("players.tierPurple")
                : t("players.tierGreen");
              const played = (player as any).wins + (player as any).draws + (player as any).losses;
              const winPct = played > 0
                ? ((((player as any).wins ?? 0) / played) * 100).toFixed(0) + "%"
                : "—";
              return (
                <div
                  key={player.id}
                  className={cn(
                    "grid grid-cols-[40px_1fr_80px_80px_60px_90px] gap-2 px-4 py-3 items-center",
                    "border-b border-zinc-800/60 last:border-0 transition-colors",
                    isMe ? "bg-yellow-500/5 border-l-2 border-l-yellow-500" : "hover:bg-zinc-800/30"
                  )}
                >
                  {/* Position */}
                  <div className="flex justify-center">
                    <span className={cn(
                      "flex h-6 w-6 items-center justify-center rounded-full text-[11px] font-black",
                      positionBadge(index)
                    )}>
                      {index < 3 ? (index === 0 ? "🥇" : index === 1 ? "🥈" : "🥉") : index + 1}
                    </span>
                  </div>

                  {/* Player */}
                  <div className="flex items-center gap-2.5 min-w-0">
                    <PlayerAvatar
                      displayName={player.display_name}
                      favoriteClub={player.favorite_club}
                      size={32}
                      bgClassName={
                        index === 0 ? "bg-yellow-400" :
                        index === 1 ? "bg-zinc-300" :
                        index === 2 ? "bg-amber-600" : "bg-zinc-700"
                      }
                    />
                    <div className="min-w-0">
                      <Link href={`/players/${player.id}`} className={cn("text-sm font-semibold truncate hover:underline", isMe ? "text-yellow-300" : "text-zinc-100")}>
                        {player.display_name}
                        {isMe && <span className="ml-1.5 text-[9px] font-bold text-yellow-400 uppercase">{t("common.you")}</span>}
                      </Link>
                      <p className="text-[10px] text-zinc-600 truncate">{player.rank || t("common.rank")}</p>
                    </div>
                  </div>

                  {/* Tier badge */}
                  <div className="hidden sm:flex justify-center">
                    <span className={cn("rounded-full border px-2 py-0.5 text-[9px] font-bold uppercase flex items-center gap-1", tier.badge)}>
                      <span className={cn("h-1.5 w-1.5 rounded-full flex-shrink-0", tier.dot)} />
                      {tierLabel}
                    </span>
                  </div>

                  {/* Team power */}
                  <div className="hidden sm:flex justify-center items-center gap-1">
                    <span className="text-xs text-zinc-400 font-medium">{(player.team_power || 0).toLocaleString()}</span>
                  </div>

                  {/* Win% */}
                  <div className="hidden sm:flex justify-center items-center">
                    <span className="text-xs text-zinc-400 font-medium tabular-nums">{winPct}</span>
                  </div>

                  <div className="text-right">
                    <p className={cn("text-base font-black tabular-nums", tier.text)}>{player.rating}</p>
                    <p className="text-[9px] text-zinc-600 uppercase">{t("common.elo")}</p>
                  </div>
                </div>
              );
            })}
          </div>
        )
      )}

      {/* ── Scorers ── */}
      {tab === "scorers" && (
        <div className="space-y-4">
          {topScorers.length === 0 ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900">
              <EmptyState icon={Target} title={t("players.noGoals")} text={t("players.noGoalsText")} />
            </div>
          ) : topScorers.map((league) => (
            <div key={league.league.id} className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
              <div className="flex items-center gap-2 px-4 py-3 border-b border-zinc-800 text-xs font-semibold uppercase tracking-wider text-zinc-400">
                <Trophy size={13} /> {league.league.name}
              </div>
              <div>
                {league.scorers.map((scorer, i) => (
                  <div key={scorer.user_id}
                    className="grid grid-cols-[36px_1fr_auto] gap-3 px-4 py-2.5 items-center border-b border-zinc-800/50 last:border-0"
                  >
                    <span className={cn("flex h-6 w-6 items-center justify-center rounded-full text-[11px] font-black mx-auto", positionBadge(i))}>
                      {i < 3 ? (i === 0 ? "🥇" : i === 1 ? "🥈" : "🥉") : scorer.position}
                    </span>
                    <div>
                      <p className="text-sm font-semibold text-zinc-200 truncate">{scorer.display_name}</p>
                      <p className="text-xs text-zinc-500 flex items-center gap-1">
                        <Shirt size={10} /> {scorer.team_power || 0}
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="text-lg font-black text-yellow-400">{scorer.goals_for}</p>
                      <p className="text-[10px] text-zinc-600 uppercase">{t("players.goals")}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
