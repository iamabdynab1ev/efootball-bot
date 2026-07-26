"use client";

import { useQuery } from "@tanstack/react-query";
import { GitBranch, Trophy } from "lucide-react";
import { fetchDoubleElim, type DENode, type DEBracketGroup } from "@/lib/api";
import { useLang } from "@/lib/i18n";
import { SkeletonBracket } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/EmptyState";
import { cn } from "@/lib/utils";

/**
 * Сетка двойной элиминации: три секции — верхняя (winners), нижняя (losers),
 * гранд-финал. Раунды внутри секции идут колонками; mobile-first с горизонтальным
 * скроллом. Данные обновляются по интервалу (как одиночная сетка).
 */
export function DoubleElimView({ leagueId, currentUserId }: { leagueId: number; currentUserId?: number }) {
  const { t } = useLang();
  const { data: brackets = [], isLoading } = useQuery({
    queryKey: ["double-elim", leagueId],
    queryFn: () => fetchDoubleElim(leagueId),
    refetchInterval: 15000,
  });

  if (isLoading) return <SkeletonBracket />;
  if (!brackets.length) {
    return (
      <div className="rounded-xl card-premium">
        <EmptyState icon={GitBranch} title={t("leagueDetail.bracketWinner")} text={t("leagueDetail.bracketTbd")} />
      </div>
    );
  }

  const meta: Record<DEBracketGroup["bracket"], { label: string; accent: string }> = {
    de_w: { label: t("leagueDetail.bracketWinners"), accent: "text-yellow-400" },
    de_l: { label: t("leagueDetail.bracketLosers"), accent: "text-zinc-400" },
    de_gf: { label: t("leagueDetail.bracketGrandFinal"), accent: "text-amber-400" },
  };

  return (
    <div className="space-y-6">
      {brackets.map((bg) => (
        <section key={bg.bracket}>
          <h3 className={cn("mb-2 flex items-center gap-2 font-display text-sm font-black uppercase tracking-wide", meta[bg.bracket].accent)}>
            {bg.bracket === "de_gf" ? <Trophy size={15} /> : <GitBranch size={15} />}
            {meta[bg.bracket].label}
          </h3>
          <div className="flex gap-4 overflow-x-auto pb-2">
            {bg.rounds.map((rg) => (
              <div key={rg.round} className="flex min-w-[200px] flex-col justify-center gap-3">
                {rg.nodes.map((n) => (
                  <DENodeCard
                    key={n.node_key}
                    node={n}
                    grand={bg.bracket === "de_gf"}
                    currentUserId={currentUserId}
                    resetLabel={t("leagueDetail.bracketReset")}
                    tbd={t("leagueDetail.bracketTbd")}
                  />
                ))}
              </div>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}

function DENodeCard({
  node,
  grand,
  currentUserId,
  resetLabel,
  tbd,
}: {
  node: DENode;
  grand: boolean;
  currentUserId?: number;
  resetLabel: string;
  tbd: string;
}) {
  const homeWin = node.winner_user_id != null && node.winner_user_id === node.home_user_id;
  const awayWin = node.winner_user_id != null && node.winner_user_id === node.away_user_id;
  const isLive = node.status === "pending_confirm";

  return (
    <div
      className={cn(
        "rounded-lg border bg-zinc-900/80 text-sm",
        grand ? "border-amber-500/40" : "border-zinc-800",
        isLive && "pulse-border",
      )}
    >
      {(grand && node.is_reset) && (
        <div className="border-b border-amber-500/20 px-3 py-1 text-[10px] font-bold uppercase tracking-widest text-amber-400">
          {resetLabel}
        </div>
      )}
      {node.best_of && node.best_of > 1 && (
        <div className="flex items-center justify-between border-b border-zinc-800 px-3 py-1 text-[10px] font-semibold text-zinc-500">
          <span>BO{node.best_of}</span>
          <span className="font-display tabular-nums text-zinc-400">{node.home_wins ?? 0}:{node.away_wins ?? 0}</span>
        </div>
      )}
      <Side
        name={node.home_name}
        goals={node.home_goals}
        win={homeWin}
        me={currentUserId != null && node.home_user_id === currentUserId}
        tbd={tbd}
      />
      <div className="h-px bg-zinc-800" />
      <Side
        name={node.away_name}
        goals={node.away_goals}
        win={awayWin}
        me={currentUserId != null && node.away_user_id === currentUserId}
        tbd={tbd}
      />
    </div>
  );
}

function Side({
  name,
  goals,
  win,
  me,
  tbd,
}: {
  name: string;
  goals?: number;
  win: boolean;
  me: boolean;
  tbd: string;
}) {
  return (
    <div className={cn("flex items-center justify-between gap-2 px-3 py-2", win ? "text-zinc-50" : "text-zinc-400")}>
      <span className={cn("min-w-0 truncate", win && "font-bold", me && "text-yellow-400")}>
        {name || <span className="italic text-zinc-600">{tbd}</span>}
      </span>
      {typeof goals === "number" && (
        <span className={cn("font-display tabular-nums", win ? "font-black text-zinc-50" : "text-zinc-500")}>{goals}</span>
      )}
    </div>
  );
}
