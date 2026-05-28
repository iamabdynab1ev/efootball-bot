"use client";

import { useMemo, useState } from "react";
import { Check, Minus, Plus, RotateCcw, Send, ShieldAlert } from "lucide-react";
import { Match, confirmMatch, disputeMatch, submitResult } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { MatchStatusBadge } from "@/components/StatusBadge";
import { TeamShield } from "@/components/TeamShield";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface Props {
  match: Match;
  onUpdate?: () => void;
  compact?: boolean;
}

function scoreOf(value?: number) {
  return typeof value === "number" ? value : "-";
}

export function MatchCard({ match, onUpdate, compact = false }: Props) {
  const { user } = useAuth();
  const [homeGoals, setHomeGoals] = useState(match.claimed_home ?? match.home_goals ?? 0);
  const [awayGoals, setAwayGoals] = useState(match.claimed_away ?? match.away_goals ?? 0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const userId = user?.id;
  const isHome = userId === match.home_user_id;
  const isAway = userId === match.away_user_id;
  const canSubmit = isHome && (match.status === "scheduled" || match.status === "disputed");
  const canConfirm = isAway && match.status === "pending_confirm";
  const displayedHome = match.status === "confirmed" ? match.home_goals : match.claimed_home;
  const displayedAway = match.status === "confirmed" ? match.away_goals : match.claimed_away;
  const isConfirmed = match.status === "confirmed";

  const roleHint = useMemo(() => {
    if (canSubmit) return "Вы хозяин поля: введите счет после матча.";
    if (canConfirm) return "Вы гость: подтвердите счет или отправьте спор.";
    if (isHome || isAway) return "Ваш матч";
    return null;
  }, [canConfirm, canSubmit, isAway, isHome]);

  const act = async (fn: () => Promise<unknown>) => {
    setError("");
    setLoading(true);
    try {
      await fn();
      onUpdate?.();
    } catch (e: any) {
      setError(e?.message || "Не удалось выполнить действие");
    } finally {
      setLoading(false);
    }
  };

  if (compact) {
    return (
      <article className="flex items-center gap-3 px-4 py-3 border-b border-zinc-800/50 last:border-0">
        <div className="flex-1 space-y-1.5">
          <div className={cn("flex items-center gap-2 text-sm", isHome && "text-yellow-400 font-semibold")}>
            <TeamShield name={match.home_name} size={22} />
            <span>{match.home_name || "Хозяин"}</span>
          </div>
          <div className={cn("flex items-center gap-2 text-sm", isAway && "text-yellow-400 font-semibold")}>
            <TeamShield name={match.away_name} size={22} />
            <span>{match.away_name || "Гость"}</span>
          </div>
        </div>
        <div className="flex flex-col items-end gap-1.5">
          {isConfirmed ? (
            <span className="text-base font-black text-green-400">{match.home_goals}:{match.away_goals}</span>
          ) : typeof displayedHome === "number" && typeof displayedAway === "number" ? (
            <span className="text-base font-black text-amber-400">{displayedHome}:{displayedAway}</span>
          ) : (
            <span className="text-xs text-zinc-600">Нет результата</span>
          )}
          <MatchStatusBadge status={match.status} />
        </div>
      </article>
    );
  }

  return (
    <article className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2.5 border-b border-zinc-800">
        <span className="text-xs font-semibold text-zinc-500 uppercase tracking-wide">Тур {match.round}</span>
        <MatchStatusBadge status={match.status} />
      </div>

      {/* Scoreline */}
      <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-4 px-5 py-5">
        <div className="flex flex-col items-center gap-2 text-center">
          <TeamShield name={match.home_name} size={44} />
          <div>
            <p className={cn("text-sm font-bold leading-tight", isHome ? "text-yellow-400" : "text-zinc-200")}>
              {match.home_name || "Хозяин"}
            </p>
            <p className="text-[10px] uppercase text-zinc-600 tracking-wide">Хозяин</p>
          </div>
        </div>

        <div className="flex items-center gap-2 px-3">
          <span className={cn("text-3xl font-black tabular-nums", isConfirmed ? "text-green-400" : "text-zinc-300")}>
            {scoreOf(displayedHome)}
          </span>
          <span className="text-lg font-bold text-zinc-600">:</span>
          <span className={cn("text-3xl font-black tabular-nums", isConfirmed ? "text-green-400" : "text-zinc-300")}>
            {scoreOf(displayedAway)}
          </span>
        </div>

        <div className="flex flex-col items-center gap-2 text-center">
          <TeamShield name={match.away_name} size={44} />
          <div>
            <p className={cn("text-sm font-bold leading-tight", isAway ? "text-yellow-400" : "text-zinc-200")}>
              {match.away_name || "Гость"}
            </p>
            <p className="text-[10px] uppercase text-zinc-600 tracking-wide">Гость</p>
          </div>
        </div>
      </div>

      {roleHint && (
        <div className="px-4 pb-3 text-xs text-zinc-500 text-center">{roleHint}</div>
      )}

      {canSubmit && (
        <div className="border-t border-zinc-800 px-4 py-3 space-y-3">
          <p className="text-xs text-zinc-500 font-medium uppercase tracking-wide">Введите счёт</p>
          <div className="flex items-center gap-3">
            <ScoreStepper label={match.home_name || "Хозяин"} value={homeGoals} onChange={setHomeGoals} />
            <span className="text-zinc-600 font-bold">:</span>
            <ScoreStepper label={match.away_name || "Гость"} value={awayGoals} onChange={setAwayGoals} />
          </div>
          <Button className="w-full" disabled={loading}
            onClick={() => act(() => submitResult(match.id, homeGoals, awayGoals))}
          >
            <Send size={15} /> Отправить счет
          </Button>
        </div>
      )}

      {canConfirm && (
        <div className="border-t border-zinc-800 px-4 py-3 flex gap-2">
          <Button className="flex-1" disabled={loading} onClick={() => act(() => confirmMatch(match.id))}>
            <Check size={15} /> Подтвердить
          </Button>
          <Button variant="destructive" className="flex-1" disabled={loading}
            onClick={() => act(() => disputeMatch(match.id))}
          >
            <ShieldAlert size={15} /> Спор
          </Button>
        </div>
      )}

      {match.status === "disputed" && match.dispute_count > 0 && (
        <div className="flex items-center gap-2 border-t border-amber-500/20 bg-amber-500/5 px-4 py-2.5">
          <RotateCcw size={13} className="text-amber-400 flex-shrink-0" />
          <span className="text-xs text-amber-300">
            Спор #{match.dispute_count}. Хозяин может отправить счет заново, либо админ решит вручную.
          </span>
        </div>
      )}

      {error && (
        <div className="border-t border-red-500/20 bg-red-500/5 px-4 py-2 text-xs text-red-400">{error}</div>
      )}
    </article>
  );
}

function ScoreStepper({ label, value, onChange }: { label: string; value: number; onChange: (v: number) => void }) {
  return (
    <div className="flex-1 flex flex-col items-center gap-1.5">
      <span className="text-[10px] text-zinc-500 truncate max-w-full">{label}</span>
      <div className="flex items-center gap-2">
        <button
          type="button"
          aria-label="Минус"
          onClick={() => onChange(Math.max(0, value - 1))}
          className="flex h-7 w-7 items-center justify-center rounded-lg border border-zinc-700 bg-zinc-800 text-zinc-300 hover:bg-zinc-700 hover:text-zinc-100 transition-colors"
        >
          <Minus size={13} />
        </button>
        <span className="w-8 text-center text-xl font-black text-zinc-100 tabular-nums">{value}</span>
        <button
          type="button"
          aria-label="Плюс"
          onClick={() => onChange(Math.min(30, value + 1))}
          className="flex h-7 w-7 items-center justify-center rounded-lg border border-zinc-700 bg-zinc-800 text-zinc-300 hover:bg-zinc-700 hover:text-zinc-100 transition-colors"
        >
          <Plus size={13} />
        </button>
      </div>
    </div>
  );
}
