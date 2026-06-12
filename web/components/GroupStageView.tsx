"use client";

import { useState } from "react";
import { useQueries } from "@tanstack/react-query";
import { ArrowRight, Trophy } from "lucide-react";
import { api } from "@/lib/api";
import { CountUp } from "@/components/CountUp";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { cn } from "@/lib/utils";

interface Standing {
  user_id: number;
  display_name: string;
  favorite_club?: string;
  points: number;
  wins: number;
  draws: number;
  losses: number;
  goals_for: number;
  goals_against: number;
  goal_diff: number;
  position?: number;
}

interface Props {
  leagueId: number;
  currentUserId?: number;
  onUpdate: () => void;
  advance: number;
  bracketStages: any[];
}

interface GroupData {
  standings: Standing[];
}

async function fetchGroupData(leagueId: number, group: string): Promise<GroupData> {
  const r = await api.get(`/api/leagues/${leagueId}/groups/${group}/standings`);
  return { standings: r.data.standings ?? [] };
}

async function fetchGroupNames(leagueId: number): Promise<string[]> {
  const r = await api.get(`/api/leagues/${leagueId}/groups`);
  return Object.keys(r.data.groups ?? {}).sort();
}

/* ── Содержимое одной группы ─────────────────────────────── */
function GroupContent({ standings, advance, me }: {
  standings: Standing[];
  advance: number;
  me?: number;
}) {

  return (
    <div className="space-y-5">

      {/* Таблица группы */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
        <div className="flex items-center justify-between px-4 py-2.5 border-b border-zinc-800">
          <span className="text-xs font-bold uppercase tracking-widest text-zinc-500">Таблица</span>
        </div>
        <div className="overflow-x-auto">
        <table className="w-full min-w-[320px]">
          <thead>
            <tr className="border-b border-zinc-800/50 text-[10px] font-bold uppercase tracking-widest text-zinc-600">
              <th className="w-7 py-2 text-center">#</th>
              <th className="py-2 text-left pl-2">Клуб</th>
              <th className="w-8 py-2 text-center">И</th>
              <th className="w-8 py-2 text-center text-green-600">В</th>
              <th className="w-8 py-2 text-center">Н</th>
              <th className="w-8 py-2 text-center text-red-600">П</th>
              <th className="w-10 py-2 text-center hidden sm:table-cell">ГР</th>
              <th className="w-9 py-2 text-right pr-3">О</th>
            </tr>
          </thead>
          <tbody>
            {standings.map((row, i) => {
              const pos      = row.position ?? i + 1;
              const advances = i < advance;
              const isMe     = row.user_id === me;
              const played   = row.wins + row.draws + row.losses;
              return (
                <tr key={row.user_id} className={cn(
                  advances ? "bg-emerald-500/5" : "",
                  isMe     ? "bg-yellow-500/5"  : "",
                  i === advance - 1 ? "border-b-2 border-emerald-500/30" : "border-b border-zinc-800/30",
                  "last:border-b-0"
                )}>
                  <td className="py-2.5 text-center">
                    <span className={cn("text-xs font-bold",
                      pos === 1 ? "text-yellow-400" : pos === 2 ? "text-zinc-300" : "text-zinc-600"
                    )}>{pos}</span>
                  </td>
                  <td className="py-2.5 pl-2">
                    <div className="flex items-center gap-2">
                      <PlayerAvatar displayName={row.display_name} favoriteClub={row.favorite_club} size={24} />
                      <span className={cn("text-xs font-semibold truncate max-w-[80px] sm:max-w-[110px]",
                        isMe ? "text-yellow-300" : advances ? "text-zinc-100" : "text-zinc-500"
                      )}>
                        <span className="sm:hidden">{row.display_name.split(" ")[0]}</span>
                        <span className="hidden sm:inline">{row.display_name}</span>
                      </span>
                      {advances && <span className="text-[8px] font-black text-emerald-400 flex-shrink-0">↑ ПО</span>}
                    </div>
                  </td>
                  <td className="py-2.5 text-center text-xs text-zinc-500">{played}</td>
                  <td className="py-2.5 text-center text-xs font-semibold text-green-400">{row.wins}</td>
                  <td className="py-2.5 text-center text-xs text-zinc-500">{row.draws}</td>
                  <td className="py-2.5 text-center text-xs text-red-400">{row.losses}</td>
                  <td className="py-2.5 text-center text-xs hidden sm:table-cell">
                    <span className={cn("font-semibold",
                      row.goal_diff > 0 ? "text-green-400" : row.goal_diff < 0 ? "text-red-400" : "text-zinc-600"
                    )}>{row.goal_diff > 0 ? `+${row.goal_diff}` : row.goal_diff}</span>
                  </td>
                  <td className="py-2.5 text-right pr-3">
                    <CountUp value={row.points} className="text-sm font-black text-yellow-400 tabular-nums" />
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
        </div>
      </div>

    </div>
  );
}

/* ── Главный компонент ───────────────────────────────────── */
export function GroupStageView({ leagueId, currentUserId, advance }: Props) {
  const [activeGroup, setActiveGroup] = useState<string | null>(null);

  const [namesQ] = useQueries({
    queries: [{
      queryKey: ["group-names", leagueId],
      queryFn:  () => fetchGroupNames(leagueId),
      staleTime: 30000,
    }],
  });

  const groupNames: string[] = namesQ.data ?? [];
  const selected = activeGroup ?? groupNames[0] ?? null;

  const groupQueries = useQueries({
    queries: groupNames.map(g => ({
      queryKey: ["group-data", leagueId, g],
      queryFn:  () => fetchGroupData(leagueId, g),
      staleTime: 30000,
      enabled:  groupNames.length > 0,
    })),
  });

  if (namesQ.isLoading) {
    return <div className="py-8 text-center text-zinc-500 text-sm">Загрузка групп...</div>;
  }
  if (groupNames.length === 0) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-4 py-8 text-center">
        <p className="text-zinc-500 text-sm">Групповая стадия не начата.</p>
      </div>
    );
  }

  const selectedIdx = groupNames.indexOf(selected ?? "");
  const selectedData = groupQueries[selectedIdx]?.data;

  return (
    <div className="space-y-4">

      {/* Схема турнира */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-4 py-3">
        <p className="text-[10px] font-bold uppercase tracking-widest text-zinc-600 mb-2">Схема турнира</p>
        <div className="flex items-center gap-2 flex-wrap">
          {groupNames.map(g => (
            <div key={g} className="rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-center">
              <p className="text-xs font-black text-yellow-400">Группа {g}</p>
              <p className="text-[10px] text-zinc-400">топ {advance}</p>
            </div>
          ))}
          <ArrowRight size={15} className="text-zinc-600 flex-shrink-0" />
          <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/5 px-3 py-1.5 text-center">
            <p className="text-xs font-bold text-emerald-400">Плей-офф</p>
            <p className="text-[10px] text-zinc-400">{groupNames.length * advance} уч.</p>
          </div>
          <ArrowRight size={15} className="text-zinc-600 flex-shrink-0" />
          <div className="rounded-lg border border-yellow-500/30 bg-yellow-500/5 px-3 py-1.5 text-center">
            <Trophy size={12} className="text-yellow-400 mx-auto mb-0.5" />
            <p className="text-[10px] text-zinc-400">Финал</p>
          </div>
        </div>
      </div>

      {/* Табы групп */}
      <div className="flex gap-1 border-b border-zinc-700">
        {groupNames.map(g => (
          <button
            key={g}
            onClick={() => setActiveGroup(g)}
            className={cn(
              "flex items-center gap-1.5 px-4 py-2.5 text-sm font-semibold border-b-2 -mb-px transition-colors",
              selected === g
                ? "border-yellow-400 text-yellow-400"
                : "border-transparent text-zinc-400 hover:text-white"
            )}
          >
            Группа {g}
          </button>
        ))}
      </div>

      {/* Контент выбранной группы */}
      {selected && (
        selectedData ? (
          <GroupContent
            standings={selectedData.standings}
            advance={advance}
            me={currentUserId}
          />
        ) : (
          <div className="py-6 text-center text-zinc-600 text-sm">Загрузка...</div>
        )
      )}
    </div>
  );
}
