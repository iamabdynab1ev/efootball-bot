"use client";

import { memo, useEffect, useMemo, useState } from "react";
import { AnimatePresence, m } from "framer-motion";
import { Standing } from "@/lib/api";
import { useLang } from "@/lib/i18n";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { FormGuide } from "@/components/FormGuide";
import { cn } from "@/lib/utils";
import { CountUp } from "@/components/CountUp";

// Турнирная таблица «как в настоящем футболе»: на групповом этапе — отдельная
// таблица для КАЖДОЙ группы (A, B, …) с зелёной зоной выхода в плей-офф и
// линией отсечения; без групп — одна общая таблица.

interface Props {
  standings: Standing[];
  currentUserId?: number;
  /** Сколько выходят из группы в плей-офф (зелёная зона + линия отсечения). */
  advance?: number;
}

function byTablePosition(a: Standing, b: Standing) {
  return (a.position ?? 999) - (b.position ?? 999)
    || b.points - a.points
    || b.goal_diff - a.goal_diff
    || b.goals_for - a.goals_for;
}

// Заголовок-аббревиатура: тап показывает полное название колонки (подсказка
// с пунктирным подчёркиванием — сигнал «нажми, чтобы понять»).
function StatTh({ w, abbr, full, onShow, align = "center" }: {
  w: string; abbr: string; full: string; align?: "center" | "right";
  onShow: (e: React.MouseEvent<HTMLButtonElement>, text: string) => void;
}) {
  return (
    <th className={cn(w, "py-2.5", align === "right" ? "text-right pr-1.5 sm:pr-4" : "text-center")}>
      <button
        type="button"
        onClick={(e) => onShow(e, full)}
        aria-label={full}
        title={full}
        className={cn(
          "inline-flex items-center text-[10px] sm:text-xs font-semibold uppercase tracking-wider text-zinc-400",
          "underline decoration-dotted decoration-zinc-600/70 underline-offset-[3px] transition-colors hover:text-zinc-200 active:text-yellow-400",
          align === "center" && "mx-auto justify-center",
        )}
      >
        {abbr}
      </button>
    </th>
  );
}

