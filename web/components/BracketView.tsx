"use client";

import { memo, useEffect, useRef, useState } from "react";
import { AnimatePresence, m, useReducedMotion } from "framer-motion";
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
const ELBOW_R   = 6;     // радиус скругления «колена»

/* Локализованное имя стадии: ключ leagueDetail.stageXX, label с бэка — фолбэк */
const STAGE_T_KEY: Record<string, string> = {
  qf: "stageQF", sf: "stageSF", final: "stageFinal", r16: "stageR16", r32: "stageR32",
};

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

/* Скруглённое «колено»: от центра родителя к точке входа ребёнка */
function elbowPath(fromY: number, midY: number): string {
  const hx = CONN_W / 2;
  const r  = Math.min(ELBOW_R, Math.abs(midY - fromY) / 2);
  if (Math.abs(midY - fromY) < 1) {
    return `M 0 ${fromY} H ${CONN_W}`;
  }
  const dir = midY > fromY ? 1 : -1; // вниз или вверх к ребёнку
  return [
    `M 0 ${fromY}`,
    `H ${hx - r}`,
    `Q ${hx} ${fromY} ${hx} ${fromY + dir * r}`,
    `V ${midY - dir * r}`,
    `Q ${hx} ${midY} ${hx + r} ${midY}`,
    `H ${CONN_W}`,
  ].join(" ");
}

