"use client";

import { useEffect, useState } from "react";
import { BracketStage } from "@/lib/api";
import { api } from "@/lib/api";
import { LeagueStandings } from "@/components/LeagueStandings";

interface Props {
  leagueId: number;
  currentUserId?: number;
  onUpdate: () => void;
  advance: number;
  bracketStages: BracketStage[];
}

interface GroupInfo {
  [group: string]: any[];
}

export function GroupStageView({ leagueId, currentUserId, advance }: Props) {
  const [groups, setGroups] = useState<GroupInfo>({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .get(`/api/leagues/${leagueId}/groups`)
      .then((r) => { setGroups(r.data.groups ?? {}); setLoading(false); })
      .catch(() => setLoading(false));
  }, [leagueId]);

  if (loading) return <div className="py-8 text-center text-zinc-500">Загрузка...</div>;

  const groupNames = Object.keys(groups).sort();

  if (groupNames.length === 0) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-4 py-8 text-center">
        <p className="text-zinc-500">Групповая стадия не начата.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {groupNames.map((g) => (
        <div key={g} className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
          <div className="flex items-center px-4 py-3 border-b border-zinc-800">
            <span className="text-sm font-bold text-yellow-400">Группа {g}</span>
            <span className="ml-2 text-xs text-zinc-500">↑ {advance} выходят</span>
          </div>
          <LeagueStandings standings={groups[g]} currentUserId={currentUserId} />
        </div>
      ))}
    </div>
  );
}
