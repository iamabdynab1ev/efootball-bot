"use client";

import { Suspense, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { BarChart2, CalendarDays, History, Info, ListOrdered, Trophy, Users } from "lucide-react";
import { EmptyState } from "@/components/EmptyState";
import { LeagueStandings } from "@/components/LeagueStandings";
import { MatchCard } from "@/components/MatchCard";
import { LeagueStatusBadge } from "@/components/StatusBadge";
import { SkeletonTable } from "@/components/ui/skeleton";
import { fetchLeague, fetchMyHistory, fetchMyMatches, fetchSchedule, fetchStandings } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

type Tab = "info" | "table" | "schedule" | "my" | "history";

function LeagueDetails() {
  const searchParams = useSearchParams();
  const id = Number(searchParams.get("id"));
  const { user } = useAuth();
  const { t } = useLang();
  const [tab, setTab] = useState<Tab>("table");

  const { data: league } = useQuery({ queryKey: ["league", id], queryFn: () => fetchLeague(id), enabled: !!id });
  const { data: standings = [] } = useQuery({ queryKey: ["standings", id], queryFn: () => fetchStandings(id), enabled: !!id });
  const { data: rounds = [], refetch: refetchSchedule } = useQuery({
    queryKey: ["schedule", id],
    queryFn: () => fetchSchedule(id),
    enabled: !!id,
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

  if (!id) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-900">
        <EmptyState icon={Trophy} title={t("leagueDetail.notFound")} text={t("leagueDetail.notFoundText")} />
      </div>
    );
  }

  const refreshAll = () => { refetchSchedule(); refetchMyMatches(); refetchHistory(); };

  const allTabs = [
    { key: "info",     icon: Info,         label: t("leagueDetail.tabInfo") },
    { key: "table",    icon: ListOrdered,  label: t("leagueDetail.tabTable") },
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
        <div className="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-yellow-500/10 text-yellow-500">
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
      <div className="flex items-center gap-1 border-b border-zinc-800 pb-0">
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
                  : "border-transparent text-zinc-400 hover:text-zinc-200"
              )}
            >
              <Icon size={14} />
              {item.label}
            </button>
          );
        })}
      </div>

      {/* Info */}
      {tab === "info" && (
        <div className="rounded-xl border border-zinc-800 bg-zinc-900 divide-y divide-zinc-800">
          {[
            { icon: BarChart2,   label: t("leagueDetail.format"),       value: league?.rounds_type === "double" ? t("common.doubleRound") : t("common.singleRound") },
            { icon: Users,       label: t("leagueDetail.maxPlayersLabel"), value: `${league?.max_players ?? "—"} ${t("common.players")}` },
            { icon: ListOrdered, label: t("leagueDetail.participants"),  value: String(standings.length) },
          ].map((row) => (
            <div key={row.label} className="flex items-center gap-3 px-4 py-3">
              <row.icon size={16} className="text-zinc-600 flex-shrink-0" />
              <span className="flex-1 text-sm text-zinc-400">{row.label}</span>
              <span className="text-sm font-semibold text-zinc-200">{row.value}</span>
            </div>
          ))}
          <div className="flex items-center gap-3 px-4 py-3">
            <Trophy size={16} className="text-zinc-600 flex-shrink-0" />
            <span className="flex-1 text-sm text-zinc-400">{t("leagueDetail.statusLabel")}</span>
            {league && <LeagueStatusBadge status={league.status} />}
          </div>
        </div>
      )}

      {/* Standings */}
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
