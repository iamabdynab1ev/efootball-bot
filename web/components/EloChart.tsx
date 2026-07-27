"use client";

import { useEffect, useMemo, useState } from "react";
import { TrendingDown, TrendingUp } from "lucide-react";
import { api } from "@/lib/api";
import { DATE_LOCALES, useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

// График динамики ELO: вольтовая линия с мягкой заливкой по последним
// пересчётам рейтинга (лиги + товарищеские). Игроки любят смотреть свой рост —
// это одна из главных причин возвращаться в профиль.

interface RatingPoint {
  rating: number;
  at: string;
}

export function EloChart({ userId }: { userId: number }) {
  const { t, lang } = useLang();
  const [points, setPoints] = useState<RatingPoint[] | null>(null);

  useEffect(() => {
    let on = true;
    api.get(`/api/players/${userId}/rating-history`)
      .then((r) => { if (on) setPoints(r.data.points ?? []); })
      .catch(() => { if (on) setPoints([]); });
    return () => { on = false; };
  }, [userId]);

  const chart = useMemo(() => {
    if (!points || points.length < 2) return null;
    const W = 600, H = 140, PAD = 8;
    const vals = points.map((p) => p.rating);
    const min = Math.min(...vals), max = Math.max(...vals);
    const span = Math.max(max - min, 20); // не даём графику «плясать» на ±2 очках
    const x = (i: number) => PAD + (i / (points.length - 1)) * (W - PAD * 2);
    const y = (v: number) => PAD + (1 - (v - min + (span - (max - min)) / 2) / span) * (H - PAD * 2);
    const line = vals.map((v, i) => `${i ? "L" : "M"}${x(i).toFixed(1)},${y(v).toFixed(1)}`).join(" ");
    const area = `${line} L${x(vals.length - 1).toFixed(1)},${H} L${x(0).toFixed(1)},${H} Z`;
    return { W, H, line, area, lastX: x(vals.length - 1), lastY: y(vals[vals.length - 1]) };
  }, [points]);

  if (!points || points.length < 2 || !chart) return null; // мало данных — не показываем

  const first = points[0].rating;
  const last = points[points.length - 1].rating;
  const delta = last - first;

  return (
    <div className="overflow-hidden rounded-xl card-premium">
      <div className="flex items-center gap-2 border-b border-zinc-800 px-4 py-3 text-xs font-bold uppercase tracking-wider text-zinc-400">
        📈 {t("elo.title")}
        <span className={cn(
          "ml-auto flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-black",
          delta >= 0 ? "bg-green-500/10 text-green-400" : "bg-red-500/10 text-red-400",
        )}>
          {delta >= 0 ? <TrendingUp size={12} /> : <TrendingDown size={12} />}
          {delta >= 0 ? `+${delta}` : delta}
        </span>
      </div>
      <div className="px-2 pb-2 pt-3">
        <svg viewBox={`0 0 ${chart.W} ${chart.H}`} className="h-28 w-full" preserveAspectRatio="none" aria-label={t("elo.title")}>
          <defs>
            <linearGradient id="eloFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#c8f135" stopOpacity="0.28" />
              <stop offset="100%" stopColor="#c8f135" stopOpacity="0" />
            </linearGradient>
          </defs>
          <path d={chart.area} fill="url(#eloFill)" />
          <path d={chart.line} fill="none" stroke="#c8f135" strokeWidth="2.5" strokeLinejoin="round" strokeLinecap="round" />
          <circle cx={chart.lastX} cy={chart.lastY} r="4" fill="#c8f135" />
          <circle cx={chart.lastX} cy={chart.lastY} r="7" fill="#c8f135" opacity="0.25" />
        </svg>
        <div className="flex items-center justify-between px-2 text-[10px] text-zinc-500">
          <span>{new Date(points[0].at).toLocaleDateString(DATE_LOCALES[lang], { day: "numeric", month: "short" })}</span>
          <span className="font-black tabular-nums text-yellow-400">{last} ELO</span>
        </div>
      </div>
    </div>
  );
}
