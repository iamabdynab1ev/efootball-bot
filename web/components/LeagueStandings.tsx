"use client";

import { memo } from "react";
import { Standing } from "@/lib/api";
import { useLang } from "@/lib/i18n";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { FormGuide } from "@/components/FormGuide";
import { cn } from "@/lib/utils";

interface Props {
  standings: Standing[];
  currentUserId?: number;
}

export const LeagueStandings = memo(function LeagueStandings({ standings, currentUserId }: Props) {
  const { t } = useLang();

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
            <th className="hidden sm:table-cell w-24 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-400">Form</th>
            <th className="w-10 py-2.5 text-right pr-2 sm:pr-4 text-xs font-semibold uppercase tracking-wider text-zinc-400">{t("standings.points")}</th>
          </tr>
        </thead>
        <tbody>
          {standings.map((row, index) => {
            const played = row.wins + row.draws + row.losses;
            const pos = row.position ?? index + 1;
            const isMine = row.user_id === currentUserId;
            const diff = row.goal_diff;

            return (
              <tr
                key={row.user_id}
                className={cn(
                  "border-b border-zinc-800/40 last:border-0 transition-colors",
                  isMine && "bg-yellow-500/5 border-l-2 border-l-yellow-500"
                )}
              >
                <td className="py-2.5 text-center">
                  <span className={cn(
                    "text-xs font-bold",
                    pos === 1 && "text-yellow-400",
                    pos === 2 && "text-zinc-300",
                    pos === 3 && "text-amber-500",
                    pos > 3 && "text-zinc-600"
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
                  <span className="text-base font-black text-yellow-400">{row.points}</span>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
});
