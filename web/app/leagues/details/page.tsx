"use client";

import { Suspense, useCallback, useEffect, useRef, useState, lazy } from "react";
import { useSearchParams } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { BarChart2, CalendarDays, GitBranch, History, Info, ListOrdered, MessageSquare, Trophy, Users } from "lucide-react";
import { EmptyState } from "@/components/EmptyState";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { LeagueStatusBadge } from "@/components/StatusBadge";
import { LiveIndicator } from "@/components/LiveIndicator";
import { ChampionCelebration } from "@/components/ChampionCelebration";

// Тяжёлые компоненты — загружаем только когда нужны
const LeagueInfoPanel  = lazy(() => import("@/components/LeagueInfoPanel").then(m => ({ default: m.LeagueInfoPanel })));
const TournamentTree   = lazy(() => import("@/components/TournamentTree").then(m => ({ default: m.TournamentTree })));
const DoubleElimView   = lazy(() => import("@/components/DoubleElimView").then(m => ({ default: m.DoubleElimView })));
const GroupStageView   = lazy(() => import("@/components/GroupStageView").then(m => ({ default: m.GroupStageView })));
const LeagueStandings  = lazy(() => import("@/components/LeagueStandings").then(m => ({ default: m.LeagueStandings })));
const MatchCard        = lazy(() => import("@/components/MatchCard").then(m => ({ default: m.MatchCard })));
const ChatPanel        = lazy(() => import("@/components/ChatPanel").then(m => ({ default: m.ChatPanel })));
import { SkeletonBracket, SkeletonTable } from "@/components/ui/skeleton";
import { fetchBracket, fetchLeague, fetchMyHistory, fetchMyMatches, fetchSchedule, fetchStandings, isPlayoffMatch, leagueFormatKeys, stageLabelKey } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useLeagueSSE } from "@/lib/sse";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

type Tab = "info" | "bracket" | "table" | "schedule" | "groups" | "my" | "history" | "chat";

function MatchRow({ match, me, flash }: { match: import("@/lib/api").Match; me?: number; flash?: boolean }) {
  const { t } = useLang();
  const isHomeMe  = match.home_user_id === me;
  const isAwayMe  = match.away_user_id === me;
  const confirmed = match.status === "confirmed";
  const isLive    = match.status === "pending_confirm";
  const homeWon   = confirmed && (match.home_goals ?? 0) > (match.away_goals ?? 0);
  const awayWon   = confirmed && (match.away_goals ?? 0) > (match.home_goals ?? 0);
  const meWon     = (isHomeMe && homeWon) || (isAwayMe && awayWon);
  const meLost    = (isHomeMe && awayWon) || (isAwayMe && homeWon);

  const statusMark = confirmed
    ? { text: meWon ? "В" : meLost ? "П" : "Н", cls: meWon ? "text-green-400" : meLost ? "text-red-400" : "text-zinc-500" }
    : match.status === "pending_confirm" ? { text: "⏳", cls: "text-yellow-500" }
    : match.status === "disputed"        ? { text: "⚠",  cls: "text-red-400"   }
    : { text: "•", cls: "text-zinc-500" };

  const sk = stageLabelKey(match.stage);
  const roundLabel = isPlayoffMatch(match) && sk
    ? (t(`leagueDetail.${sk}` as any) as string)
    : `${t("leagueDetail.groupRound")} ${match.round}`;

  return (
    <div className={cn(
      "flex items-center gap-1.5 px-2 sm:px-3 py-2.5 border-b border-zinc-800/40 last:border-0",
      flash && "row-flash",
    )}>
      {/* Статус */}
      <span className={cn("w-4 text-[10px] font-black flex-shrink-0 text-center", statusMark.cls)}>
        {statusMark.text}
      </span>

      {/* Хозяин */}
      <div className="flex items-center gap-1.5 flex-1 min-w-0 justify-end">
        <div className="text-right min-w-0">
          <p className={cn("text-[11px] font-semibold truncate max-w-[80px] sm:max-w-[120px]",
            isHomeMe ? "text-yellow-300" : "text-zinc-200")}>
            {match.home_name}
          </p>
          <p className="text-[9px] text-zinc-400">{t("leagueDetail.home")}</p>
        </div>
        <PlayerAvatar displayName={match.home_name} favoriteClub={match.home_club} size={24} />
      </div>

      {/* Счёт + тур */}
      <div className="flex-shrink-0 w-12 sm:w-14 text-center">
        {confirmed ? (
          <p className="text-sm font-black tabular-nums text-zinc-100">{match.home_goals}:{match.away_goals}</p>
        ) : typeof match.claimed_home === "number" ? (
          <p className="text-xs font-black tabular-nums text-amber-400">{match.claimed_home}:{match.claimed_away}</p>
        ) : (
          <p className="text-[10px] text-zinc-500">vs</p>
        )}
        <p className="text-[8px] text-zinc-500 leading-tight mt-0.5 truncate">{roundLabel}</p>
        {isLive && (
          <span className="mt-0.5 inline-block rounded-full bg-cyan-400/10 px-1.5 text-[8px] font-bold tracking-widest text-cyan-300">
            LIVE
          </span>
        )}
      </div>

      {/* Гость */}
      <div className="flex items-center gap-1.5 flex-1 min-w-0">
        <PlayerAvatar displayName={match.away_name} favoriteClub={match.away_club} size={24} />
        <div className="min-w-0">
          <p className={cn("text-[11px] font-semibold truncate max-w-[80px] sm:max-w-[120px]",
            isAwayMe ? "text-yellow-300" : "text-zinc-200")}>
            {match.away_name}
          </p>
          <p className="text-[9px] text-zinc-400">{t("leagueDetail.away")}</p>
        </div>
      </div>
    </div>
  );
}

