"use client";

import { League, Standing } from "@/lib/api";
import { useLang } from "@/lib/i18n";

interface Props {
  league: League;
  standings: Standing[];
  hasPlayoff: boolean;
}

export function LeagueInfoPanel({ league, standings, hasPlayoff }: Props) {
  const { t } = useLang();
  const totalPlayed = standings.reduce((s, m) => s + m.wins + m.draws + m.losses, 0);
  const totalGoals = standings.reduce((s, m) => s + m.goals_for, 0);

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-4 space-y-4">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Stat label="Участники" value={standings.length} />
        <Stat label="Тип" value={league.rounds_type} />
        <Stat label="Сыграно матчей" value={totalPlayed} />
        <Stat label="Голов забито" value={totalGoals} />
      </div>
      {hasPlayoff && (
        <p className="text-xs text-zinc-500">Плей-офф сетка доступна во вкладке «Кубок».</p>
      )}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg bg-zinc-800/60 px-3 py-2 text-center">
      <p className="text-xs text-zinc-500">{label}</p>
      <p className="mt-0.5 text-lg font-black text-zinc-200">{value}</p>
    </div>
  );
}
