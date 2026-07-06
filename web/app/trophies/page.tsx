"use client";

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Trophy, Lock, CheckCircle2 } from "lucide-react";
import { api, fetchPlayerProfile } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/utils";
import type { PlayerAward } from "@/components/TrophyCabinet";

// Трофейная комната — каталог ВСЕХ наград проекта с условиями получения.
// Залогиненному игроку показывает прогресс: полученные горят, остальные
// приглушены с замком — видно, за что бороться.

interface CatalogItem {
  key: string;          // award_type или код достижения
  emoji: string;
  name: string;
  how: string;          // как получить
  grad: string;
}

const TOURNAMENT: CatalogItem[] = [
  { key: "champion",     emoji: "🏆", name: "Чемпион",           how: "Выиграй турнир", grad: "from-yellow-300/30 to-yellow-900/30" },
  { key: "runner_up",    emoji: "🥈", name: "Серебро",           how: "Займи 2-е место", grad: "from-zinc-200/25 to-zinc-700/25" },
  { key: "third_place",  emoji: "🥉", name: "Бронза",            how: "Займи 3-е место", grad: "from-orange-300/25 to-orange-950/30" },
  { key: "top_scorer",   emoji: "👟", name: "Лучший бомбардир",  how: "Забей больше всех в турнире", grad: "from-yellow-200/25 to-emerald-900/25" },
  { key: "best_defense", emoji: "🛡️", name: "Лучшая защита",     how: "Пропусти меньше всех", grad: "from-sky-300/25 to-slate-900/30" },
  { key: "unbeaten",     emoji: "💯", name: "Непобеждённый",     how: "Пройди турнир без поражений (3+ побед)", grad: "from-fuchsia-300/25 to-purple-950/30" },
  { key: "golden_glove", emoji: "🧤", name: "Золотая перчатка",  how: "Больше всех сухих матчей (от 2)", grad: "from-amber-200/25 to-amber-950/30" },
  { key: "best_diff",    emoji: "⚡", name: "Лучшая разница",    how: "Лучшая разница мячей турнира", grad: "from-lime-200/25 to-green-950/30" },
  { key: "biggest_win",  emoji: "💥", name: "Разгром турнира",   how: "Самая крупная победа (от +5)", grad: "from-red-300/25 to-rose-950/30" },
  { key: "win_streak",   emoji: "🔥", name: "Победная серия",    how: "Самая длинная серия побед (от 3)", grad: "from-orange-200/25 to-red-950/30" },
];

const ACHIEVEMENTS: CatalogItem[] = [
  { key: "first_win",   emoji: "🥇", name: "Первая победа",       how: "Выиграй свой первый матч", grad: "from-yellow-200/20 to-zinc-900/30" },
  { key: "hat_trick",   emoji: "⚽", name: "Хет-трик",             how: "Забей 3+ гола в одном матче", grad: "from-green-200/20 to-zinc-900/30" },
  { key: "poker_5",     emoji: "🎪", name: "Голевое шоу",          how: "Забей 8+ голов в одном матче", grad: "from-lime-200/20 to-zinc-900/30" },
  { key: "thriller_8",  emoji: "🧨", name: "Триллер",              how: "Выиграй перестрелку: 10+ голов на двоих при разнице ≤2 (например 6:5)", grad: "from-red-200/20 to-zinc-900/30" },
  { key: "streak_3",    emoji: "🔥", name: "3 победы подряд",      how: "Победная серия из 3 матчей", grad: "from-orange-200/20 to-zinc-900/30" },
  { key: "streak_5",    emoji: "💥", name: "5 побед подряд",       how: "Победная серия из 5 матчей", grad: "from-rose-200/20 to-zinc-900/30" },
  { key: "streak_10",   emoji: "👑", name: "10 побед подряд",      how: "Победная серия из 10 матчей", grad: "from-amber-200/20 to-zinc-900/30" },
  { key: "scorer_10",   emoji: "👟", name: "10 голов за сезон",    how: "Забей 10 голов в одной лиге", grad: "from-emerald-200/20 to-zinc-900/30" },
  { key: "clean_sheet_5", emoji: "🧤", name: "5 сухих подряд",     how: "5 матчей подряд без пропущенных", grad: "from-sky-200/20 to-zinc-900/30" },
  { key: "veteran",     emoji: "🎖️", name: "50 матчей",           how: "Сыграй 50 матчей", grad: "from-zinc-200/20 to-zinc-900/30" },
  { key: "veteran_100", emoji: "🏅", name: "100 матчей",           how: "Сыграй 100 матчей", grad: "from-yellow-200/20 to-zinc-900/30" },
  { key: "veteran_200", emoji: "🏛", name: "200 матчей",           how: "Сыграй 200 матчей", grad: "from-purple-200/20 to-zinc-900/30" },
  { key: "goals_100",   emoji: "💯", name: "Клуб 100",             how: "Забей 100 голов за карьеру", grad: "from-lime-200/20 to-zinc-900/30" },
  { key: "goals_250",   emoji: "🚀", name: "Клуб 250",             how: "Забей 250 голов за карьеру", grad: "from-cyan-200/20 to-zinc-900/30" },
  { key: "goals_500",   emoji: "🌋", name: "Клуб 500",             how: "Забей 500 голов за карьеру", grad: "from-red-200/20 to-zinc-900/30" },
  { key: "elo_1200",    emoji: "📈", name: "Рейтинг 1200",         how: "Подними ELO до 1200", grad: "from-blue-200/20 to-zinc-900/30" },
  { key: "elo_1300",    emoji: "🚁", name: "Рейтинг 1300",         how: "Подними ELO до 1300", grad: "from-indigo-200/20 to-zinc-900/30" },
  { key: "league_champion", emoji: "🏆", name: "Чемпион лиги",     how: "Выиграй любую лигу", grad: "from-yellow-200/20 to-zinc-900/30" },
  { key: "champ_2",     emoji: "👑", name: "Двукратный чемпион",   how: "Выиграй 2 турнира", grad: "from-amber-200/20 to-zinc-900/30" },
  { key: "champ_3",     emoji: "💎", name: "Трёхкратный чемпион",  how: "Выиграй 3 турнира", grad: "from-cyan-200/20 to-zinc-900/30" },
  { key: "champ_5",     emoji: "🌟", name: "Легенда — 5 титулов",  how: "Выиграй 5 турниров", grad: "from-fuchsia-200/20 to-zinc-900/30" },
];

