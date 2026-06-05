"use client";

import { League, Standing, BracketStage } from "@/lib/api";
import { BracketView } from "@/components/BracketView";
import { buildSeededEmptyBracket, playoffSize } from "@/lib/bracket";
import { Trophy } from "lucide-react";

interface Props {
  league: League;
  standings: Standing[];
  bracketStages: BracketStage[];
  currentUserId?: number;
}

export function TournamentTree({ league, bracketStages, currentUserId }: Props) {
  const isGroups  = league.rounds_type === "groups" || league.rounds_type === "groups_playoff";
  const numGroups = league.num_groups   ?? 0;
  const groupAdv  = league.group_advance ?? 2;
  const total     = playoffSize(numGroups, groupAdv);

  // Реальный плей-офф
  if (bracketStages.length > 0) {
    return <BracketView stages={bracketStages} currentUserId={currentUserId} />;
  }

  // Групповой формат — показываем пустую сетку с посевом
  if (isGroups && total >= 2) {
    return (
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <span className="text-xs font-bold uppercase tracking-widest text-zinc-500">
            Плей-офф · {total} участников
          </span>
          <span className="text-[10px] font-bold text-amber-500 bg-amber-500/10 border border-amber-500/20 px-2 py-0.5 rounded-full">
            Ожидает групповой этап
          </span>
        </div>
        <BracketView
          stages={buildSeededEmptyBracket(numGroups, groupAdv)}
          currentUserId={currentUserId}
        />
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900 py-12 flex flex-col items-center gap-3">
      <Trophy size={32} className="text-zinc-700" />
      <p className="text-zinc-500 text-sm">Плей-офф ещё не сгенерирован</p>
    </div>
  );
}
