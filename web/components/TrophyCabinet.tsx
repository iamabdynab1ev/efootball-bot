"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";

// Витрина трофеев игрока: золото/серебро/бронза, золотая бутса, лучшая защита.
// Трофеи выдаются автоматически при завершении турнира и остаются навсегда —
// профиль превращается в «шкаф с кубками», за который хочется бороться.

export interface PlayerAward {
  award_type: string;
  league_name: string;
  season_name: string;
  value: number;
  created_at: string;
}

const TROPHIES: Record<string, { emoji: string; label: string; grad: string; ring: string }> = {
  champion:     { emoji: "🏆", label: "Чемпион",            grad: "from-yellow-300/30 via-amber-500/20 to-yellow-900/30", ring: "ring-yellow-400/50" },
  runner_up:    { emoji: "🥈", label: "Серебро",            grad: "from-zinc-200/25 via-zinc-400/15 to-zinc-700/25",      ring: "ring-zinc-300/40" },
  third_place:  { emoji: "🥉", label: "Бронза",             grad: "from-orange-300/25 via-orange-700/15 to-orange-950/30", ring: "ring-orange-400/40" },
  top_scorer:   { emoji: "👟", label: "Лучший бомбардир",   grad: "from-yellow-200/25 via-lime-500/15 to-emerald-900/25",  ring: "ring-lime-400/40" },
  best_defense: { emoji: "🛡️", label: "Лучшая защита",      grad: "from-sky-300/25 via-blue-600/15 to-slate-900/30",       ring: "ring-sky-400/40" },
  unbeaten:     { emoji: "💯", label: "Непобеждённый",      grad: "from-fuchsia-300/25 via-purple-600/15 to-purple-950/30", ring: "ring-fuchsia-400/40" },
  golden_glove: { emoji: "🧤", label: "Золотая перчатка",   grad: "from-amber-200/25 via-yellow-600/15 to-amber-950/30",   ring: "ring-amber-400/40" },
  best_diff:    { emoji: "⚡", label: "Лучшая разница",     grad: "from-lime-200/25 via-green-600/15 to-green-950/30",     ring: "ring-lime-400/40" },
  biggest_win:  { emoji: "💥", label: "Разгром турнира",    grad: "from-red-300/25 via-rose-600/15 to-rose-950/30",        ring: "ring-rose-400/40" },
  win_streak:   { emoji: "🔥", label: "Победная серия",     grad: "from-orange-200/25 via-red-600/15 to-red-950/30",       ring: "ring-orange-400/40" },
};

function TrophyMedal({ a }: { a: PlayerAward }) {
  const t = TROPHIES[a.award_type] ?? { emoji: "🏅", label: a.award_type, grad: "from-zinc-400/20 to-zinc-800/30", ring: "ring-zinc-500/40" };
  const hint =
    a.award_type === "top_scorer" ? `${a.value} голов` :
    a.award_type === "best_defense" ? `${a.value} пропущено` :
    a.award_type === "golden_glove" ? `${a.value} сухих матчей` :
    a.award_type === "best_diff" ? `разница +${a.value}` :
    a.award_type === "biggest_win" ? `победа с разницей ${a.value}` :
    a.award_type === "win_streak" ? `${a.value} побед подряд` : `${a.value} очков`;
  return (
    <div className="flex w-[104px] flex-shrink-0 flex-col items-center gap-2" title={`${t.label} · ${a.league_name} · ${hint}`}>
      <div className={cn(
        "flex h-[72px] w-[72px] items-center justify-center rounded-full bg-gradient-to-br ring-2 shadow-lg shadow-black/40",
        t.grad, t.ring,
      )}>
        <span className="text-[34px] leading-none drop-shadow-[0_2px_4px_rgba(0,0,0,0.55)]">{t.emoji}</span>
      </div>
      <div className="text-center">
        <p className="text-[11px] font-bold leading-tight text-zinc-100">{t.label}</p>
        <p className="mt-0.5 line-clamp-2 text-[10px] leading-tight text-zinc-500">{a.league_name || a.season_name}</p>
      </div>
    </div>
  );
}

export function TrophyCabinet({ userId }: { userId: number }) {
  const [awards, setAwards] = useState<PlayerAward[] | null>(null);

  useEffect(() => {
    let on = true;
    api.get(`/api/players/${userId}/awards`)
      .then((r) => { if (on) setAwards(r.data.awards ?? []); })
      .catch(() => { if (on) setAwards([]); });
    return () => { on = false; };
  }, [userId]);

  if (!awards || awards.length === 0) return null; // пустой шкаф не показываем

  return (
    <div className="overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900">
      <div className="flex items-center gap-2 border-b border-zinc-800 px-4 py-3 text-xs font-bold uppercase tracking-wider text-zinc-400">
        <span className="text-yellow-400">🏆</span> Витрина трофеев
        <span className="ml-auto rounded-full bg-yellow-400/10 px-2 py-0.5 text-[10px] font-black text-yellow-400">{awards.length}</span>
        <Link href="/trophies" className="text-[10px] font-semibold text-zinc-500 hover:text-yellow-400 transition-colors">Все трофеи →</Link>
      </div>
      <div className="flex gap-3 overflow-x-auto px-4 py-4 scrollbar-none">
        {awards.map((a, i) => <TrophyMedal key={`${a.award_type}-${a.league_name}-${i}`} a={a} />)}
      </div>
    </div>
  );
}
