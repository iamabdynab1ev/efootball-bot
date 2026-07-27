"use client";

import { memo, useMemo } from "react";
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

function StandingsTable({ rows, currentUserId, advance }: { rows: Standing[]; currentUserId?: number; advance?: number }) {
  const { t } = useLang();
  const cutAfter = advance && advance > 0 && advance < rows.length ? advance : 0;

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-zinc-800">
            <th className="w-8 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-400">#</th>
            <th className="py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-zinc-400 pl-2">{t("standings.player")}</th>
            <th className="w-8 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-400">{t("standings.played")}</th>
            <th className="w-8 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-400">{t("standings.wins")}</th>
            <th className="hidden sm:table-cell w-8 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-400">{t("standings.draws")}</th>
            <th className="hidden sm:table-cell w-8 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-400">{t("standings.losses")}</th>
            <th className="w-10 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-400">{t("standings.diff")}</th>
            <th className="hidden sm:table-cell w-24 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-400">{t("standings.form")}</th>
            <th className="w-10 py-2.5 text-right pr-2 sm:pr-4 text-xs font-semibold uppercase tracking-wider text-zinc-400">{t("standings.points")}</th>
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
                className={cn(
                  "border-b border-zinc-800/40 last:border-0 transition-colors",
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
                <td className="py-2.5 pl-2">
                  <div className="flex items-center gap-2">
                    <PlayerAvatar
                      displayName={row.display_name}
                      favoriteClub={row.favorite_club}
                      size={28}
                      bgClassName="bg-zinc-700"
                    />
                    <span className={cn("font-semibold truncate", isMine ? "text-yellow-400" : "text-zinc-200")}>
                      <span className="sm:hidden">{(row.display_name || t("standings.player")).split(" ")[0]}</span>
                      <span className="hidden sm:inline">{row.display_name || t("standings.player")}</span>
                    </span>
                  </div>
                </td>
                <td className="py-2.5 text-center text-zinc-400">{played}</td>
                <td className="py-2.5 text-center text-zinc-400">{row.wins}</td>
                <td className="hidden sm:table-cell py-2.5 text-center text-zinc-400">{row.draws}</td>
                <td className="hidden sm:table-cell py-2.5 text-center text-zinc-400">{row.losses}</td>
                <td className="py-2.5 text-center">
                  <span className={cn(
                    "text-sm font-semibold",
                    diff > 0 ? "text-green-400" : diff < 0 ? "text-red-400" : "text-zinc-500"
                  )}>
                    {diff > 0 ? `+${diff}` : diff}
                  </span>
                </td>
                <td className="hidden sm:table-cell py-2.5 text-center">
                  <FormGuide form={row.form ?? []} />
                </td>
                <td className="py-2.5 text-right pr-2 sm:pr-4">
                  <CountUp value={row.points} className="text-base font-black text-yellow-400 tabular-nums" />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
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
    const entries = [...map.entries()].sort(([a], [b]) => a.localeCompare(b));
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
                Топ-{advance} → плей-офф
              </span>
            ) : null}
          </div>
          <StandingsTable rows={rows} currentUserId={currentUserId} advance={advance} />
        </section>
      ))}
    </div>
  );
});
