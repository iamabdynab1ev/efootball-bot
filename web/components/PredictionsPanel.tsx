"use client";

import { useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Minus, Plus, Send, Sparkles } from "lucide-react";
import { toast } from "sonner";
import {
  fetchMyPredictions, fetchPredictionLeaderboard, submitPrediction,
  type Match, type MyPrediction,
} from "@/lib/api";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { playTick } from "@/lib/sound";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

// Вкладка «Прогнозы»: ставь счёт на чужие матчи лиги (виртуальные очки).
// Точный счёт — 5, верная разница — 3, верный исход — 1. Прогнозы других
// скрыты до подтверждения матча; таблица прогнозистов — соревнование в
// соревновании, втягивает даже тех, кто в этом туре не играет.

function ScoreStepperMini({ value, onChange, label }: {
  value: number; onChange: (v: number) => void; label: string;
}) {
  const step = (d: number) => {
    const next = Math.min(20, Math.max(0, value + d));
    if (next === value) return;
    onChange(next);
    playTick(d > 0 ? "up" : "down");
  };
  return (
    <div className="flex items-center gap-1">
      <button type="button" aria-label={`-1 ${label}`} onClick={() => step(-1)}
        className="flex h-8 w-8 items-center justify-center rounded-lg border border-zinc-700 bg-zinc-800/70 text-zinc-300 transition-all active:scale-90 active:bg-zinc-700">
        <Minus size={13} />
      </button>
      <span className="w-6 text-center font-display text-lg font-black tabular-nums text-zinc-100">{value}</span>
      <button type="button" aria-label={`+1 ${label}`} onClick={() => step(1)}
        className="flex h-8 w-8 items-center justify-center rounded-lg border border-yellow-400/40 bg-yellow-400/10 text-yellow-400 transition-all active:scale-90 active:bg-yellow-400/20">
        <Plus size={13} />
      </button>
    </div>
  );
}

