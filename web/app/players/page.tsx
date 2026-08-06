"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { Activity, Crown, Flame, LucideIcon, Shirt, Swords, Target, Trophy, TrendingUp, Zap } from "lucide-react";
import { EmptyState } from "@/components/EmptyState";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { SkeletonTable } from "@/components/ui/skeleton";
import {
  fetchPlayers, fetchTopScorers,
  fetchStatWinRate, fetchStatStreaks, fetchStatAvgGoals,
  fetchStatTeamPower, fetchStatActivity,
  StatEntry,
} from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useLang } from "@/lib/i18n";
import { usePresence } from "@/lib/presence";
import { cn } from "@/lib/utils";

type Tab = "rating" | "scorers" | "winrate" | "streaks" | "avggoals" | "power" | "activity";

function tierColor(rating: number) {
  if (rating >= 1200) return { dot: "bg-yellow-400", text: "text-yellow-400", badge: "bg-yellow-500/10 text-yellow-400 border-yellow-500/30" };
  if (rating >= 1100) return { dot: "bg-blue-400",   text: "text-blue-400",   badge: "bg-blue-400/10 text-blue-400 border-blue-400/30"   };
  if (rating >= 1050) return { dot: "bg-purple-400", text: "text-purple-400", badge: "bg-purple-400/10 text-purple-400 border-purple-400/30" };
  return               { dot: "bg-green-500",  text: "text-green-400",  badge: "bg-green-500/10 text-green-400 border-green-500/30"  };
}

function Pos({ i, id, me }: { i: number; id: number; me?: number }) {
  const isMe = id === me;
  if (i < 3) return <span className="text-lg leading-none select-none">{i === 0 ? "🥇" : i === 1 ? "🥈" : "🥉"}</span>;
  return <span className={cn("text-xs font-bold", isMe ? "text-yellow-400" : "text-zinc-500")}>{i + 1}</span>;
}

function StatRow({ entry, i, me, right }: {
  entry: StatEntry; i: number; me?: number;
  right: React.ReactNode;
}) {
  const { t } = useLang();
  const isMe = entry.user_id === me;
  return (
    <div className={cn(
      "flex items-center gap-3 px-4 py-2.5 border-b border-zinc-800/50 last:border-0 transition-colors",
      isMe && "bg-yellow-500/5 border-l-2 border-l-yellow-500"
    )}>
      <div className="w-6 flex justify-center flex-shrink-0">
        <Pos i={i} id={entry.user_id} me={me} />
      </div>
      <PlayerAvatar displayName={entry.display_name} favoriteClub={entry.favorite_club} size={34} />
      <Link href={`/players/details?id=${entry.user_id}`} className="flex-1 min-w-0 group">
        <p className={cn("text-sm font-semibold truncate group-hover:underline", isMe ? "text-yellow-300" : "text-zinc-200")}>
          <span className="sm:hidden">{entry.display_name.split(" ")[0]}</span>
          <span className="hidden sm:inline">{entry.display_name}</span>
          {isMe && <span className="ml-1.5 text-[9px] font-bold text-yellow-400 uppercase">{t("common.you")}</span>}
        </p>
        <p className="text-[10px] text-zinc-600">{entry.rank}</p>
      </Link>
      <div className="text-right flex-shrink-0">{right}</div>
    </div>
  );
}

function Val({ top, bot, color = "text-zinc-200" }: { top: React.ReactNode; bot: string; color?: string }) {
  return (
    <div>
      <p className={cn("text-base font-black tabular-nums leading-tight", color)}>{top}</p>
      <p className="text-[9px] text-zinc-400 uppercase tracking-wide">{bot}</p>
    </div>
  );
}

function EmptyCard({ icon, text }: { icon: LucideIcon; text: string }) {
  const { t } = useLang();
  return (
    <div className="rounded-xl card-premium">
      <EmptyState icon={icon} title={t("hallOfFame.noData")} text={text} />
    </div>
  );
}

