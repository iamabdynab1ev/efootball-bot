"use client";

import { useMemo } from "react";
import Link from "next/link";
import { useQueries } from "@tanstack/react-query";
import { fetchLeagueDeadlines, stageLabelKey, type Match } from "@/lib/api";
import { DeadlineCountdown } from "@/components/DeadlineCountdown";
import { useLang } from "@/lib/i18n";

// Баннер дедлайнов на главной: игроку показывается, на каком он туре/стадии
// в каждой лиге и сколько времени осталось сыграть и отправить счёт —
// тикающие часики честного турнира.

const KNOCKOUT = new Set(["r32", "r16", "qf", "sf", "final", "3rd"]);

export function DeadlineBanner({ leagueIds, leagueNames, myMatches, userId }: {
  leagueIds: number[];
  leagueNames: Record<number, string>;
  myMatches: Match[];
  userId?: number;
}) {
  const { t } = useLang();

  const deadlineQueries = useQueries({
    queries: leagueIds.map((id) => ({
      queryKey: ["league-deadlines", id],
      queryFn: () => fetchLeagueDeadlines(id),
      staleTime: 60000,
    })),
  });

  const rows = useMemo(() => {
    const out: { leagueId: number; label: string; deadline: string }[] = [];
    leagueIds.forEach((leagueId, i) => {
      const deadlines = deadlineQueries[i]?.data ?? [];
      if (deadlines.length === 0) return;
      // Мои незакрытые матчи этой лиги (играть или подтвердить).
      const mine = myMatches.filter((m) =>
        m.league_id === leagueId &&
        (m.home_user_id === userId || m.away_user_id === userId) &&
        m.status !== "confirmed" && m.status !== "cancelled",
      );
      if (mine.length === 0) return;
      // Ближайший дедлайн, касающийся моих матчей.
      let best: { label: string; deadline: string } | null = null;
      for (const m of mine) {
        const isKO = m.stage && KNOCKOUT.has(m.stage);
        const dl = deadlines.find((d) => isKO ? d.stage === m.stage : (d.stage === "" && d.round === m.round));
        if (!dl) continue;
        const sk = m.stage ? stageLabelKey(m.stage) : undefined;
        const label = isKO && sk
          ? (t(`leagueDetail.${sk}` as never) as string)
          : `${t("deadline.round")} ${m.round}`;
        if (!best || new Date(dl.deadline) < new Date(best.deadline)) {
          best = { label, deadline: dl.deadline };
        }
      }
      if (best) out.push({ leagueId, ...best });
    });
    return out.sort((a, b) => +new Date(a.deadline) - +new Date(b.deadline));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [leagueIds.join(","), deadlineQueries.map((q) => q.dataUpdatedAt).join(","), myMatches, userId, t]);

  if (rows.length === 0) return null;

  return (
    <div className="space-y-2">
      {rows.map((r) => (
        <Link
          key={r.leagueId}
          href={`/leagues/details?id=${r.leagueId}&tab=my`}
          className="pressable flex items-center gap-3 rounded-xl card-premium border border-yellow-400/15 px-4 py-3"
        >
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-bold text-zinc-100">{leagueNames[r.leagueId] ?? ""}</p>
            <p className="mt-0.5 text-[11px] text-zinc-500">{t("deadline.submitHint")}</p>
          </div>
          <DeadlineCountdown deadline={r.deadline} label={r.label} />
        </Link>
      ))}
    </div>
  );
}
