"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { League } from "@/lib/api";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

interface Props {
  league: League;
  joined: boolean;
  pending: boolean;
  canJoin: boolean;
  onJoin: () => void;
  joining: boolean;
}

type Countdown = { h: number; m: number; s: number } | "expired" | null;

function useCountdown(deadline?: string): Countdown {
  const [remaining, setRemaining] = useState<Countdown>(null);
  useEffect(() => {
    if (!deadline) return;
    const tick = () => {
      const diff = new Date(deadline).getTime() - Date.now();
      if (diff <= 0) { setRemaining("expired"); return; }
      setRemaining({
        h: Math.floor(diff / 3600000),
        m: Math.floor((diff % 3600000) / 60000),
        s: Math.floor((diff % 60000) / 1000),
      });
    };
    tick();
    const t = setInterval(tick, 1000);
    return () => clearInterval(t);
  }, [deadline]);
  return remaining;
}

/* «Стадионные часы»: каждая единица — отдельный бокс с display-шрифтом */
function ClockUnit({ value, unit }: { value: number; unit: string }) {
  return (
    <span className="inline-flex items-baseline gap-0.5 rounded-md bg-zinc-950/70 px-1.5 py-0.5">
      <span className="font-display text-sm font-black tabular-nums text-yellow-400">
        {String(value).padStart(2, "0")}
      </span>
      <span className="text-[9px] font-semibold text-zinc-500">{unit}</span>
    </span>
  );
}

export function RegistrationCountdown({ league, joined, pending, canJoin, onJoin, joining }: Props) {
  const { t } = useLang();
  const countdown = useCountdown(league.registration_deadline);
  return (
    <div className="rounded-xl border border-yellow-500/30 bg-zinc-900 p-4 card-interactive">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <Link href={`/leagues/details?id=${league.id}`} className="font-semibold text-zinc-200 hover:text-yellow-400 transition-colors truncate block">
            {league.name}
          </Link>
          <p className="text-xs text-zinc-500 mt-0.5">{t("leagues.regOpen")}</p>
          {countdown === "expired" && (
            <p className="text-xs text-zinc-500 mt-1">⏱ {t("leagues.regClosed")}</p>
          )}
          {countdown && countdown !== "expired" && (
            <p className="mt-1.5 flex items-center gap-1" aria-label={`${countdown.h}:${countdown.m}:${countdown.s}`}>
              <ClockUnit value={countdown.h} unit={t("leagues.cdHours")} />
              <ClockUnit value={countdown.m} unit={t("leagues.cdMin")} />
              <ClockUnit value={countdown.s} unit={t("leagues.cdSec")} />
            </p>
          )}
        </div>
        <div className="flex-shrink-0">
          {joined ? (
            <span className="rounded-full bg-green-500/20 px-3 py-1 text-xs font-semibold text-green-400">{t("leagues.youMember")}</span>
          ) : pending ? (
            <span className="rounded-full bg-yellow-500/20 px-3 py-1 text-xs font-semibold text-yellow-400">{t("leagues.underReview")}</span>
          ) : canJoin ? (
            <button
              onClick={onJoin}
              disabled={joining}
              className={cn(
                "rounded-full px-4 py-1.5 text-xs font-bold transition-all",
                joining
                  ? "bg-zinc-700 text-zinc-400 cursor-not-allowed"
                  : "bg-yellow-400 text-zinc-950 hover:bg-yellow-300 hover:shadow-[0_0_14px_rgb(200_241_53/0.3)] active:scale-95"
              )}
            >
              {joining ? "..." : t("leagues.joinNow")}
            </button>
          ) : null}
        </div>
      </div>
    </div>
  );
}