export default function PlayersPage() {
  const { user } = useAuth();
  const { t } = useLang();
  const [tab, setTab] = useState<Tab>("rating");
  const me = user?.id;
  const presence = usePresence();

  const { data: players = [], isLoading: loadingPlayers } =
    useQuery({ queryKey: ["players", 300], queryFn: () => fetchPlayers(300), staleTime: 60000 });
  const { data: topScorers = [] } =
    useQuery({ queryKey: ["top-scorers"], queryFn: fetchTopScorers, staleTime: 60000 });
  const { data: winRate = [], isLoading: loadingWR } =
    useQuery({ queryKey: ["stats-winrate"], queryFn: fetchStatWinRate, staleTime: 60000 });
  const { data: streaks = [], isLoading: loadingStreaks } =
    useQuery({ queryKey: ["stats-streaks"], queryFn: fetchStatStreaks, staleTime: 60000 });
  const { data: avgGoals = [], isLoading: loadingAvg } =
    useQuery({ queryKey: ["stats-avggoals"], queryFn: fetchStatAvgGoals, staleTime: 60000 });
  const { data: power = [], isLoading: loadingPower } =
    useQuery({ queryKey: ["stats-power"], queryFn: fetchStatTeamPower, staleTime: 60000 });
  const { data: activity = [], isLoading: loadingActivity } =
    useQuery({ queryKey: ["stats-activity"], queryFn: fetchStatActivity, staleTime: 60000 });

  const TABS: { key: Tab; label: string; icon: LucideIcon }[] = [
    { key: "rating",   label: t("players.tabElo"),     icon: Crown      },
    { key: "scorers",  label: t("players.tabScorers"), icon: Target     },
    { key: "winrate",  label: t("players.tabWinRate"),  icon: TrendingUp },
    { key: "streaks",  label: t("players.tabStreaks"),  icon: Flame      },
    { key: "avggoals", label: t("players.tabAvgGoals"), icon: Swords     },
    { key: "power",    label: t("players.tabPower"),    icon: Zap        },
    { key: "activity", label: t("players.tabActivity"), icon: Activity   },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-zinc-500 mb-1">{t("players.subtitle")}</p>
          <h1 className="font-display text-2xl font-bold text-zinc-100">{t("players.title")}</h1>
          <Link href="/friendlies" className="mt-1.5 inline-flex items-center gap-1.5 rounded-full border border-orange-400/30 bg-orange-400/5 px-3 py-1 text-[11px] font-semibold text-orange-400 transition-colors hover:bg-orange-400/10">
            ⚔️ {t("friendlies.title")}
          </Link>
        </div>
        <div className="text-right">
          <p className="text-lg font-black text-yellow-400">{players.length}</p>
          <p className="text-[10px] uppercase text-zinc-600 tracking-wide">{t("common.players")}</p>
        </div>
      </div>

      {/* Tabs — горизонтальный скролл на мобиле */}
      <div className="overflow-x-auto -mx-4 px-4">
        <div className="flex items-center gap-1 border-b border-zinc-700 min-w-max">
          {TABS.map((tb) => {
            const Icon = tb.icon;
            return (
              <button key={tb.key} onClick={() => setTab(tb.key)}
                role="tab"
                aria-selected={tab === tb.key}
                className={cn(
                  "flex items-center gap-1.5 px-3 py-2.5 text-xs font-semibold transition-colors border-b-2 -mb-px whitespace-nowrap",
                  tab === tb.key ? "border-yellow-400 text-yellow-400" : "border-transparent text-zinc-400 hover:text-white"
                )}
              >
                <Icon size={13} aria-hidden="true" />
                {tb.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* ── ELO Rating ── */}
      {tab === "rating" && (
        loadingPlayers ? <SkeletonTable rows={10} /> :
        players.length === 0 ? <EmptyCard icon={Crown} text={t("players.noPlayersText")} /> : (
          <div className="rounded-xl card-premium overflow-hidden">
            <div className="grid grid-cols-[32px_1fr_70px] sm:grid-cols-[40px_1fr_80px_80px_60px_90px] gap-2 px-3 sm:px-4 py-2.5 border-b border-zinc-800 text-[10px] font-bold uppercase tracking-widest text-zinc-600">
              <div className="text-center">{t("players.colNum")}</div>
              <div>{t("players.colPlayer")}</div>
              <div className="text-center hidden sm:block">{t("players.colTier")}</div>
              <div className="text-center hidden sm:block">{t("players.colPower")}</div>
              <div className="text-center hidden sm:block">Win%</div>
              <div className="text-right">{t("players.colElo")}</div>
            </div>
            {players.map((player, index) => {
              const tier = tierColor(player.rating);
              const isMe = player.id === me;
              const tierLabel = player.rating >= 1200 ? t("players.tierGold")
                : player.rating >= 1100 ? t("players.tierBlue")
                : player.rating >= 1050 ? t("players.tierPurple")
                : t("players.tierGreen");
              const played = (player as any).wins + (player as any).draws + (player as any).losses;
              const winPct = played > 0 ? ((((player as any).wins ?? 0) / played) * 100).toFixed(0) + "%" : "—";
              return (
                <div key={player.id} className={cn(
                  "grid grid-cols-[32px_1fr_70px] sm:grid-cols-[40px_1fr_80px_80px_60px_90px] gap-2 px-3 sm:px-4 py-2.5 items-center",
                  "border-b border-zinc-800/60 last:border-0 transition-colors",
                  isMe ? "bg-yellow-500/5 border-l-2 border-l-yellow-500" : "hover:bg-zinc-800/30 active:bg-zinc-800/50"
                )}>
                  <div className="flex justify-center">
                    <Pos i={index} id={player.id} me={me} />
                  </div>
                  <div className="flex items-center gap-2.5 min-w-0">
                    <PlayerAvatar displayName={player.display_name} favoriteClub={player.favorite_club} size={32} online={presence.isOnline(player.id)} />
                    <div className="min-w-0">
                      <Link href={`/players/details?id=${player.id}`} className={cn("text-sm font-semibold truncate hover:underline", isMe ? "text-yellow-300" : "text-zinc-100")}>
                        <span className="sm:hidden">{player.display_name.split(" ")[0]}</span>
                        <span className="hidden sm:inline">{player.display_name}</span>
                        {isMe && <span className="ml-1.5 text-[9px] font-bold text-yellow-400 uppercase">{t("common.you")}</span>}
                      </Link>
                      <p className="text-[10px] text-zinc-600 truncate">{player.rank || t("common.rank")}</p>
                    </div>
                  </div>
                  <div className="hidden sm:flex justify-center">
                    <span className={cn("rounded-full border px-2 py-0.5 text-[9px] font-bold uppercase flex items-center gap-1", tier.badge)}>
                      <span className={cn("h-1.5 w-1.5 rounded-full flex-shrink-0", tier.dot)} />
                      {tierLabel}
                    </span>
                  </div>
                  <div className="hidden sm:flex justify-center">
                    <span className="text-xs text-zinc-400 font-medium">{(player.team_power || 0).toLocaleString()}</span>
                  </div>
                  <div className="hidden sm:flex justify-center">
                    <span className="text-xs text-zinc-400 font-medium tabular-nums">{winPct}</span>
                  </div>
                  <div className="text-right">
                    <p className={cn("text-base font-black tabular-nums", tier.text)}>{player.rating}</p>
                    <p className="text-[9px] text-zinc-400 uppercase">{t("common.elo")}</p>
                  </div>
                </div>
              );
            })}
          </div>
        )
      )}

      {/* ── Бомбардиры ── */}
      {tab === "scorers" && (
        <div className="space-y-4">
          {topScorers.length === 0 ? <EmptyCard icon={Target} text={t("players.noGoalsText")} /> :
          topScorers.map((league) => (
            <div key={league.league.id} className="rounded-xl card-premium overflow-hidden">
              <div className="flex items-center gap-2 px-4 py-3 border-b border-zinc-800 text-xs font-semibold uppercase tracking-wider text-zinc-400">
                <Trophy size={13} aria-hidden="true" /> {league.league.name}
              </div>
              {league.scorers.map((scorer, i) => (
                <div key={scorer.user_id} className="flex items-center gap-3 px-4 py-2.5 border-b border-zinc-800/50 last:border-0">
                  <span className="w-6 text-center flex-shrink-0 select-none" style={{ fontSize: i < 3 ? 18 : 13, lineHeight: 1 }}>
                    {i === 0 ? "🥇" : i === 1 ? "🥈" : i === 2 ? "🥉" : <span className="text-zinc-500 font-semibold">{i + 1}</span>}
                  </span>
                  <PlayerAvatar displayName={scorer.display_name} favoriteClub={scorer.favorite_club} size={36} />
                  <Link href={`/players/details?id=${scorer.user_id}`} className="flex-1 min-w-0 group">
                    <p className="text-sm font-semibold text-zinc-200 truncate group-hover:underline">{scorer.display_name}</p>
                    <p className="text-xs text-zinc-500 flex items-center gap-1"><Shirt size={10} /> {scorer.team_power || 0}</p>
                  </Link>
                  <Val top={scorer.goals_for} bot={t("players.goals")} color="text-yellow-400" />
                </div>
              ))}
            </div>
          ))}
        </div>
      )}

      {/* ── Win Rate ── */}
      {tab === "winrate" && (
        loadingWR ? <SkeletonTable rows={10} /> :
        winRate.length === 0 ? <EmptyCard icon={TrendingUp} text={t("players.need5Matches")} /> : (
          <div className="rounded-xl card-premium overflow-hidden">
            <div className="px-4 py-2.5 border-b border-zinc-800 text-[10px] font-bold uppercase tracking-widest text-zinc-600 flex gap-2">
              <span className="w-6" />
              <span className="flex-1">{t("players.colPlayer")} {t("players.colPlayerMin5Suffix")}</span>
              <span className="w-24 text-right">{t("standings.played")} / {t("standings.wins")} / {t("standings.draws")} / {t("standings.losses")}</span>
              <span className="w-16 text-right">Win%</span>
            </div>
            {winRate.map((e, i) => (
              <StatRow key={e.user_id} entry={e} i={i} me={me} right={
                <div className="flex items-center gap-3">
                  <span className="text-[10px] text-zinc-600 tabular-nums hidden sm:block">
                    {e.played} / {e.wins} / {e.draws} / {e.losses}
                  </span>
                  <Val top={`${e.win_rate.toFixed(1)}%`} bot="win rate" color={e.win_rate >= 60 ? "text-green-400" : e.win_rate >= 40 ? "text-yellow-400" : "text-red-400"} />
                </div>
              } />
            ))}
          </div>
        )
      )}

      {/* ── Серии побед ── */}
      {tab === "streaks" && (
        loadingStreaks ? <SkeletonTable rows={10} /> :
        streaks.length === 0 ? <EmptyCard icon={Flame} text={t("players.noStreaks")} /> : (
          <div className="rounded-xl card-premium overflow-hidden">
            <div className="px-4 py-2.5 border-b border-zinc-800 text-[10px] font-bold uppercase tracking-widest text-zinc-600">
              {t("players.hdrStreakNow")}
            </div>
            {streaks.map((e, i) => (
              <StatRow key={e.user_id} entry={e} i={i} me={me} right={
                <div className="flex items-center gap-1.5">
                  <Val top={<>{e.streak} <span className="text-sm">🔥</span></>} bot={t("misc.streakSuffix")} color="text-orange-400" />
                </div>
              } />
            ))}
          </div>
        )
      )}

      {/* ── Голы за матч ── */}
      {tab === "avggoals" && (
        loadingAvg ? <SkeletonTable rows={10} /> :
        avgGoals.length === 0 ? <EmptyCard icon={Swords} text={t("players.need5Matches")} /> : (
          <div className="rounded-xl card-premium overflow-hidden">
            <div className="px-4 py-2.5 border-b border-zinc-800 text-[10px] font-bold uppercase tracking-widest text-zinc-600 flex gap-2">
              <span className="w-6" />
              <span className="flex-1">{t("players.colPlayer")} {t("players.colPlayerMin5Suffix")}</span>
              <span className="w-24 text-right hidden sm:block">{t("players.hdrGoalsMatches")}</span>
              <span className="w-16 text-right">{t("players.hdrGPerMatch")}</span>
            </div>
            {avgGoals.map((e, i) => (
              <StatRow key={e.user_id} entry={e} i={i} me={me} right={
                <div className="flex items-center gap-3">
                  <span className="text-[10px] text-zinc-600 hidden sm:block tabular-nums">
                    {e.goals_for} {t("players.goalsShort")} / {e.played} {t("players.gamesShort")}
                  </span>
                  <Val top={e.avg_goals.toFixed(2)} bot={t("misc.goalsPerMatch")} color="text-emerald-400" />
                </div>
              } />
            ))}
          </div>
        )
      )}

      {/* ── Сила команды ── */}
      {tab === "power" && (
        loadingPower ? <SkeletonTable rows={10} /> :
        power.length === 0 ? <EmptyCard icon={Zap} text={t("misc.noPower")} /> : (
          <div className="rounded-xl card-premium overflow-hidden">
            <div className="px-4 py-2.5 border-b border-zinc-800 text-[10px] font-bold uppercase tracking-widest text-zinc-600">
              {t("players.hdrPowerRating")}
            </div>
            {power.map((e, i) => (
              <StatRow key={e.user_id} entry={e} i={i} me={me} right={
                <Val top={e.team_power.toLocaleString()} bot={t("misc.teamPowerLabel")} color="text-sky-400" />
              } />
            ))}
          </div>
        )
      )}

      {/* ── Активность ── */}
      {tab === "activity" && (
        loadingActivity ? <SkeletonTable rows={10} /> :
        activity.length === 0 ? <EmptyCard icon={Activity} text={t("misc.noActivity")} /> : (
          <div className="rounded-xl card-premium overflow-hidden">
            <div className="px-4 py-2.5 border-b border-zinc-800 text-[10px] font-bold uppercase tracking-widest text-zinc-600 flex gap-2">
              <span className="w-6" />
              <span className="flex-1">{t("players.colPlayer")}</span>
              <span className="w-20 text-right hidden sm:block">{t("profile.wins")}</span>
              <span className="w-16 text-right">{t("playerPage.matches")}</span>
            </div>
            {activity.map((e, i) => (
              <StatRow key={e.user_id} entry={e} i={i} me={me} right={
                <div className="flex items-center gap-3">
                  <span className="text-[10px] text-zinc-400 hidden sm:block">{e.wins} {t("players.winsShort")}</span>
                  <Val top={e.played} bot={t("misc.matchesLabel")} color="text-violet-400" />
                </div>
              } />
            ))}
          </div>
        )
      )}
    </div>
  );
}