/* ─── SVG коннекторы ─────────────────────────────────────── */
function Connectors({ prevPos, currPos, h, prevSlots, championId, stageIndex, reduced }: {
  prevPos: number[]; currPos: number[]; h: number;
  prevSlots: BracketSlot[]; championId?: number; stageIndex: number; reduced: boolean;
}) {
  const gradId = `bracket-gold-${stageIndex}`;
  const base: { d: string; won: boolean; champ: boolean }[] = [];

  currPos.forEach((cy, j) => {
    const midY = cy + CARD_H / 2;
    for (const pi of [j * 2, j * 2 + 1]) {
      if (pi >= prevPos.length) continue;
      const slot   = prevSlots[pi];
      const fromY  = prevPos[pi] + CARD_H / 2;
      const won    = !!slot?.winner_user_id;
      const champ  = !!championId && slot?.winner_user_id === championId;
      base.push({ d: elbowPath(fromY, midY), won, champ });
    }
  });

  return (
    <svg width={CONN_W} height={h} style={{ flexShrink: 0, overflow: "visible" }} aria-hidden="true">
      <defs>
        <linearGradient id={gradId} x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" stopColor="#fde047" />
          <stop offset="100%" stopColor="#f59e0b" />
        </linearGradient>
      </defs>
      {/* базовый слой: контур всей сетки + подсветка пройденных путей */}
      <g strokeWidth={1.5} strokeLinecap="round" fill="none">
        {base.map((p, i) => (
          <path key={i} d={p.d} stroke={p.won ? "#71717a" : "#3f3f46"} />
        ))}
      </g>
      {/* золотой путь чемпиона — рисуется поверх с draw-in анимацией */}
      <g strokeWidth={2.5} strokeLinecap="round" fill="none"
        style={{ filter: "drop-shadow(0 0 4px rgb(245 158 11 / 0.5))" }}>
        {base.filter(p => p.champ).map((p, i) => (
          <path
            key={i}
            d={p.d}
            stroke={`url(#${gradId})`}
            pathLength={1}
            className={reduced ? undefined : "bracket-draw"}
            style={reduced ? undefined : { animationDelay: `${stageIndex * 0.18}s` }}
          />
        ))}
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
        won    && "bg-green-500/5 shadow-[inset_2px_0_0_0_#c8f135]",
      )}
      style={{ height: ROW_H }}
    >
      {hasPlayer ? (
        <>
          <PlayerAvatar displayName={name} favoriteClub={club} size={20} />
          <span className={cn(
            "flex-1 text-xs truncate min-w-0",
            won    ? "font-bold text-yellow-400"  :
            isMe   ? "font-semibold text-yellow-300"  :
            userId ? "font-semibold text-zinc-200"    : "font-semibold text-zinc-500 italic",
          )}>
            {name || "TBD"}
            {won && <Crown size={9} className="inline ml-1 text-yellow-400" />}
          </span>
        </>
      ) : (
        <>
          <div className="h-5 w-5 rounded-full bg-zinc-800 flex-shrink-0" />
          <span className="flex-1 text-[11px] text-zinc-500 italic">TBD</span>
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
const MatchCard = memo(function MatchCard({ slot, me, isChampSlot, label, onShow, onHide }: {
  slot: BracketSlot; me?: number; isChampSlot: boolean; label: string;
  onShow: (slot: BracketSlot, label: string, rect: DOMRect) => void;
  onHide: () => void;
}) {
  const confirmed = slot.match_status === "confirmed";
  const pending   = slot.match_status === "pending_confirm" || slot.match_status === "scheduled";
  const homeWon   = !!slot.winner_user_id && slot.winner_user_id === slot.home_user_id;
  const awayWon   = !!slot.winner_user_id && slot.winner_user_id === slot.away_user_id;

  const show = (e: React.SyntheticEvent<HTMLDivElement>) =>
    onShow(slot, label, e.currentTarget.getBoundingClientRect());

  const ariaLabel = `${label}: ${slot.home_name || "TBD"} — ${slot.away_name || "TBD"}` +
    (confirmed ? `, ${slot.home_goals}:${slot.away_goals}` : "");

  return (
    <div
      tabIndex={0}
      role="button"
      aria-label={ariaLabel}
      onMouseEnter={show}
      onMouseLeave={onHide}
      onFocus={show}
      onBlur={onHide}
      onClick={show}
      className={cn(
        "rounded-xl border bg-zinc-900 overflow-hidden outline-none",
        "focus-visible:ring-2 focus-visible:ring-yellow-400/60",
        confirmed
          ? (isChampSlot ? "border-amber-500/40" : "border-zinc-700")
          : pending ? "border-yellow-500/30 pulse-border" : "border-zinc-800",
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

/* ─── Попап с деталями матча ─────────────────────────────── */
type PopoverState = { slot: BracketSlot; label: string; x: number; y: number; below: boolean };

function SlotPopover({ pop, tbdText }: { pop: PopoverState; tbdText: string }) {
  const { slot } = pop;
  const confirmed = slot.match_status === "confirmed";
  const row = (name: string, club: string | undefined, goals: number | undefined, won: boolean) => (
    <div className="flex items-center gap-2 min-w-0">
      <PlayerAvatar displayName={name || "TBD"} favoriteClub={club} size={22} />
      <span className={cn("flex-1 truncate text-sm min-w-0", won ? "font-bold text-yellow-400" : "text-zinc-200")}>
        {name || "TBD"}
      </span>
      {typeof goals === "number" && (
        <span className={cn("text-base font-black tabular-nums", won ? "text-yellow-400" : "text-zinc-400")}>
          {goals}
        </span>
      )}
    </div>
  );

  return (
    <m.div
      initial={{ opacity: 0, scale: 0.96, y: pop.below ? -4 : 4 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      exit={{ opacity: 0, scale: 0.96 }}
      transition={{ duration: 0.14 }}
      role="tooltip"
      className="fixed z-50 w-[240px] max-w-[260px] rounded-xl border border-zinc-700 bg-zinc-900/95 backdrop-blur px-3.5 py-3 shadow-xl"
      style={{
        left: pop.x,
        top: pop.y,
        transform: `translateX(-50%)${pop.below ? "" : " translateY(-100%)"}`,
      }}
    >
      <p className="mb-2 text-[10px] font-bold uppercase tracking-widest text-zinc-500">{pop.label}</p>
      <div className="space-y-1.5">
        {row(slot.home_name, slot.home_club, confirmed ? slot.home_goals : undefined,
          !!slot.winner_user_id && slot.winner_user_id === slot.home_user_id)}
        {row(slot.away_name, slot.away_club, confirmed ? slot.away_goals : undefined,
          !!slot.winner_user_id && slot.winner_user_id === slot.away_user_id)}
      </div>
      {!confirmed && (
        <p className="mt-2 text-[11px] text-zinc-500 break-words">{tbdText}</p>
      )}
    </m.div>
  );
}

/* ─── Главный компонент ──────────────────────────────────── */
interface Props {
  stages: BracketStage[];
  currentUserId?: number;
  /** Кнопка «повторить» церемонию чемпиона (если лига завершена). */
  onCelebrate?: () => void;
}

export function BracketView({ stages, currentUserId, onCelebrate }: Props) {
  const { t } = useLang();
  const stageLabel = (st: BracketStage) =>
    STAGE_T_KEY[st.stage] ? (t(`leagueDetail.${STAGE_T_KEY[st.stage]}` as any) as string) : st.label;
  const reduced = !!useReducedMotion();
  const scrollRef = useRef<HTMLDivElement>(null);
  const [pop, setPop] = useState<PopoverState | null>(null);
  const [fade, setFade] = useState({ left: false, right: false });

  const stagesLen = stages?.length ?? 0;

  /* Автоскролл к «живому фронту» сетки + edge-fade подсказки */
  useEffect(() => {
    const el = scrollRef.current;
    if (!el || !stagesLen) return;
    let live = stages.findIndex(s => s.slots.some(sl => !sl.winner_user_id));
    if (live < 0) live = stagesLen - 1;
    el.scrollTo({ left: Math.max(0, live * (CARD_W + CONN_W) - 16) });
    const update = () =>
      setFade({
        left:  el.scrollLeft > 8,
        right: el.scrollLeft + el.clientWidth < el.scrollWidth - 8,
      });
    update();
    el.addEventListener("scroll", update, { passive: true });
    return () => el.removeEventListener("scroll", update);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stagesLen]);

  /* Попап закрывается по Escape и при скролле страницы */
  useEffect(() => {
    if (!pop) return;
    const close = () => setPop(null);
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && close();
    window.addEventListener("scroll", close, { capture: true, passive: true });
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("scroll", close, { capture: true });
      window.removeEventListener("keydown", onKey);
    };
  }, [pop]);

  if (!stagesLen) {
    return (
      <div className="py-12 flex flex-col items-center gap-3 text-zinc-500">
        <Trophy size={32} className="text-zinc-700" />
        <p className="text-sm">{t("leagueDetail.bracketPending")}</p>
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
  const finalSlot  = stages.find(s => s.stage === "final")?.slots[0];
  const champion   = finalSlot?.winner_name;
  const championId = finalSlot?.winner_user_id;

  const showPopover = (slot: BracketSlot, label: string, rect: DOMRect) => {
    const below = rect.top < 190;
    setPop({
      slot,
      label,
      x: Math.min(Math.max(rect.left + rect.width / 2, 130), window.innerWidth - 130),
      y: below ? rect.bottom + 8 : rect.top - 8,
      below,
    });
  };

  return (
    <div className="space-y-4">

      {/* Чемпион — золото, display-шрифт, spring-трофей */}
      {champion && (
        <div className="relative flex items-center gap-3 overflow-hidden rounded-xl border border-amber-500/30 bg-amber-500/5 px-4 py-3 glow-gold">
          <m.div
            initial={reduced ? false : { scale: 0, rotate: -20 }}
            animate={{ scale: 1, rotate: 0 }}
            transition={{ type: "spring", stiffness: 260, damping: 14, delay: 0.15 }}
            className="flex h-10 w-10 items-center justify-center rounded-full flex-shrink-0 text-zinc-900"
            style={{ background: "var(--grad-gold)" }}
          >
            <Trophy size={20} strokeWidth={2.5} />
          </m.div>
          <div className="flex items-center gap-2 min-w-0">
            <PlayerAvatar displayName={champion} favoriteClub={finalSlot?.winner_club} size={32} />
            <div className="min-w-0">
              <p className="text-[10px] font-bold uppercase tracking-widest text-zinc-500">
                {t("leagueDetail.bracketChampion")}
              </p>
              <p className="font-display text-base font-black text-gradient-gold truncate">{champion}</p>
            </div>
          </div>
          {onCelebrate && (
            <button
              onClick={onCelebrate}
              className="ml-auto flex-shrink-0 rounded-lg border border-amber-500/30 px-2.5 py-1.5 text-base leading-none transition-colors hover:bg-amber-500/10"
              aria-label={t("leagueDetail.bracketChampion")}
            >
              🎉
            </button>
          )}
        </div>
      )}

      <div className="relative">
        {/* edge-fade подсказки горизонтального скролла */}
        <div className={cn(
          "pointer-events-none absolute inset-y-0 left-0 z-10 w-8 bg-gradient-to-r from-zinc-950 to-transparent transition-opacity",
          fade.left ? "opacity-100" : "opacity-0",
        )} />
        <div className={cn(
          "pointer-events-none absolute inset-y-0 right-0 z-10 w-8 bg-gradient-to-l from-zinc-950 to-transparent transition-opacity",
          fade.right ? "opacity-100" : "opacity-0",
        )} />

        <div ref={scrollRef} className="overflow-x-auto pb-2" style={{ scrollSnapType: "x proximity" }}>
          <div style={{ display: "inline-flex", alignItems: "flex-start", gap: 0 }}>

            {stages.map((stage, si) => {
              const pos     = posByStage[si];
              const prevPos = si > 0 ? posByStage[si - 1] : null;

              return (
                <div key={stage.stage} style={{ display: "flex", alignItems: "flex-start", scrollSnapAlign: "start" }}>

                  {/* SVG-соединитель слева от этой стадии */}
                  {prevPos && (
                    <div style={{ marginTop: HEADER_H }}>
                      <Connectors
                        prevPos={prevPos} currPos={pos} h={bh}
                        prevSlots={stages[si - 1].slots}
                        championId={championId}
                        stageIndex={si}
                        reduced={reduced}
                      />
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
                        <m.div
                          key={slot.slot}
                          initial={reduced ? false : { opacity: 0, y: 6 }}
                          animate={{ opacity: 1, y: 0 }}
                          transition={{ duration: 0.25, delay: si * 0.07 + i * 0.02 }}
                          style={{ position: "absolute", top: pos[i] ?? 0 }}
                        >
                          <MatchCard
                            slot={slot}
                            me={currentUserId}
                            isChampSlot={!!championId &&
                              (slot.home_user_id === championId || slot.away_user_id === championId)}
                            label={stageLabel(stage)}
                            onShow={showPopover}
                            onHide={() => setPop(null)}
                          />
                        </m.div>
                      ))}
                    </div>

                  </div>
                </div>
              );
            })}

          </div>
        </div>
      </div>

      <AnimatePresence>
        {pop && <SlotPopover pop={pop} tbdText={t("leagueDetail.bracketTbd")} />}
      </AnimatePresence>
    </div>
  );
}
