"use client";

import { Suspense, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { BarChart2, CalendarDays, GitBranch, History, Info, ListOrdered, Trophy, Users } from "lucide-react";
import { LeagueInfoPanel } from "@/components/LeagueInfoPanel";
import { TournamentTree } from "@/components/TournamentTree";
import { GroupStageView } from "@/components/GroupStageView";
import { EmptyState } from "@/components/EmptyState";
import { LeagueStandings } from "@/components/LeagueStandings";
import { MatchCard } from "@/components/MatchCard";
import { LeagueStatusBadge } from "@/components/StatusBadge";
import { SkeletonTable } from "@/components/ui/skeleton";
import { fetchBracket, fetchLeague, fetchMyHistory, fetchMyMatches, fetchSchedule, fetchStandings } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useLeagueSSE } from "@/lib/sse";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

type Tab = "info" | "bracket" | "table" | "schedule" | "groups" | "my" | "history";

function LeagueDetails() {
  const searchParams = useSearchParams();
  const id = Number(searchParams.get("id"));
  const { user } = useAuth();
  const { t } = useLang();
  const [tab, setTab] = useState<Tab>("table");

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
    if (league.status === "registration") setTab("info");
    setTabInitialized(true);
  }

  if (!id || leagueError) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-900">
        <EmptyState icon={Trophy} title={t("leagueDetail.notFound")} text={t("leagueDetail.notFoundText")} />
      </div>
    );
  }

  const refreshAll = () => { refetchSchedule(); refetchMyMatches(); refetchHistory(); };

  // SSE — live обновления без F5
  useLeagueSSE(id || 0, refreshAll);

  const isGroupsFormat =
    league?.rounds_type === "groups" || league?.rounds_type === "groups_playoff";

  const allTabs = [
    { key: "info",    icon: Info,         label: t("leagueDetail.tabInfo") },
    { key: "bracket", icon: GitBranch,    label: "Сетка" },   // всегда видна
    ...(isGroupsFormat
      ? [{ key: "groups", icon: Users, label: "Группы" }]
      : [{ key: "table",  icon: ListOrdered, label: t("leagueDetail.tabTable") }]
    ),
    { key: "schedule", icon: CalendarDays, label: t("leagueDetail.tabSchedule") },
    ...(user ? [
      { key: "my",      icon: Users,   label: t("leagueDetail.tabMy") },
      { key: "history", icon: History, label: t("leagueDetail.tabHistory") },
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
          <p className="text-xs font-semibold uppercase tracking-widest text-zinc-500 mb-0.5">{t("leagueDetail.leagueLabel")}</p>
          <h1 className="text-xl font-bold text-zinc-100 truncate">{league?.name || t("leagueDetail.leagueLabel")}</h1>
          <p className="text-xs text-zinc-500">
            {league?.rounds_type === "double" ? t("common.doubleRound") : t("common.singleRound")}
            {" · "}{league?.max_players ?? "—"} {t("common.players")}
          </p>
        </div>
        {league && <LeagueStatusBadge status={league.status} />}
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1 border-b border-zinc-700 pb-0">
        {allTabs.map((item) => {
          const Icon = item.icon;
          return (
            <button
              key={item.key}
              onClick={() => setTab(item.key as Tab)}
              className={cn(
                "flex items-center gap-1.5 px-3 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px",
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

      {/* Schedule */}
      {tab === "schedule" && (
        <div className="space-y-3">
          {rounds.length > 0 && user && (
            <div className="flex items-start gap-2 rounded-xl border border-zinc-700/50 bg-zinc-800/40 px-4 py-3">
              <Info size={14} className="text-zinc-400 flex-shrink-0 mt-0.5" />
              <p className="text-xs text-zinc-400 leading-relaxed">{t("matchCard.scheduleHint")}</p>
            </div>
          )}
          {rounds.length === 0 ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900">
              <EmptyState icon={CalendarDays} title={t("leagueDetail.scheduleEmpty")} text={t("leagueDetail.scheduleEmptyText")} />
            </div>
          ) : (
            rounds.map((round) => (
              <div key={round.round} className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
                <div className="flex items-center justify-between px-4 py-2.5 border-b border-zinc-800">
                  <span className="text-sm font-semibold text-zinc-300">{round.round} {t("leagueDetail.roundLabel")}</span>
                  <span className="text-xs text-zinc-600">
                    {round.matches.length} {t("leagueDetail.matchLabel")}
                  </span>
                </div>
                {round.matches.map((match) => (
                  <MatchCard key={match.id} match={match} compact onUpdate={refreshAll} />
                ))}
              </div>
            ))
          )}
        </div>
      )}

      {/* My matches */}
      {tab === "my" && (
        <div className="space-y-3">
          {myMatches.length === 0 ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900">
              <EmptyState icon={Users} title={t("leagueDetail.noActiveMatches")} text={t("leagueDetail.noActiveMatchesText")} />
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {myMatches.map((match) => (
                <MatchCard key={match.id} match={match} onUpdate={refreshAll} />
              ))}
            </div>
          )}
        </div>
      )}

      {/* Bracket */}
      {tab === "bracket" && league && (
        <TournamentTree
          league={league}
          standings={standings}
          bracketStages={bracketStages}
          currentUserId={user?.id}
        />
      )}

      {/* History */}
      {tab === "history" && (
        <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
          {history.length === 0 ? (
            <EmptyState icon={History} title={t("leagueDetail.historyEmpty")} text={t("leagueDetail.historyEmptyText")} />
          ) : (
            <div>
              {history.map((match) => (
                <div key={match.id} className="flex items-center gap-4 px-4 py-3 border-b border-zinc-800/50 last:border-0">
                  <span className="text-xs text-zinc-600 flex-shrink-0">{t("leagueDetail.round")} {match.round}</span>
                  <span className="flex-1 text-sm font-semibold text-zinc-200 truncate">
                    {match.home_name}{" "}
                    <span className="text-yellow-400">{match.home_goals}:{match.away_goals}</span>
                    {" "}{match.away_name}
                  </span>
                  <span className="text-xs text-zinc-600 flex-shrink-0">
                    {match.played_at ? new Date(match.played_at).toLocaleDateString("ru-RU") : t("leagueDetail.confirmedLabel")}
                  </span>
                </div>
              ))}
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
