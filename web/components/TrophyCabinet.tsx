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

// Каждый трофей — сочный, со своим цветом и свечением: витрина должна
// выглядеть как настоящий шкаф с кубками, за который хочется бороться.
const TROPHIES: Record<string, { emoji: string; label: string; grad: string; ring: string; glow: string }> = {
  champion:     { emoji: "🏆", label: "Чемпион",          grad: "from-yellow-200 via-amber-400 to-yellow-600",  ring: "ring-yellow-300/70",  glow: "shadow-[0_0_26px_rgba(250,204,21,0.4)]" },
  runner_up:    { emoji: "🥈", label: "Серебро",          grad: "from-zinc-100 via-zinc-300 to-zinc-500",       ring: "ring-zinc-200/60",    glow: "shadow-[0_0_22px_rgba(212,212,216,0.3)]" },
  third_place:  { emoji: "🥉", label: "Бронза",           grad: "from-orange-200 via-orange-400 to-orange-700", ring: "ring-orange-300/60",  glow: "shadow-[0_0_22px_rgba(251,146,60,0.32)]" },
  top_scorer:   { emoji: "👟", label: "Лучший бомбардир", grad: "from-lime-200 via-lime-400 to-emerald-600",    ring: "ring-lime-300/60",    glow: "shadow-[0_0_22px_rgba(163,230,53,0.32)]" },
  best_defense: { emoji: "🛡️", label: "Лучшая защита",    grad: "from-sky-200 via-sky-400 to-blue-700",         ring: "ring-sky-300/60",     glow: "shadow-[0_0_22px_rgba(56,189,248,0.32)]" },
  unbeaten:     { emoji: "💯", label: "Непобеждённый",    grad: "from-fuchsia-200 via-fuchsia-400 to-purple-700", ring: "ring-fuchsia-300/60", glow: "shadow-[0_0_22px_rgba(232,121,249,0.32)]" },
  golden_glove: { emoji: "🧤", label: "Золотая перчатка", grad: "from-amber-100 via-amber-400 to-amber-700",    ring: "ring-amber-300/60",   glow: "shadow-[0_0_22px_rgba(251,191,36,0.35)]" },
  best_diff:    { emoji: "⚡", label: "Лучшая разница",   grad: "from-lime-100 via-green-400 to-green-700",     ring: "ring-green-300/60",   glow: "shadow-[0_0_22px_rgba(74,222,128,0.32)]" },
  biggest_win:  { emoji: "💥", label: "Разгром турнира",  grad: "from-rose-200 via-rose-400 to-rose-700",       ring: "ring-rose-300/60",    glow: "shadow-[0_0_22px_rgba(251,113,133,0.32)]" },
  win_streak:   { emoji: "🔥", label: "Победная серия",   grad: "from-orange-200 via-red-400 to-red-700",       ring: "ring-orange-300/60",  glow: "shadow-[0_0_22px_rgba(248,113,113,0.35)]" },
};

function TrophyMedal({ a }: { a: PlayerAward }) {
  const t = TROPHIES[a.award_type] ?? { emoji: "🏅", label: a.award_type, grad: "from-zinc-300 to-zinc-600", ring: "ring-zinc-400/50", glow: "shadow-lg shadow-black/40" };
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
        "relative flex h-[72px] w-[72px] items-center justify-center overflow-hidden rounded-full bg-gradient-to-br ring-2",
        t.grad, t.ring, t.glow,
      )}>
        {/* Блик сверху — «стеклянная» медаль */}
        <span className="pointer-events-none absolute inset-0 rounded-full bg-[radial-gradient(circle_at_32%_22%,rgba(255,255,255,0.55),transparent_46%)]" />
        <span className="relative text-[34px] leading-none drop-shadow-[0_2px_4px_rgba(0,0,0,0.55)]">{t.emoji}</span>
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
    <div className="overflow-hidden rounded-xl card-premium">
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