function TrophyCard({ item, earned, count }: { item: CatalogItem; earned: boolean; count: number }) {
  return (
    <div className={cn(
      "relative flex items-center gap-3 rounded-xl border px-3.5 py-3 transition-colors",
      earned ? "border-yellow-400/25 bg-zinc-900" : "border-zinc-800 bg-zinc-900/60",
    )}>
      <div className={cn(
        "flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-gradient-to-br ring-1 shadow-md shadow-black/30",
        item.grad,
        earned ? "ring-yellow-400/40" : "ring-white/10 opacity-45 grayscale",
      )}>
        <span className="text-[24px] leading-none drop-shadow-[0_2px_3px_rgba(0,0,0,0.5)]">{item.emoji}</span>
      </div>
      <div className="min-w-0 flex-1">
        <p className={cn("text-sm font-bold leading-tight", earned ? "text-zinc-100" : "text-zinc-400")}>{item.name}</p>
        <p className="mt-0.5 text-[11px] leading-snug text-zinc-500">{item.how}</p>
      </div>
      {earned ? (
        <span className="flex flex-shrink-0 items-center gap-1 text-yellow-400">
          {count > 1 && <span className="text-[11px] font-black tabular-nums">×{count}</span>}
          <CheckCircle2 size={16} />
        </span>
      ) : (
        <Lock size={14} className="flex-shrink-0 text-zinc-700" />
      )}
    </div>
  );
}

export default function TrophiesPage() {
  const { user } = useAuth();

  // Что уже получено: трофеи турниров + коды достижений.
  const [awards, setAwards] = useState<PlayerAward[]>([]);
  useEffect(() => {
    if (!user) return;
    let on = true;
    api.get(`/api/players/${user.id}/awards`)
      .then((r) => { if (on) setAwards(r.data.awards ?? []); })
      .catch(() => { /* каталог работает и без прогресса */ });
    return () => { on = false; };
  }, [user]);

  const { data: profile } = useQuery({
    queryKey: ["player-profile", user?.id],
    queryFn: () => fetchPlayerProfile(user!.id),
    enabled: !!user,
  });

  const awardCount = (key: string) => awards.filter((a) => a.award_type === key).length;
  const achievedCodes = new Set((profile?.achievements ?? []).map((ua) => ua.achievement?.code).filter(Boolean));

  const earnedTotal =
    TOURNAMENT.filter((t) => awardCount(t.key) > 0).length +
    ACHIEVEMENTS.filter((a) => achievedCodes.has(a.key)).length;

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div>
        <p className="mb-1 text-xs font-semibold uppercase tracking-widest text-zinc-500">Награды проекта</p>
        <h1 className="flex items-center gap-2 font-display text-2xl font-bold text-zinc-100">
          <Trophy size={22} className="text-yellow-400" /> Трофейная комната
        </h1>
        {user && (
          <p className="mt-1 text-sm text-zinc-500">
            Собрано <span className="font-bold text-yellow-400">{earnedTotal}</span> из {TOURNAMENT.length + ACHIEVEMENTS.length}
          </p>
        )}
      </div>

      <section className="space-y-2.5">
        <h2 className="text-xs font-bold uppercase tracking-wider text-zinc-400">🏟 Трофеи турнира — выдаются по итогам каждой лиги</h2>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {TOURNAMENT.map((t) => (
            <TrophyCard key={t.key} item={t} earned={awardCount(t.key) > 0} count={awardCount(t.key)} />
          ))}
        </div>
      </section>

      <section className="space-y-2.5">
        <h2 className="text-xs font-bold uppercase tracking-wider text-zinc-400">🏅 Достижения — за матчи и карьеру, навсегда</h2>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {ACHIEVEMENTS.map((a) => (
            <TrophyCard key={a.key} item={a} earned={achievedCodes.has(a.key)} count={1} />
          ))}
        </div>
      </section>

      {!user && (
        <p className="rounded-xl border border-zinc-800 bg-zinc-900 px-4 py-3 text-center text-sm text-zinc-500">
          Войдите, чтобы видеть свой прогресс по наградам.
        </p>
      )}
    </div>
  );
}