function FilterBar({ filters, active, onChange }: {
  filters: { key: string; label: string }[];
  active: string;
  onChange: (key: string) => void;
}) {
  if (filters.length <= 1) return null;
  return (
    <div className="flex gap-1.5 flex-wrap">
      {filters.map(f => (
        <button
          key={f.key}
          onClick={() => onChange(f.key)}
          className={cn(
            "px-3 py-1.5 rounded-full text-xs font-semibold transition-colors border",
            active === f.key
              ? f.key === "playoff"
                ? "bg-amber-500/15 border-amber-500/40 text-amber-400"
                : "bg-yellow-500/15 border-yellow-500/40 text-yellow-400"
              : "bg-zinc-800/60 border-zinc-700 text-zinc-400 hover:text-white"
          )}
        >
          {f.label}
        </button>
      ))}
    </div>
  );
}

function LeagueDetails() {
  const searchParams = useSearchParams();
  const id = Number(searchParams.get("id"));
  const urlTab = searchParams.get("tab") as Tab | null;
  const { user } = useAuth();
  const { t } = useLang();
  const [tab, setTab] = useState<Tab>(urlTab ?? "table");
  const [scheduleFilter, setScheduleFilter] = useState<string>("all");
  const [myFilter, setMyFilter] = useState<string>("all");

  const { data: league, isError: leagueError } = useQuery({ queryKey: ["league", id], queryFn: () => fetchLeague(id), enabled: !!id });
  const { data: standings = [] } = useQuery({ queryKey: ["standings", id], queryFn: () => fetchStandings(id), enabled: !!id, staleTime: 30000 });
  const { data: rounds = [], refetch: refetchSchedule } = useQuery({
    queryKey: ["schedule", id],
    queryFn: () => fetchSchedule(id),
    enabled: !!id,
    staleTime: 30000,
  });
  const { data: myMatches = [], refetch: refetchMyMatches } = useQuery({
    queryKey: ["my-matches", id],
    queryFn: () => fetchMyMatches(id),
    enabled: !!user && !!id,
  });
  const { data: history = [], refetch: refetchHistory } = useQuery({
    queryKey: ["me", "history", id],
    queryFn: () => fetchMyHistory(id),
    enabled: !!user && !!id,
  });
  const { data: bracketStages = [] } = useQuery({
    queryKey: ["bracket", id],
    queryFn: () => fetchBracket(id),
    enabled: !!id,
    refetchInterval: tab === "bracket" ? 15000 : false,
  });

  // Умный дефолтный таб: при регистрации — "info", при активной с матчами у юзера — "my"
  const [tabInitialized, setTabInitialized] = useState(false);
  if (!tabInitialized && league) {
    if (!urlTab && league.status === "registration") setTab("info");
    setTabInitialized(true);
  }

  const qc = useQueryClient();
  // Живое табло: id матча, изменившегося по SSE — его строка вспыхивает.
  const [flashId, setFlashId] = useState(0);
  const flashTimer = useRef<ReturnType<typeof setTimeout>>();

  // Обновляет ВСЕ данные лиги: раньше по SSE обновлялись только
  // расписание/мои матчи/история, а таблица, сетка и группы — нет.
  const refreshAll = useCallback((matchId?: number) => {
    refetchSchedule();
    refetchMyMatches();
    refetchHistory();
    qc.invalidateQueries({ queryKey: ["standings", id] });
    qc.invalidateQueries({ queryKey: ["bracket", id] });
    qc.invalidateQueries({ queryKey: ["group-data", id] });
    qc.invalidateQueries({ queryKey: ["league", id] });
    if (matchId) {
      setFlashId(matchId);
      if (flashTimer.current) clearTimeout(flashTimer.current);
      flashTimer.current = setTimeout(() => setFlashId(0), 2600);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, qc]);

  // SSE — live обновления без F5 (хук должен вызываться до любых early return)
  const { live } = useLeagueSSE(id || 0, refreshAll);

  // Церемония чемпиона: автопоказ один раз после завершения финала.
  const [celebrate, setCelebrate] = useState(false);
  const finalSlot = bracketStages.find((s) => s.stage === "final")?.slots[0];
  const championName = finalSlot?.winner_name;
  useEffect(() => {
    if (!id || !championName) return;
    const seenKey = `champion_seen_${id}`;
    if (typeof window !== "undefined" && !localStorage.getItem(seenKey)) {
      localStorage.setItem(seenKey, "1");
      setCelebrate(true);
    }
  }, [id, championName]);

  if (!id || leagueError) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-900">
        <EmptyState icon={Trophy} title={t("leagueDetail.notFound")} text={t("leagueDetail.notFoundText")} />
      </div>
    );
  }

  const isGroupsFormat =
    league?.rounds_type === "groups" || league?.rounds_type === "groups_playoff";

  const allTabs = [
    { key: "info",    icon: Info,         label: t("leagueDetail.tabInfo") },
    { key: "bracket", icon: GitBranch,    label: t("leagueDetail.tabBracket") },   // всегда видна
    ...(isGroupsFormat
      ? [{ key: "groups", icon: Users, label: t("leagueDetail.tabGroups") }]
      : [{ key: "table",  icon: ListOrdered, label: t("leagueDetail.tabTable") }]
    ),
    { key: "schedule", icon: CalendarDays, label: t("leagueDetail.tabSchedule") },
    ...(user ? [
      { key: "my",      icon: Users,         label: t("leagueDetail.tabMy") },
      { key: "history", icon: History,       label: t("leagueDetail.tabHistory") },
      { key: "chat",    icon: MessageSquare, label: "Чат" },
    ] : []),
  ] as const;

  return (
    <div className="space-y-5">
      {/* League header */}
      <div className="flex items-center gap-4">
        <div className="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-yellow-500/10 text-yellow-400">
          <Trophy size={22} />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-xs font-semibold uppercase tracking-widest text-zinc-400 mb-0.5">{t("leagueDetail.leagueLabel")}</p>
          <h1 className="font-display text-xl font-bold text-zinc-100 truncate min-w-0">{league?.name || t("leagueDetail.leagueLabel")}</h1>
          <p className="text-xs text-zinc-400">
            {t(leagueFormatKeys(league?.rounds_type).label as never)}
            {" · "}{league?.players_count ?? "—"} {t("common.players")}
          </p>
        </div>
        <div className="flex items-center gap-2 flex-shrink-0">
          <LiveIndicator live={live && league?.status === "active"} />
          {league && <LeagueStatusBadge status={league.status} />}
        </div>
      </div>

      {/* Tabs */}
      <div role="tablist" className="flex items-center gap-1 border-b border-zinc-700 pb-0 overflow-x-auto -mx-4 px-4 scrollbar-none">
        {allTabs.map((item) => {
          const Icon = item.icon;
          return (
            <button
              key={item.key}
              onClick={() => setTab(item.key as Tab)}
              role="tab"
              aria-selected={tab === item.key}
              className={cn(
                "flex items-center gap-1.5 px-3 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-yellow-400/60 rounded-t",
                tab === item.key
                  ? "border-yellow-400 text-yellow-400"
                  : "border-transparent text-zinc-300 hover:text-white"
              )}
            >
              <Icon size={14} />
              {item.label}
            </button>
          );
        })}
      </div>

      {/* Info */}
      {tab === "info" && league && (
        <LeagueInfoPanel
          league={league}
          standings={standings}
          hasPlayoff={bracketStages.length > 0}
        />
      )}

      {/* Групповая стадия */}
      {tab === "groups" && (
        <GroupStageView
          leagueId={id}
          currentUserId={user?.id}
          onUpdate={refreshAll}
          advance={league?.group_advance ?? 2}
          bracketStages={bracketStages}
        />
      )}

      {/* Чат турнира */}
      {tab === "chat" && (
        <ChatPanel leagueId={id} currentUserId={user?.id} isAdmin={user?.is_admin} />
      )}

      {/* Standings (только для обычных лиг) */}
      {tab === "table" && (
        <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
          {standings.length === 0 ? (
            <EmptyState icon={ListOrdered} title={t("leagueDetail.standingsEmpty")} text={t("leagueDetail.standingsEmptyText")} />
          ) : (
            <LeagueStandings standings={standings} currentUserId={user?.id} />
          )}
        </div>
      )}

      {/* Schedule — все матчи лиги с фильтрами */}
      {tab === "schedule" && (() => {
        const allMatches   = rounds.flatMap(r => r.matches);
        const groupLabels  = Array.from(new Set(
          allMatches.filter(m => !isPlayoffMatch(m) && m.stage?.length === 1).map(m => m.stage!)
        )).sort();
        const hasPlayoff   = allMatches.some(m => isPlayoffMatch(m));
        const filters      = [
          { key: "all",     label: t("leagueDetail.allMatches") },
          ...groupLabels.map(g => ({ key: g, label: `${t("leagueDetail.groupTitle")} ${g}` })),
          ...(hasPlayoff ? [{ key: "playoff", label: t("leagueDetail.filterPlayoff") }] : []),
        ];
        const visibleRounds = rounds.map(r => ({
          ...r,
          matches: r.matches.filter(m =>
            scheduleFilter === "all"     ? true :
            scheduleFilter === "playoff" ? isPlayoffMatch(m) :
            m.stage === scheduleFilter
          ),
        })).filter(r => r.matches.length > 0);

        return (
          <div className="space-y-3">
            <FilterBar filters={filters} active={scheduleFilter} onChange={setScheduleFilter} />
            {visibleRounds.length === 0
              ? <div className="rounded-xl border border-zinc-800 bg-zinc-900">
                  <EmptyState icon={CalendarDays} title={t("leagueDetail.scheduleEmpty")} text={t("leagueDetail.scheduleEmptyText")} />
                </div>
              : visibleRounds.map(round => {
                  const po = isPlayoffMatch(round.matches[0]);
                  return (
                    <div key={round.round} className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
                      <div className="flex items-center justify-between px-4 py-2.5 border-b border-zinc-800">
                        <span className={cn("text-sm font-semibold", po ? "text-amber-400" : "text-zinc-300")}>
                          {po
                            ? (t(`leagueDetail.${stageLabelKey(round.matches[0]?.stage) ?? "filterPlayoff"}` as any) as string)
                            : `${round.round} ${t("leagueDetail.roundLabel")}`}
                        </span>
                        <span className="text-xs text-zinc-500">{round.matches.length} {t("leagueDetail.matchLabel")}</span>
                      </div>
                      {round.matches.map(m => <MatchRow key={m.id} match={m} me={user?.id} flash={m.id === flashId} />)}
                    </div>
                  );
                })
            }
          </div>
        );
      })()}

      {/* My matches — ВСЕ матчи пользователя в этой лиге */}
      {tab === "my" && (() => {
        // Берём все матчи лиги и фильтруем по текущему пользователю
        const allUserMatches = rounds
          .flatMap(r => r.matches)
          .filter(m => m.home_user_id === user?.id || m.away_user_id === user?.id);

        const groupLabels = Array.from(new Set(
          allUserMatches.filter(m => !isPlayoffMatch(m) && m.stage?.length === 1).map(m => m.stage!)
        )).sort();
        const hasPlayoff = allUserMatches.some(m => isPlayoffMatch(m));
        const filters = [
          { key: "all",     label: t("leagueDetail.allMatches") },
          ...groupLabels.map(g => ({ key: g, label: `${t("leagueDetail.groupTitle")} ${g}` })),
          ...(hasPlayoff ? [{ key: "playoff", label: t("leagueDetail.filterPlayoff") }] : []),
        ];
        const shown = allUserMatches.filter(m =>
          myFilter === "all"     ? true :
          myFilter === "playoff" ? isPlayoffMatch(m) :
          m.stage === myFilter
        );


        return (
          <div className="space-y-3">
            <FilterBar filters={filters} active={myFilter} onChange={setMyFilter} />
            {shown.length === 0
              ? <div className="rounded-xl border border-zinc-800 bg-zinc-900">
                  <EmptyState icon={Users} title={t("leagueDetail.noActiveMatches")} text={t("leagueDetail.matchesNotFound")} />
                </div>
              : <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
                  {shown.map(m => <MatchRow key={m.id} match={m} me={user?.id} flash={m.id === flashId} />)}
                </div>
            }
          </div>
        );
      })()}

      {/* Bracket */}
      {tab === "bracket" && league && (
        <Suspense fallback={<SkeletonBracket />}>
          {league.rounds_type === "double_elim" ? (
            <DoubleElimView leagueId={id} currentUserId={user?.id} />
          ) : (
            <TournamentTree
              league={league}
              standings={standings}
              bracketStages={bracketStages}
              currentUserId={user?.id}
              onCelebrate={() => setCelebrate(true)}
            />
          )}
        </Suspense>
      )}

      {/* Церемония чемпиона */}
      {championName && (
        <ChampionCelebration
          open={celebrate}
          onClose={() => setCelebrate(false)}
          championName={championName}
          championClub={finalSlot?.winner_club}
          leagueName={league?.name}
          finalScore={
            typeof finalSlot?.home_goals === "number" && typeof finalSlot?.away_goals === "number"
              ? `${finalSlot.home_goals}:${finalSlot.away_goals}`
              : undefined
          }
        />
      )}

      {/* History */}
      {tab === "history" && (
        <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
          {history.length === 0 ? (
            <EmptyState icon={History} title={t("leagueDetail.historyEmpty")} text={t("leagueDetail.historyEmptyText")} />
          ) : (
            <div>
              {history.map((match) => {
                const isHomeMe = match.home_user_id === user?.id;
                const isAwayMe = match.away_user_id === user?.id;
                const homeWon  = (match.home_goals ?? 0) > (match.away_goals ?? 0);
                const awayWon  = (match.away_goals ?? 0) > (match.home_goals ?? 0);
                const meWon    = (isHomeMe && homeWon) || (isAwayMe && awayWon);
                const meLost   = (isHomeMe && awayWon) || (isAwayMe && homeWon);
                return (
                  <div key={match.id} className="flex items-center gap-3 px-4 py-3 border-b border-zinc-800/40 last:border-0">
                    {/* Результат для текущего пользователя */}
                    <span className={cn(
                      "w-5 text-[10px] font-black flex-shrink-0 text-center",
                      meWon  ? "text-green-400" :
                      meLost ? "text-red-400"   : "text-zinc-500"
                    )}>
                      {meWon ? "В" : meLost ? "П" : "Н"}
                    </span>

                    {/* Хозяин */}
                    <div className="flex items-center gap-1.5 flex-1 min-w-0 justify-end">
                      <span className={cn("text-[11px] truncate max-w-[70px] sm:max-w-[100px] text-right",
                        isHomeMe ? "text-yellow-300 font-semibold" : "text-zinc-300"
                      )}>{match.home_name}</span>
                      <PlayerAvatar displayName={match.home_name} favoriteClub={match.home_club} size={22} />
                    </div>

                    {/* Счёт */}
                    <div className="flex-shrink-0 w-14 text-center">
                      <span className={cn("text-sm font-black tabular-nums",
                        homeWon || awayWon ? "text-zinc-100" : "text-zinc-400"
                      )}>
                        {match.home_goals} : {match.away_goals}
                      </span>
                    </div>

                    {/* Гость */}
                    <div className={cn("flex items-center gap-1.5 flex-1 min-w-0")}>
                      <PlayerAvatar displayName={match.away_name} favoriteClub={match.away_club} size={24} />
                      <span className={cn("text-xs truncate max-w-[90px]",
                        isAwayMe ? "text-yellow-300 font-semibold" : "text-zinc-300"
                      )}>{match.away_name}</span>
                    </div>

                    {/* Дата */}
                    <span className="text-[10px] text-zinc-600 flex-shrink-0 hidden sm:block">
                      {match.played_at ? new Date(match.played_at).toLocaleDateString("ru-RU") : ""}
                    </span>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default function LeagueDetailsPage() {
  return (
    <Suspense fallback={
      <div className="rounded-xl border border-zinc-800 bg-zinc-900">
        <SkeletonTable rows={5} />
      </div>
    }>
      <LeagueDetails />
    </Suspense>
  );
}
