"use client";

import { BracketSlot, BracketStage } from "@/lib/api";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { Crown, Trophy } from "lucide-react";

interface Props {
  stages: BracketStage[];
  currentUserId?: number;
}

function SlotCard({ slot, currentUserId }: { slot: BracketSlot; currentUserId?: number }) {
  const isHomeMe = slot.home_user_id === currentUserId;
  const isAwayMe = slot.away_user_id === currentUserId;
  const isPlayed = slot.match_status === "confirmed";
  const isPending = slot.match_status === "pending_confirm" || slot.match_status === "scheduled";

  const homeIsWinner = slot.winner_user_id === slot.home_user_id;
  const awayIsWinner = slot.winner_user_id === slot.away_user_id;

  return (
    <div className={cn(
      "rounded-xl border bg-zinc-900 overflow-hidden w-[220px] flex-shrink-0",
      isPlayed ? "border-zinc-700" : isPending ? "border-yellow-500/30" : "border-zinc-800"
    )}>
      {/* Home */}
      <div className={cn(
        "flex items-center gap-2 px-3 py-2 border-b border-zinc-800",
        isHomeMe && "bg-yellow-500/5",
        homeIsWinner && "bg-green-500/5"
      )}>
        <div className={cn(
          "flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full text-[10px] font-black",
          slot.home_user_id
            ? homeIsWinner ? "bg-yellow-400 text-zinc-900" : "bg-zinc-700 text-zinc-300"
            : "bg-zinc-800 text-zinc-600"
        )}>
          {slot.home_name ? slot.home_name.slice(0, 1).toUpperCase() : "?"}
        </div>
        <span className={cn(
          "flex-1 text-sm font-semibold truncate",
          homeIsWinner ? "text-yellow-400" : slot.home_user_id ? "text-zinc-200" : "text-zinc-600"
        )}>
          {slot.home_name || "TBD"}
          {isHomeMe && <span className="ml-1 text-[9px] text-yellow-500 font-bold uppercase">вы</span>}
          {homeIsWinner && <Crown size={10} className="inline ml-1 text-yellow-400" />}
        </span>
        {typeof slot.home_goals === "number" && (
          <span className={cn(
            "text-base font-black tabular-nums w-5 text-right",
            homeIsWinner ? "text-yellow-400" : "text-zinc-400"
          )}>
            {slot.home_goals}
          </span>
        )}
      </div>

      {/* Away */}
      <div className={cn(
        "flex items-center gap-2 px-3 py-2",
        isAwayMe && "bg-yellow-500/5",
        awayIsWinner && "bg-green-500/5"
      )}>
        <div className={cn(
          "flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full text-[10px] font-black",
          slot.away_user_id
            ? awayIsWinner ? "bg-yellow-400 text-zinc-900" : "bg-zinc-700 text-zinc-300"
            : "bg-zinc-800 text-zinc-600"
        )}>
          {slot.away_name ? slot.away_name.slice(0, 1).toUpperCase() : "?"}
        </div>
        <span className={cn(
          "flex-1 text-sm font-semibold truncate",
          awayIsWinner ? "text-yellow-400" : slot.away_user_id ? "text-zinc-200" : "text-zinc-600"
        )}>
          {slot.away_name || "TBD"}
          {isAwayMe && <span className="ml-1 text-[9px] text-yellow-500 font-bold uppercase">вы</span>}
          {awayIsWinner && <Crown size={10} className="inline ml-1 text-yellow-400" />}
        </span>
        {typeof slot.away_goals === "number" && (
          <span className={cn(
            "text-base font-black tabular-nums w-5 text-right",
            awayIsWinner ? "text-yellow-400" : "text-zinc-400"
          )}>
            {slot.away_goals}
          </span>
        )}
      </div>
    </div>
  );
}

export function BracketView({ stages, currentUserId }: Props) {
  const { t } = useLang();

  if (!stages || stages.length === 0) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 py-12 flex flex-col items-center gap-3 text-zinc-500">
        <Trophy size={32} className="text-zinc-700" />
        <p className="text-sm">{t("leagueDetail.scheduleEmpty")}</p>
        <p className="text-xs text-zinc-600">{t("leagueDetail.scheduleEmptyText")}</p>
      </div>
    );
  }

  // Find champion (winner of final stage)
  const finalStage = stages.find((s) => s.stage === "final");
  const champion = finalStage?.slots[0]?.winner_name;

  return (
    <div className="space-y-6">
      {/* Champion banner */}
      {champion && (
        <div className="rounded-xl border border-yellow-500/30 bg-yellow-500/5 px-4 py-3 flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-yellow-400 text-zinc-900">
            <Trophy size={20} strokeWidth={2.5} />
          </div>
          <div>
            <p className="text-xs text-zinc-500 uppercase tracking-wider font-semibold">Чемпион</p>
            <p className="text-lg font-black text-yellow-400">{champion}</p>
          </div>
        </div>
      )}

      {/* Bracket stages — horizontal scroll */}
      <div className="overflow-x-auto pb-4">
        <div className="flex gap-6 min-w-max">
          {stages.map((stage) => (
            <div key={stage.stage} className="flex flex-col gap-2">
              {/* Stage header */}
              <div className="px-1 mb-1">
                <p className="text-xs font-bold uppercase tracking-widest text-zinc-400">{stage.label}</p>
                <p className="text-[10px] text-zinc-600">{stage.slots.length} {stage.slots.length === 1 ? "матч" : "матча"}</p>
              </div>

              {/* Slots with vertical spacing for alignment */}
              <div className={cn(
                "flex flex-col",
                // Space out slots so they align with the next round's single slot
                stage.slots.length > 1 ? "gap-4" : "gap-0"
              )}>
                {stage.slots.map((slot) => (
                  <SlotCard key={slot.slot} slot={slot} currentUserId={currentUserId} />
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-4 text-[10px] text-zinc-600 uppercase tracking-wide">
        <div className="flex items-center gap-1.5">
          <span className="inline-block h-2 w-2 rounded-full bg-yellow-400" />
          Победитель
        </div>
        <div className="flex items-center gap-1.5">
          <span className="inline-block h-2 w-2 rounded-full bg-zinc-700" />
          Участник
        </div>
        <div className="flex items-center gap-1.5">
          <span className="inline-block h-2 w-2 rounded-full bg-zinc-800" />
          TBD — ждём результата
        </div>
      </div>
    </div>
  );
}