function StandingsTable({ rows, currentUserId, advance }: { rows: Standing[]; currentUserId?: number; advance?: number }) {
  const { t } = useLang();
  const cutAfter = advance && advance > 0 && advance < rows.length ? advance : 0;

  // Подсказка полного названия колонки — fixed по координатам заголовка
  // (не обрезается overflow таблицы), авто-скрытие и закрытие по тапу/скроллу.
  const [tip, setTip] = useState<{ text: string; x: number; y: number } | null>(null);
  useEffect(() => {
    if (!tip) return;
    const close = () => setTip(null);
    window.addEventListener("scroll", close, true);
    document.addEventListener("click", close);
    const timer = window.setTimeout(close, 2600);
    return () => {
      window.removeEventListener("scroll", close, true);
      document.removeEventListener("click", close);
      window.clearTimeout(timer);
    };
  }, [tip]);
  const showTip = (e: React.MouseEvent<HTMLButtonElement>, text: string) => {
    e.stopPropagation();
    const r = e.currentTarget.getBoundingClientRect();
    setTip((cur) => (cur && cur.text === text ? null : {
      text,
      x: Math.min(Math.max(r.left + r.width / 2, 88), window.innerWidth - 88),
      y: r.bottom + 6,
    }));
  };

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-zinc-800">
            <StatTh w="w-6 sm:w-8" abbr="#" full={t("standings.posFull")} onShow={showTip} />
            <th className="py-2.5 text-left text-[10px] sm:text-xs font-semibold uppercase tracking-wider text-zinc-400 pl-1.5 sm:pl-2">{t("standings.player")}</th>
            <StatTh w="w-6 sm:w-8" abbr={t("standings.played")} full={t("standings.playedFull")} onShow={showTip} />
            <StatTh w="w-6 sm:w-8" abbr={t("standings.wins")} full={t("standings.winsFull")} onShow={showTip} />
            <StatTh w="w-6 sm:w-8" abbr={t("standings.draws")} full={t("standings.drawsFull")} onShow={showTip} />
            <StatTh w="w-6 sm:w-8" abbr={t("standings.losses")} full={t("standings.lossesFull")} onShow={showTip} />
            <StatTh w="w-8 sm:w-10" abbr={t("standings.diff")} full={t("standings.diffFull")} onShow={showTip} />
            <StatTh w="w-9 sm:w-24" abbr={t("standings.form")} full={t("standings.formFull")} onShow={showTip} />
            <StatTh w="w-8 sm:w-10" abbr={t("standings.points")} full={t("standings.pointsFull")} onShow={showTip} align="right" />
          </tr>
        </thead>
        <tbody>
          {rows.map((row, index) => {
            const played = row.wins + row.draws + row.losses;
            const pos = row.position ?? index + 1;
            const isMine = row.user_id === currentUserId;
            const qualifies = cutAfter > 0 && index < cutAfter;
            const diff = row.goal_diff;

            return (
              <tr
                key={row.user_id}
                style={{ "--row-i": index } as React.CSSProperties}
                className={cn(
                  "row-in border-b border-zinc-800/40 last:border-0 transition-colors",
                  "hover:bg-white/[0.03] active:bg-white/[0.05]",
                  // Линия отсечения плей-офф — как в таблицах реального футбола.
                  cutAfter > 0 && index === cutAfter - 1 && "border-b-2 border-b-yellow-400/30",
                  qualifies && "bg-green-500/[0.05]",
                  isMine && "bg-yellow-500/5 border-l-2 border-l-yellow-500",
                )}
              >
                <td className="py-2.5 text-center">
                  <span className={cn(
                    "relative inline-flex h-5 w-5 items-center justify-center rounded text-xs font-bold",
                    qualifies ? "bg-green-500/15 text-green-400" :
                    pos === 1 ? "text-yellow-400" :
                    pos === 2 ? "text-zinc-300" :
                    pos === 3 ? "text-amber-500" : "text-zinc-600"
                  )}>
                    {pos}
                  </span>
                </td>
                <td className="py-2.5 pl-1.5 sm:pl-2">
                  <div className="flex items-center gap-1.5 sm:gap-2">
                    <span className="hidden sm:block"><PlayerAvatar
                      displayName={row.display_name}
                      favoriteClub={row.favorite_club}
                      size={28}
                      bgClassName="bg-zinc-700"
                    /></span>
                    <span className="sm:hidden"><PlayerAvatar
                      displayName={row.display_name}
                      favoriteClub={row.favorite_club}
                      size={22}
                      bgClassName="bg-zinc-700"
                    /></span>
                    <span className={cn("font-semibold truncate text-[13px] sm:text-sm max-w-[72px] sm:max-w-none", isMine ? "text-yellow-400" : "text-zinc-200")}>
                      <span className="sm:hidden">{(row.display_name || t("standings.player")).split(" ")[0]}</span>
                      <span className="hidden sm:inline">{row.display_name || t("standings.player")}</span>
                    </span>
                  </div>
                </td>
                <td className="py-2.5 text-center text-[13px] sm:text-sm text-zinc-400 tabular-nums">{played}</td>
                <td className="py-2.5 text-center text-[13px] sm:text-sm text-zinc-400 tabular-nums">{row.wins}</td>
                <td className="py-2.5 text-center text-[13px] sm:text-sm text-zinc-400 tabular-nums">{row.draws}</td>
                <td className="py-2.5 text-center text-[13px] sm:text-sm text-zinc-400 tabular-nums">{row.losses}</td>
                <td className="py-2.5 text-center">
                  <span className={cn(
                    "text-[13px] sm:text-sm font-semibold tabular-nums",
                    diff > 0 ? "text-green-400" : diff < 0 ? "text-red-400" : "text-zinc-500"
                  )}>
                    {diff > 0 ? `+${diff}` : diff}
                  </span>
                </td>
                <td className="py-2.5 text-center">
                  {/* Мобиле: последние 3 матча компактными точками; десктоп — все 5 */}
                  <span className="sm:hidden"><FormGuide form={row.form ?? []} compact max={3} /></span>
                  <span className="hidden sm:inline-block"><FormGuide form={row.form ?? []} /></span>
                </td>
                <td className="py-2.5 text-right pr-1.5 sm:pr-4">
                  <CountUp value={row.points} className="text-sm sm:text-base font-black text-yellow-400 tabular-nums" />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>

      <AnimatePresence>
        {tip && (
          <m.div
            initial={{ opacity: 0, y: -4, scale: 0.96 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, scale: 0.96 }}
            transition={{ duration: 0.13, ease: "easeOut" }}
            role="tooltip"
            className="pointer-events-none fixed z-[60] max-w-[72vw] -translate-x-1/2 whitespace-nowrap rounded-lg border border-zinc-700 bg-zinc-900/95 px-3 py-1.5 text-center text-xs font-semibold text-zinc-100 shadow-xl backdrop-blur"
            style={{ left: tip.x, top: tip.y }}
          >
            {tip.text}
          </m.div>
        )}
      </AnimatePresence>
    </div>
  );
}

export const LeagueStandings = memo(function LeagueStandings({ standings, currentUserId, advance }: Props) {
  const { t } = useLang();

  // Группировка: если у участников проставлены группы — по таблице на группу.
  const groups = useMemo(() => {
    const map = new Map<string, Standing[]>();
    for (const s of standings) {
      const g = s.group_name ?? "";
      if (!map.has(g)) map.set(g, []);
      map.get(g)!.push(s);
    }
    const entries = Array.from(map.entries()).sort(([a], [b]) => a.localeCompare(b));
    for (const [, rows] of entries) rows.sort(byTablePosition);
    return entries;
  }, [standings]);

  const hasGroups = groups.length > 1 || (groups.length === 1 && groups[0][0] !== "");

  if (!hasGroups) {
    return <StandingsTable rows={groups[0]?.[1] ?? []} currentUserId={currentUserId} advance={advance} />;
  }

  return (
    <div className="divide-y divide-zinc-800">
      {groups.map(([name, rows]) => (
        <section key={name || "-"}>
          <div className="flex items-center justify-between border-b border-zinc-800 bg-zinc-950/40 px-4 py-2.5">
            <h3 className="flex items-center gap-2 text-xs font-black uppercase tracking-widest text-zinc-200">
              <span className="flex h-5 w-5 items-center justify-center rounded bg-yellow-400/15 font-display text-[11px] text-yellow-400">{name || "—"}</span>
              {t("leagueDetail.groupTitle")} {name}
            </h3>
            {advance ? (
              <span className="text-[10px] font-semibold uppercase tracking-wide text-green-400/80">
                {t("standingsX.topAdvance").replace("{n}", String(advance))}
              </span>
            ) : null}
          </div>
          <StandingsTable rows={rows} currentUserId={currentUserId} advance={advance} />
        </section>
      ))}
    </div>
  );
});
