"use client";

import { Standing } from "@/lib/api";
import { TeamShield } from "@/components/TeamShield";
import { cn } from "@/lib/utils";

interface Props {
  standings: Standing[];
  currentUserId?: number;
}

export function LeagueStandings({ standings, currentUserId }: Props) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-zinc-800">
            <th className="w-8 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-500">#</th>
            <th className="py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-zinc-500 pl-2">Игрок</th>
            <th className="w-10 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-500">И</th>
            <th className="w-10 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-500">В</th>
            <th className="w-10 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-500">Н</th>
            <th className="w-10 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-500">П</th>
            <th className="w-14 py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-zinc-500">Раз</th>
            <th className="w-12 py-2.5 text-right pr-4 text-xs font-semibold uppercase tracking-wider text-zinc-500">Очк</th>
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
                    "inline-flex h-5 min-w-[20px] items-center justify-center rounded px-1 text-xs font-bold",
                    pos === 1 && "bg-yellow-500/20 text-yellow-400",
                    pos === 2 && "bg-zinc-700 text-zinc-300",
                    pos === 3 && "bg-amber-900/40 text-amber-500",
                    pos > 3 && "text-zinc-600"
                  )}>
                    {pos}
                  </span>
                </td>
                <td className="py-2.5 pl-2">
                  <div className="flex items-center gap-2">
                    <TeamShield name={row.display_name} size={28} />
                    <span className={cn("font-semibold truncate", isMine ? "text-yellow-400" : "text-zinc-200")}>
                      {row.display_name || "Игрок"}
                    </span>
                  </div>
                </td>
                <td className="py-2.5 text-center text-zinc-400">{played}</td>
                <td className="py-2.5 text-center text-zinc-400">{row.wins}</td>
                <td className="py-2.5 text-center text-zinc-400">{row.draws}</td>
                <td className="py-2.5 text-center text-zinc-400">{row.losses}</td>
                <td className="py-2.5 text-center">
                  <span className={cn(
                    "text-sm font-semibold",
                    diff > 0 ? "text-green-400" : diff < 0 ? "text-red-400" : "text-zinc-500"
                  )}>
                    {diff > 0 ? `+${diff}` : diff}
                  </span>
                </td>
                <td className="py-2.5 text-right pr-4">
                  <span className="text-base font-black text-yellow-400">{row.points}</span>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
