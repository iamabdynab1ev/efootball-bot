"use client";

import { useEffect, useState } from "react";
import { AlarmClock } from "lucide-react";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

// Тикающий обратный отсчёт до дедлайна тура/стадии — «часики» честного
// турнира: игрок всегда видит, сколько осталось на матч и отправку счёта.
// < 24 ч — янтарный, < 3 ч — красный с пульсом, истёк — «время вышло».

function useNow(active: boolean) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!active) return;
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, [active]);
  return now;
}

export function formatLeft(ms: number, dayLabel: string): string {
  if (ms <= 0) return "0:00:00";
  const s = Math.floor(ms / 1000);
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const pad = (n: number) => String(n).padStart(2, "0");
  if (d > 0) return `${d}${dayLabel} ${pad(h)}:${pad(m)}:${pad(sec)}`;
  return `${h}:${pad(m)}:${pad(sec)}`;
}

export function DeadlineCountdown({ deadline, label, compact = false }: {
  deadline: string;       // ISO
  label?: string;         // «Тур 3» / «Четвертьфинал»
  compact?: boolean;      // маленький чип для шапки лиги
}) {
  const { t } = useLang();
  const target = new Date(deadline).getTime();
  const now = useNow(Number.isFinite(target));
  if (!Number.isFinite(target)) return null;

  const left = target - now;
  const expired = left <= 0;
  const critical = !expired && left < 3 * 3600_000;
  const warning = !expired && !critical && left < 24 * 3600_000;

  const tone = expired
    ? "border-zinc-700 bg-zinc-800/60 text-zinc-500"
    : critical
      ? "border-red-500/50 bg-red-500/10 text-red-400"
      : warning
        ? "border-amber-400/50 bg-amber-400/10 text-amber-400"
        : "border-yellow-400/30 bg-yellow-400/5 text-yellow-400";

  return (
    <span className={cn(
      "inline-flex items-center gap-1.5 rounded-lg border font-semibold tabular-nums",
      compact ? "px-2 py-0.5 text-[11px]" : "px-2.5 py-1 text-xs",
      tone,
    )}>
      <AlarmClock size={compact ? 11 : 13} className={cn(critical && "animate-pulse")} />
      {label && <span className="font-bold">{label}</span>}
      {expired
        ? t("deadline.expired")
        : <span className="font-black">{formatLeft(left, t("deadline.dayShort"))}</span>}
    </span>
  );
}
