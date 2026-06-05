"use client";

import { memo } from "react";
import { BracketSlot, BracketStage } from "@/lib/api";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { Crown, Trophy } from "lucide-react";

/* ─── Размеры (компактные для мобиле) ───────────────────────── */
const CARD_W    = 160;   // ширина карточки матча
const ROW_H     = 30;    // высота одной строки (хозяин / гость)
const CARD_H    = ROW_H * 2; // 60
const CARD_GAP  = 8;     // отступ между карточками первого раунда
const CONN_W    = 32;    // ширина SVG-соединителя
const HEADER_H  = 28;    // высота заголовка стадии

/* ─── Расчёт позиций Y ───────────────────────────────────── */
function firstPositions(n: number): number[] {
  return Array.from({ length: n }, (_, i) => i * (CARD_H + CARD_GAP));
}
function advancePositions(prev: number[]): number[] {
  const res: number[] = [];
  for (let i = 0; i + 1 < prev.length; i += 2) {
    const c1 = prev[i]     + CARD_H / 2;
    const c2 = prev[i + 1] + CARD_H / 2;
    res.push((c1 + c2) / 2 - CARD_H / 2);
  }
  return res;
}
function bracketHeight(pos0: number[]): number {
  return pos0.length > 0 ? pos0[pos0.length - 1] + CARD_H : CARD_H;
}

/* ─── SVG коннекторы ─────────────────────────────────────── */
function Connectors({ prevPos, currPos, h }: {
  prevPos: number[]; currPos: number[]; h: number;
}) {
  const hx = CONN_W / 2;
  return (
    <svg width={CONN_W} height={h} style={{ flexShrink: 0, overflow: "visible" }}>
      <g stroke="#3f3f46" strokeWidth={1.5} strokeLinecap="round" fill="none">
        {currPos.map((_, j) => {
          const ti = j * 2;
          const bi = j * 2 + 1;
          if (bi >= prevPos.length) return null;
          const topY   = prevPos[ti] + CARD_H / 2;
          const botY   = prevPos[bi] + CARD_H / 2;
          const midY   = currPos[j]  + CARD_H / 2;
          return (
            <g key={j}>
              <line x1={0}    y1={topY} x2={hx}     y2={topY} />
              <line x1={hx}   y1={topY} x2={hx}     y2={botY} />
              <line x1={0}    y1={botY} x2={hx}     y2={botY} />
              <line x1={hx}   y1={midY} x2={CONN_W} y2={midY} />
            </g>
          );
        })}
      </g>
    </svg>
  );
}

/* ─── Одна строка игрока внутри карточки ────────────────────── */
function PlayerRow({ name, club, userId, isMe, won, goals, border }: {
  name: string; club?: string; userId?: number;
  isMe: boolean; won: boolean; goals?: number; border: boolean;
}) {
  const hasPlayer = !!userId || !!name;
  return (
    <div
      className={cn(
        "flex items-center gap-2 px-2.5",
        border && "border-t border-zinc-800/60",
        isMe   && "bg-yellow-500/8",
        won    && "bg-green-500/5",
      )}
      style={{ height: ROW_H }}
    >
      {hasPlayer ? (
        <>
          <PlayerAvatar displayName={name} favoriteClub={club} size={20} />
          <span className={cn(
            "flex-1 text-xs font-semibold truncate",
            won    ? "text-yellow-400"  :
            isMe   ? "text-yellow-300"  :
            userId ? "text-zinc-200"    : "text-zinc-500 italic",
          )}>
            {name || "TBD"}
            {won && <Crown size={9} className="inline ml-1 text-yellow-400" />}
          </span>
        </>
      ) : (
        <>
          <div className="h-5 w-5 rounded-full bg-zinc-800 flex-shrink-0" />
          <span className="flex-1 text-[11px] text-zinc-700 italic">TBD</span>
        </>
      )}
      {typeof goals === "number" && (
        <span className={cn("text-sm font-black tabular-nums w-5 text-right flex-shrink-0",
          won ? "text-yellow-400" : "text-zinc-400"
        )}>
          {goals}
        </span>
      )}
    </div>
  );
}

