"use client";

import { League, Standing, BracketStage } from "@/lib/api";
import { BracketView } from "@/components/BracketView";

interface Props {
  league: League;
  standings: Standing[];
  bracketStages: BracketStage[];
  currentUserId?: number;
}

export function TournamentTree({ league, standings, bracketStages, currentUserId }: Props) {
  if (bracketStages.length === 0) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-4 py-8 text-center">
        <p className="text-zinc-500">Плей-офф ещё не сгенерирован.</p>
      </div>
    );
  }
  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
      <BracketView stages={bracketStages} currentUserId={currentUserId} />
    </div>
  );
}
