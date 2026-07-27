"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { Trophy, X } from "lucide-react";
import { fetchLatestClosedSeason } from "@/lib/api";
import { useLang } from "@/lib/i18n";

// Баннер на главной: «Церемония сезона “X” — смотреть». Показывается 10 дней
// после закрытия сезона, скрывается тапом навсегда (localStorage).

const SHOW_DAYS = 10;

export function SeasonCeremonyBanner() {
  const { t } = useLang();
  const [hidden, setHidden] = useState(false);
  const { data: season } = useQuery({
    queryKey: ["season", "latest-closed"],
    queryFn: fetchLatestClosedSeason,
    staleTime: 5 * 60_000,
  });

  if (hidden || !season) return null;
  const age = Date.now() - new Date(season.closed_at).getTime();
  if (age > SHOW_DAYS * 24 * 3600_000) return null;
  try {
    if (localStorage.getItem(`season_seen_${season.id}`)) return null;
  } catch { /* private mode */ }

  const dismiss = () => {
    try { localStorage.setItem(`season_seen_${season.id}`, "1"); } catch { /* ок */ }
    setHidden(true);
  };

  return (
    <div className="relative overflow-hidden rounded-xl border border-amber-500/30 bg-gradient-to-r from-amber-500/10 via-yellow-400/5 to-transparent glow-gold">
      <Link
        href={`/season?id=${season.id}`}
        onClick={dismiss}
        className="pressable flex items-center gap-3 px-4 py-3"
      >
        <span className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full text-zinc-900" style={{ background: "var(--grad-gold)" }}>
          <Trophy size={18} />
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-[10px] font-bold uppercase tracking-widest text-amber-400/90">{t("season.watchBanner")}</p>
          <p className="truncate font-display text-sm font-black text-zinc-50">«{season.name}»</p>
        </div>
        <span className="flex-shrink-0 rounded-lg bg-yellow-400 px-3 py-1.5 text-xs font-black text-zinc-950">
          {t("season.watch")} →
        </span>
      </Link>
      <button
        onClick={dismiss}
        aria-label={t("season.skip")}
        className="absolute right-1.5 top-1.5 rounded-full p-1 text-zinc-600 transition-colors hover:text-zinc-300"
      >
        <X size={12} />
      </button>
    </div>
  );
}