function PredictCard({ m, mine, leagueId }: { m: Match; mine?: MyPrediction; leagueId: number }) {
  const { t } = useLang();
  const qc = useQueryClient();
  const [h, setH] = useState(mine?.home_goals ?? 0);
  const [a, setA] = useState(mine?.away_goals ?? 0);
  const [busy, setBusy] = useState(false);
  const changed = !mine || mine.home_goals !== h || mine.away_goals !== a;

  const save = async () => {
    setBusy(true);
    try {
      await submitPrediction(m.id, h, a);
      toast.success(t("predict.saved"));
      qc.invalidateQueries({ queryKey: ["predictions", "my", leagueId] });
    } catch (e: any) {
      toast.error(e?.response?.data?.error ?? t("common.error"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="rounded-xl card-premium px-3.5 py-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 flex-1 items-center gap-1.5">
          <PlayerAvatar displayName={m.home_name} favoriteClub={m.home_club} size={22} />
          <span className="truncate text-[13px] font-semibold text-zinc-200">{m.home_name}</span>
        </div>
        <span className="flex-shrink-0 text-[10px] font-bold uppercase text-zinc-600">vs</span>
        <div className="flex min-w-0 flex-1 items-center justify-end gap-1.5">
          <span className="truncate text-right text-[13px] font-semibold text-zinc-200">{m.away_name}</span>
          <PlayerAvatar displayName={m.away_name} favoriteClub={m.away_club} size={22} />
        </div>
      </div>
      <div className="mt-3 flex items-center justify-between gap-2">
        <ScoreStepperMini value={h} onChange={setH} label={m.home_name ?? ""} />
        <span className="font-bold text-zinc-600">:</span>
        <ScoreStepperMini value={a} onChange={setA} label={m.away_name ?? ""} />
        <button
          onClick={save}
          disabled={busy || (!changed && !!mine)}
          className={cn(
            "ml-1 flex h-8 items-center gap-1.5 rounded-lg px-3 text-xs font-bold transition-all active:scale-95 disabled:opacity-60",
            mine && !changed
              ? "border border-green-500/40 bg-green-500/10 text-green-400"
              : "volt-grad volt-shadow text-zinc-950",
          )}
        >
          {mine && !changed ? <><Check size={13} /> {h}:{a}</> : <><Send size={13} /> {t("predict.submit")}</>}
        </button>
      </div>
    </div>
  );
}

export function PredictionsPanel({ leagueId, matches, currentUserId }: {
  leagueId: number;
  matches: Match[];
  currentUserId?: number;
}) {
  const { t } = useLang();

  const { data: mine = [] } = useQuery({
    queryKey: ["predictions", "my", leagueId],
    queryFn: () => fetchMyPredictions(leagueId),
    enabled: !!currentUserId,
    staleTime: 30_000,
  });
  const { data: board = [] } = useQuery({
    queryKey: ["predictions", "board", leagueId],
    queryFn: () => fetchPredictionLeaderboard(leagueId),
    staleTime: 60_000,
  });

  const myByMatch = useMemo(() => {
    const map = new Map<number, MyPrediction>();
    for (const p of mine) map.set(p.match_id, p);
    return map;
  }, [mine]);

  // Открытые для прогноза: запланированы и не мои.
  const open = useMemo(
    () => matches.filter((m) =>
      m.status === "scheduled" &&
      m.home_user_id !== currentUserId && m.away_user_id !== currentUserId),
    [matches, currentUserId],
  );
  // Мои оценённые прогнозы — с очками.
  const scored = useMemo(
    () => mine.filter((p) => typeof p.points === "number")
      .map((p) => ({ p, m: matches.find((mm) => mm.id === p.match_id) }))
      .filter((x) => x.m)
      .slice(-10)
      .reverse(),
    [mine, matches],
  );
  const myTotal = mine.reduce((sum, p) => sum + (p.points ?? 0), 0);

  return (
    <div className="space-y-4">
      {/* Правила — одна строка, без занудства */}
      <div className="flex items-start gap-2.5 rounded-xl border border-purple-400/20 bg-purple-400/5 px-3.5 py-2.5">
        <Sparkles size={15} className="mt-0.5 flex-shrink-0 text-purple-300" />
        <p className="text-[11px] leading-relaxed text-zinc-400">
          {t("predict.rules")}
        </p>
      </div>

      {/* Топ прогнозистов */}
      {board.length > 0 && (
        <div className="overflow-hidden rounded-xl card-premium">
          <div className="flex items-center gap-2 border-b border-zinc-800 px-4 py-2.5 text-xs font-bold uppercase tracking-wider text-zinc-400">
            🔮 {t("predict.top")}
            {currentUserId && myTotal > 0 && (
              <span className="ml-auto rounded-full bg-purple-400/10 px-2 py-0.5 text-[10px] font-black text-purple-300">
                {t("predict.myPoints")}: {myTotal}
              </span>
            )}
          </div>
          {board.slice(0, 10).map((r, i) => (
            <div
              key={r.user_id}
              style={{ "--row-i": i } as React.CSSProperties}
              className={cn(
                "row-in flex items-center gap-2.5 border-b border-zinc-800/40 px-4 py-2.5 last:border-0",
                r.user_id === currentUserId && "bg-yellow-500/5",
              )}
            >
              <span className={cn(
                "w-5 text-center text-xs font-black",
                i === 0 ? "text-yellow-400" : i === 1 ? "text-zinc-300" : i === 2 ? "text-amber-500" : "text-zinc-600",
              )}>{i + 1}</span>
              <PlayerAvatar displayName={r.name} favoriteClub={r.club} size={26} />
              <span className={cn("min-w-0 flex-1 truncate text-sm font-semibold",
                r.user_id === currentUserId ? "text-yellow-400" : "text-zinc-200")}>{r.name}</span>
              <span className="text-[10px] text-zinc-500">🎯 {r.exact}</span>
              <span className="w-8 text-right font-display text-base font-black tabular-nums text-purple-300">{r.points}</span>
            </div>
          ))}
        </div>
      )}

      {/* Открытые матчи для прогноза */}
      <div>
        <h3 className="mb-2 text-xs font-bold uppercase tracking-wider text-zinc-400">⚽ {t("predict.open")}</h3>
        {!currentUserId ? (
          <p className="rounded-xl card-premium px-4 py-6 text-center text-sm text-zinc-500">{t("predict.loginPrompt")}</p>
        ) : open.length === 0 ? (
          <p className="rounded-xl card-premium px-4 py-6 text-center text-sm text-zinc-500">{t("predict.empty")}</p>
        ) : (
          <div className="space-y-2">
            {open.map((m) => (
              <PredictCard key={m.id} m={m} mine={myByMatch.get(m.id)} leagueId={leagueId} />
            ))}
          </div>
        )}
      </div>

      {/* Мои результаты */}
      {scored.length > 0 && (
        <div>
          <h3 className="mb-2 text-xs font-bold uppercase tracking-wider text-zinc-400">📜 {t("predict.myResults")}</h3>
          <div className="overflow-hidden rounded-xl card-premium">
            {scored.map(({ p, m }) => (
              <div key={p.match_id} className="flex items-center gap-2 border-b border-zinc-800/40 px-3.5 py-2.5 text-[13px] last:border-0">
                <span className="min-w-0 flex-1 truncate text-zinc-300">
                  {m!.home_name} <b className="tabular-nums">{m!.home_goals}:{m!.away_goals}</b> {m!.away_name}
                </span>
                <span className="flex-shrink-0 text-zinc-500 tabular-nums">({p.home_goals}:{p.away_goals})</span>
                <span className={cn(
                  "w-9 flex-shrink-0 rounded-md py-0.5 text-center text-xs font-black",
                  p.points === 5 ? "bg-yellow-400/15 text-yellow-400" :
                  p.points === 3 ? "bg-green-500/15 text-green-400" :
                  p.points === 1 ? "bg-sky-500/15 text-sky-400" : "bg-zinc-800 text-zinc-500",
                )}>+{p.points}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
