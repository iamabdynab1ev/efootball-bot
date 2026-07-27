"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Check, Minus, MessageSquare, Plus, RotateCcw, Send, ShieldAlert, ShieldCheck, Trash2 } from "lucide-react";
import { Match, adminCancelScore, adminSetScore, confirmMatch, disputeMatch, submitResult } from "@/lib/api";
import { openDirect } from "@/lib/chat";
import { useAuth } from "@/lib/auth";
import { playTick } from "@/lib/sound";
import { useLang } from "@/lib/i18n";
import { MatchStatusBadge } from "@/components/StatusBadge";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface Props {
  match: Match;
  onUpdate?: () => void;
  compact?: boolean;
  // Раскрыть блок админ-контроля счёта сразу (для ввода счёта из списка матчей).
  defaultAdminOpen?: boolean;
}

function scoreOf(value?: number) {
  return typeof value === "number" ? value : "-";
}

export function MatchCard({ match, onUpdate, compact = false, defaultAdminOpen = false }: Props) {
  const { user } = useAuth();
  const { t } = useLang();
  const router = useRouter();
  const [msgBusy, setMsgBusy] = useState(false);
  const [homeGoals, setHomeGoals] = useState(match.claimed_home ?? match.home_goals ?? 0);
  const [awayGoals, setAwayGoals] = useState(match.claimed_away ?? match.away_goals ?? 0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  // Админ-контроль счёта (поверх ролей игроков): отдельный, сворачиваемый блок.
  const isAdmin = !!user?.is_admin;
  const [adminOpen, setAdminOpen] = useState(defaultAdminOpen);
  const [adminHome, setAdminHome] = useState(match.home_goals ?? 0);
  const [adminAway, setAdminAway] = useState(match.away_goals ?? 0);

  const userId = user?.id;
  const isHome = userId === match.home_user_id;
  const isAway = userId === match.away_user_id;
  const canSubmit = isHome && (match.status === "scheduled" || match.status === "disputed");
  const canConfirm = isAway && match.status === "pending_confirm";
  const displayedHome = match.status === "confirmed" ? match.home_goals : match.claimed_home;
  const displayedAway = match.status === "confirmed" ? match.away_goals : match.claimed_away;
  const isConfirmed = match.status === "confirmed";

  const isWaitingConfirm = isHome && match.status === "pending_confirm";
  const isDisputed       = match.status === "disputed";

  // ЛС сопернику: доступно участнику матча (кроме админ-просмотра чужих матчей).
  const opponentId   = isHome ? match.away_user_id : isAway ? match.home_user_id : undefined;
  const opponentName = isHome ? match.away_name : isAway ? match.home_name : "";
  const messageOpponent = async () => {
    if (!opponentId || msgBusy) return;
    setError("");
    setMsgBusy(true);
    try {
      const room = await openDirect(opponentId);
      router.push(`/messages?room=${room.id}`);
    } catch (e: any) {
      setError(e?.message || "Не удалось открыть личный чат");
    } finally {
      setMsgBusy(false);
    }
  };

  const roleHint = useMemo(() => {
    if (canSubmit && isDisputed) return null; // заменим на отдельный banner ниже
    if (canSubmit) return t("matchCard.roleHome");
    if (canConfirm) return t("matchCard.roleAway");
    if (isWaitingConfirm) return null; // отдельный banner
    if (isHome || isAway) return t("matchCard.roleOwn");
    return null;
  }, [canConfirm, canSubmit, isAway, isHome, isDisputed, isWaitingConfirm, t]);

  // Тап по +/− меняет счёт прямо на большом табло: щелчок + лёгкая вибрация.
  const step = (side: "home" | "away", delta: 1 | -1) => {
    const [val, set] = side === "home" ? [homeGoals, setHomeGoals] : [awayGoals, setAwayGoals];
    const next = Math.min(50, Math.max(0, val + delta));
    if (next === val) return;
    set(next);
    playTick(delta > 0 ? "up" : "down");
    try { navigator.vibrate?.(8); } catch { /* не поддерживается — ок */ }
  };

  const act = async (fn: () => Promise<unknown>) => {
    setError("");
    setLoading(true);
    try {
      await fn();
      onUpdate?.();
    } catch (e: any) {
      setError(e?.message || t("matchCard.actionError"));
    } finally {
      setLoading(false);
    }
  };

  if (compact) {
    return (
      <article className="flex items-center gap-3 px-4 py-3 border-b border-zinc-800/50 last:border-0">
        <div className="flex-1 space-y-1.5">
          <div className={cn("flex items-center gap-2 text-sm", isHome && "text-yellow-400 font-semibold")}>
            <PlayerAvatar displayName={match.home_name} favoriteClub={match.home_club} size={22} />
            <span>{match.home_name || t("matchCard.home")}</span>
          </div>
          <div className={cn("flex items-center gap-2 text-sm", isAway && "text-yellow-400 font-semibold")}>
            <PlayerAvatar displayName={match.away_name} favoriteClub={match.away_club} size={22} />
            <span>{match.away_name || t("matchCard.away")}</span>
          </div>
        </div>
        <div className="flex flex-col items-end gap-1.5">
          {isConfirmed ? (
            <span className="text-base font-black text-green-400">{match.home_goals}:{match.away_goals}</span>
          ) : typeof displayedHome === "number" && typeof displayedAway === "number" ? (
            <span className="text-base font-black text-amber-400">{displayedHome}:{displayedAway}</span>
          ) : (
            <span className="text-xs text-zinc-600">{t("matchCard.noResult")}</span>
          )}
          <MatchStatusBadge status={match.status} />
        </div>
      </article>
    );
  }

  return (
    <article className={cn(
      "rounded-xl card-premium overflow-hidden card-interactive border-t-2",
      // Статусная полоса сверху: зелёная — подтверждён, вольт — идёт, красная — спор
      isConfirmed ? "border-t-green-500/60"
        : isDisputed ? "border-t-red-500/60"
        : "border-t-yellow-400/50",
    )}>
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2.5 border-b border-zinc-800">
        <span className="text-xs font-semibold text-zinc-500 uppercase tracking-wide">
          {t("matchCard.round")} {match.round}
        </span>
        <div className="flex items-center gap-2">
          {match.best_of && match.best_of > 1 && (
            <span className="rounded-md bg-zinc-800 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-zinc-400">
              {t("matchCard.series")} BO{match.best_of} · <span className="font-display tabular-nums text-zinc-200">{match.home_wins ?? 0}:{match.away_wins ?? 0}</span>
            </span>
          )}
          <MatchStatusBadge status={match.status} />
        </div>
      </div>

      {/* Scoreline — «табло»: утопленная тёмная панель, display-шрифт.
          Хозяину при вводе счёта цифры на табло живые: +/− под игроками
          меняют их сразу, без дублирующего блока внизу. */}
      <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-4 px-5 py-5">
        <div className="flex flex-col items-center gap-2 text-center min-w-0">
          <PlayerAvatar displayName={match.home_name} favoriteClub={match.home_club} size={44} />
          <div className="min-w-0 max-w-full">
            <p className={cn("text-sm font-bold leading-tight truncate", isHome ? "text-yellow-400" : "text-zinc-200")}>
              {match.home_name || t("matchCard.home")}
            </p>
            <p className="text-[10px] uppercase text-zinc-600 tracking-wide">{t("matchCard.home")}</p>
          </div>
          {canSubmit && (
            <div className="flex items-center gap-2 pt-0.5">
              <StepBtn dir="down" name={match.home_name || t("matchCard.home")} onClick={() => step("home", -1)} />
              <StepBtn dir="up" name={match.home_name || t("matchCard.home")} onClick={() => step("home", 1)} />
            </div>
          )}
        </div>

        <div className={cn(
          "flex items-center gap-2 rounded-xl bg-zinc-950/60 px-4 py-2 shadow-[inset_0_2px_8px_rgb(0_0_0/0.45)]",
          canSubmit && "ring-1 ring-yellow-400/30",
        )}>
          <span
            key={canSubmit ? `h${homeGoals}` : "h"}
            className={cn(
              "font-display text-3xl font-black tabular-nums",
              isConfirmed ? "text-green-400" : canSubmit ? "score-pop text-yellow-400" : "text-zinc-300",
            )}
          >
            {canSubmit ? homeGoals : scoreOf(displayedHome)}
          </span>
          <span className="text-lg font-bold text-zinc-600">:</span>
          <span
            key={canSubmit ? `a${awayGoals}` : "a"}
            className={cn(
              "font-display text-3xl font-black tabular-nums",
              isConfirmed ? "text-green-400" : canSubmit ? "score-pop text-yellow-400" : "text-zinc-300",
            )}
          >
            {canSubmit ? awayGoals : scoreOf(displayedAway)}
          </span>
        </div>

        <div className="flex flex-col items-center gap-2 text-center min-w-0">
          <PlayerAvatar displayName={match.away_name} favoriteClub={match.away_club} size={44} />
          <div className="min-w-0 max-w-full">
            <p className={cn("text-sm font-bold leading-tight truncate", isAway ? "text-yellow-400" : "text-zinc-200")}>
              {match.away_name || t("matchCard.away")}
            </p>
            <p className="text-[10px] uppercase text-zinc-600 tracking-wide">{t("matchCard.away")}</p>
          </div>
          {canSubmit && (
            <div className="flex items-center gap-2 pt-0.5">
              <StepBtn dir="down" name={match.away_name || t("matchCard.away")} onClick={() => step("away", -1)} />
              <StepBtn dir="up" name={match.away_name || t("matchCard.away")} onClick={() => step("away", 1)} />
            </div>
          )}
        </div>
      </div>

      {roleHint && (
        <div className="px-4 pb-3 text-xs text-zinc-500 text-center">{roleHint}</div>
      )}

      {/* Написать сопернику — личный чат (хозяину при вводе счёта кнопка
          чата уезжает в общий ряд действий рядом с «Отправить счёт») */}
      {opponentId && !canSubmit && (
        <div className="px-4 pb-3">
          <button
            onClick={messageOpponent}
            disabled={msgBusy}
            className="flex w-full items-center justify-center gap-2 rounded-lg border border-zinc-700 bg-zinc-800/50 py-2 text-xs font-semibold text-zinc-300 hover:bg-zinc-800 hover:text-yellow-400 transition-colors disabled:opacity-50"
          >
            <MessageSquare size={14} />
            {msgBusy ? "Открываю…" : `Написать${opponentName ? " " + opponentName : " сопернику"}`}
          </button>
        </div>
      )}

      {/* Ряд действий хозяина: чат + большая «Отправить счёт» на одной линии */}
      {canSubmit && (
        <div className="flex gap-2 border-t border-zinc-800 px-4 py-3">
          {opponentId && (
            <button
              onClick={messageOpponent}
              disabled={msgBusy}
              aria-label={`Написать${opponentName ? " " + opponentName : " сопернику"}`}
              title={`Написать${opponentName ? " " + opponentName : " сопернику"}`}
              className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg border border-zinc-700 bg-zinc-800/50 text-zinc-300 transition-all hover:bg-zinc-800 hover:text-yellow-400 active:scale-95 disabled:opacity-50"
            >
              <MessageSquare size={16} />
            </button>
          )}
          <Button
            className="h-10 flex-1"
            disabled={loading}
            loading={loading}
            onClick={() => act(() => submitResult(match.id, homeGoals, awayGoals))}
          >
            <Send size={15} /> {t("matchCard.submit")}
          </Button>
        </div>
      )}

      {canConfirm && (
        <div className="border-t border-zinc-800 px-4 py-3 flex gap-2">
          <Button className="flex-1" disabled={loading} onClick={() => act(() => confirmMatch(match.id))}>
            <Check size={15} /> {t("matchCard.confirm")}
          </Button>
          <Button variant="destructive" className="flex-1" disabled={loading}
            onClick={() => act(() => disputeMatch(match.id))}
          >
            <ShieldAlert size={15} /> {t("matchCard.dispute")}
          </Button>
        </div>
      )}

      {/* Хозяин ждёт подтверждения */}
      {isWaitingConfirm && (
        <div className="flex items-center gap-2 border-t border-blue-500/20 bg-blue-500/5 px-4 py-2.5">
          <Check size={13} className="text-blue-400 flex-shrink-0" />
          <span className="text-xs text-blue-300">{t("matchCard.waitingConfirm")}</span>
        </div>
      )}

      {/* Счёт оспорен — хозяин должен переотправить */}
      {isDisputed && isHome && (
        <div className="flex items-center gap-2 border-t border-red-500/20 bg-red-500/5 px-4 py-2.5">
          <RotateCcw size={13} className="text-red-400 flex-shrink-0" />
          <span className="text-xs text-red-300">{t("matchCard.disputedByAway")}</span>
        </div>
      )}

      {/* Для гостя при споре */}
      {isDisputed && isAway && (
        <div className="flex items-center gap-2 border-t border-amber-500/20 bg-amber-500/5 px-4 py-2.5">
          <RotateCcw size={13} className="text-amber-400 flex-shrink-0" />
          <span className="text-xs text-amber-300">{t("matchCard.disputeWarning")}</span>
        </div>
      )}

      {/* Админ-контроль счёта — поверх ролей игроков, приоритет администратора */}
      {isAdmin && (
        <div className="border-t border-zinc-800">
          <button
            onClick={() => setAdminOpen((o) => !o)}
            className="flex w-full items-center gap-2 px-4 py-2.5 text-xs font-semibold text-zinc-400 hover:text-yellow-400 transition-colors"
          >
            <ShieldCheck size={14} /> Админ-контроль счёта
          </button>
          {adminOpen && (
            <div className="px-4 pb-3 space-y-3">
              <div className="flex items-center gap-3">
                <ScoreStepper label={match.home_name || t("matchCard.home")} value={adminHome} onChange={setAdminHome} side="home" />
                <span className="text-zinc-600 font-bold">:</span>
                <ScoreStepper label={match.away_name || t("matchCard.away")} value={adminAway} onChange={setAdminAway} side="away" />
              </div>
              <div className="flex gap-2">
                <Button className="flex-1" disabled={loading} onClick={() => act(() => adminSetScore(match.id, adminHome, adminAway))}>
                  <ShieldCheck size={15} /> Сохранить счёт
                </Button>
                {(isConfirmed || match.status === "disputed" || match.status === "pending_confirm") && (
                  <Button variant="destructive" disabled={loading} onClick={() => act(() => adminCancelScore(match.id))}>
                    <Trash2 size={15} /> Отменить
                  </Button>
                )}
              </div>
              <p className="text-[10px] text-zinc-600">
                Админ имеет приоритет: счёт можно установить, изменить или отменить в любой момент — таблица пересчитается автоматически.
              </p>
            </div>
          )}
        </div>
      )}

      {error && (
        <div className="border-t border-red-500/20 bg-red-500/5 px-4 py-2 text-xs text-red-400">{error}</div>
      )}
    </article>
  );
}

// Круглая кнопка +/− под игроком: «плюс» акцентирован вольтовым цветом,
// tap-target 44px — удобно большим пальцем на смартфоне.
function StepBtn({ dir, name, onClick }: { dir: "up" | "down"; name: string; onClick: () => void }) {
  const up = dir === "up";
  return (
    <button
      type="button"
      aria-label={`${up ? "+1" : "−1"} ${name}`}
      onClick={onClick}
      className={cn(
        "flex h-9 w-9 items-center justify-center rounded-lg border transition-all active:scale-90",
        up
          ? "border-yellow-400/40 bg-yellow-400/10 text-yellow-400 active:bg-yellow-400/20"
          : "border-zinc-700 bg-zinc-800/70 text-zinc-300 active:bg-zinc-700",
      )}
    >
      {up ? <Plus size={15} /> : <Minus size={15} />}
    </button>
  );
}

function ScoreStepper({
  label, value, onChange, side,
}: {
  label: string; value: number; onChange: (v: number) => void;
  side: "home" | "away";
}) {
  const handleInput = (raw: string) => {
    const n = parseInt(raw, 10);
    if (!isNaN(n) && n >= 0 && n <= 50) onChange(n);
    else if (raw === "") onChange(0);
  };

  return (
    <div className="flex-1 flex flex-col items-center gap-1.5">
      <span className="text-[10px] text-zinc-400 truncate max-w-full">{label}</span>
      <div className="flex items-center gap-2">
        <button
          type="button"
          aria-label={`-1 ${label}`}
          onClick={() => { onChange(Math.max(0, value - 1)); playTick("down"); }}
          className="flex h-9 w-9 items-center justify-center rounded-lg border border-zinc-700 bg-zinc-800 text-zinc-300 transition-all hover:bg-zinc-700 hover:text-zinc-100 active:scale-95"
        >
          <Minus size={13} />
        </button>
        <input
          type="number"
          min={0}
          max={50}
          value={value}
          onChange={(e) => handleInput(e.target.value)}
          aria-label={`${label} goals`}
          className="w-10 text-center text-xl font-black text-zinc-100 tabular-nums bg-transparent border-b border-zinc-700 focus:border-yellow-400 focus:outline-none transition-colors [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
        />
        <button
          type="button"
          aria-label={`+1 ${label}`}
          onClick={() => { onChange(Math.min(50, value + 1)); playTick("up"); }}
          className="flex h-9 w-9 items-center justify-center rounded-lg border border-zinc-700 bg-zinc-800 text-zinc-300 transition-all hover:bg-zinc-700 hover:text-zinc-100 active:scale-95"
        >
          <Plus size={13} />
        </button>
      </div>
    </div>
  );
}