/* ─── Карточка матча ─────────────────────────────────────── */
const MatchCard = memo(function MatchCard({ slot, me }: {
  slot: BracketSlot; me?: number;
}) {
  const confirmed = slot.match_status === "confirmed";
  const pending   = slot.match_status === "pending_confirm" || slot.match_status === "scheduled";
  const homeWon   = !!slot.winner_user_id && slot.winner_user_id === slot.home_user_id;
  const awayWon   = !!slot.winner_user_id && slot.winner_user_id === slot.away_user_id;

  return (
    <div
      className={cn(
        "rounded-xl border bg-zinc-900 overflow-hidden",
        confirmed ? "border-zinc-700"        :
        pending   ? "border-yellow-500/30"   : "border-zinc-800",
      )}
      style={{ width: CARD_W, height: CARD_H }}
    >
      <PlayerRow
        name={slot.home_name} club={slot.home_club} userId={slot.home_user_id}
        isMe={slot.home_user_id === me} won={homeWon}
        goals={confirmed ? slot.home_goals : undefined}
        border={false}
      />
      <PlayerRow
        name={slot.away_name} club={slot.away_club} userId={slot.away_user_id}
        isMe={slot.away_user_id === me} won={awayWon}
        goals={confirmed ? slot.away_goals : undefined}
        border
      />
    </div>
  );
});

/* ─── Главный компонент ──────────────────────────────────── */
interface Props {
  stages: BracketStage[];
  currentUserId?: number;
}

export function BracketView({ stages, currentUserId }: Props) {
  const { t } = useLang();

  if (!stages?.length) {
    return (
      <div className="py-12 flex flex-col items-center gap-3 text-zinc-500">
        <Trophy size={32} className="text-zinc-700" />
        <p className="text-sm">Плей-офф ещё не сгенерирован</p>
      </div>
    );
  }

  /* Вычисляем Y-позиции каждой стадии */
  const posByStage: number[][] = [firstPositions(stages[0].slots.length)];
  for (let i = 1; i < stages.length; i++) {
    posByStage.push(advancePositions(posByStage[i - 1]));
  }
  const bh = bracketHeight(posByStage[0]); // общая высота

  /* Победитель финала */
  const finalSlot = stages.find(s => s.stage === "final")?.slots[0];
  const champion  = finalSlot?.winner_name;

  return (
    <div className="space-y-4">

      {/* Победитель */}
      {champion && (
        <div className="flex items-center gap-3 rounded-xl border border-yellow-500/30 bg-yellow-500/5 px-4 py-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-yellow-400 text-zinc-900 flex-shrink-0">
            <Trophy size={20} strokeWidth={2.5} />
          </div>
          <div className="flex items-center gap-2">
            <PlayerAvatar displayName={champion} favoriteClub={finalSlot?.winner_club} size={32} />
            <div>
              <p className="text-[10px] font-bold uppercase tracking-widest text-zinc-500">Чемпион</p>
              <p className="text-base font-black text-yellow-400">{champion}</p>
            </div>
          </div>
        </div>
      )}

      <div className="overflow-x-auto pb-2">
        <div style={{ display: "inline-flex", alignItems: "flex-start", gap: 0 }}>

          {stages.map((stage, si) => {
            const pos     = posByStage[si];
            const prevPos = si > 0 ? posByStage[si - 1] : null;

            return (
              <div key={stage.stage} style={{ display: "flex", alignItems: "flex-start" }}>

                {/* SVG-соединитель слева от этой стадии */}
                {prevPos && (
                  <div style={{ marginTop: HEADER_H }}>
                    <Connectors prevPos={prevPos} currPos={pos} h={bh} />
                  </div>
                )}

                {/* Стадия: заголовок + карточки */}
                <div style={{ width: CARD_W }}>

                  {/* Заголовок */}
                  <div style={{ height: HEADER_H }} className="flex flex-col justify-end pb-1 px-0.5">
                    <p className="text-[10px] font-bold uppercase tracking-widest text-zinc-500">
                      {stage.label}
                    </p>
                  </div>

                  {/* Карточки с абсолютным позиционированием */}
                  <div style={{ position: "relative", height: bh }}>
                    {stage.slots.map((slot, i) => (
                      <div
                        key={slot.slot}
                        style={{ position: "absolute", top: pos[i] ?? 0 }}
                      >
                        <MatchCard slot={slot} me={currentUserId} />
                      </div>
                    ))}
                  </div>

                </div>
              </div>
            );
          })}

        </div>
      </div>
    </div>
  );
}
