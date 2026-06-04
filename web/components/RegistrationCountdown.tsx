"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { League } from "@/lib/api";
import { cn } from "@/lib/utils";

interface Props {
  league: League;
  joined: boolean;
  pending: boolean;
  canJoin: boolean;
  onJoin: () => void;
  joining: boolean;
}

function useCountdown(deadline?: string) {
  const [remaining, setRemaining] = useState<string>("");
  useEffect(() => {
    if (!deadline) return;
    const tick = () => {
      const diff = new Date(deadline).getTime() - Date.now();
      if (diff <= 0) { setRemaining("Завершена"); return; }
      const h = Math.floor(diff / 3600000);
      const m = Math.floor((diff % 3600000) / 60000);
      const s = Math.floor((diff % 60000) / 1000);
      setRemaining(`${h}ч ${m}м ${s}с`);
    };
    tick();
    const t = setInterval(tick, 1000);
    return () => clearInterval(t);
  }, [deadline]);
  return remaining;
}

export function RegistrationCountdown({ league, joined, pending, canJoin, onJoin, joining }: Props) {
  const countdown = useCountdown(league.registration_deadline);
  return (
    <div className="rounded-xl border border-yellow-500/30 bg-zinc-900 p-4">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <Link href={`/leagues/details?id=${league.id}`} className="font-semibold text-zinc-200 hover:text-yellow-400 transition-colors truncate block">
            {league.name}
          </Link>
          <p className="text-xs text-zinc-500 mt-0.5">Регистрация открыта</p>
          {countdown && (
            <p className="text-xs text-yellow-400 mt-0.5">⏱ {countdown}</p>
          )}
        </div>
        <div className="flex-shrink-0">
          {joined ? (
            <span className="rounded-full bg-green-500/20 px-3 py-1 text-xs font-semibold text-green-400">Вы участник</span>
          ) : pending ? (
            <span className="rounded-full bg-yellow-500/20 px-3 py-1 text-xs font-semibold text-yellow-400">На рассмотрении</span>
          ) : canJoin ? (
            <button
              onClick={onJoin}
              disabled={joining}
              className={cn(
                "rounded-full px-4 py-1.5 text-xs font-bold transition-colors",
                joining ? "bg-zinc-700 text-zinc-400 cursor-not-allowed" : "bg-yellow-400 text-zinc-950 hover:bg-yellow-300"
              )}
            >
              {joining ? "..." : "Вступить"}
            </button>
          ) : null}
        </div>
      </div>
    </div>
  );
}
