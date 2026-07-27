"use client";

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Trophy, Lock, CheckCircle2 } from "lucide-react";
import { api, fetchPlayerProfile } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { tr, useLang } from "@/lib/i18n";
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
  { key: "champion",     emoji: "🏆", name: "Чемпион",           how: "Выиграй турнир", grad: "from-yellow-200 via-yellow-400 to-yellow-700" },
  { key: "runner_up",    emoji: "🥈", name: "Серебро",           how: "Займи 2-е место", grad: "from-zinc-100 via-zinc-300 to-zinc-500" },
  { key: "third_place",  emoji: "🥉", name: "Бронза",            how: "Займи 3-е место", grad: "from-orange-200 via-orange-400 to-orange-700" },
  { key: "top_scorer",   emoji: "👟", name: "Лучший бомбардир",  how: "Забей больше всех в турнире", grad: "from-yellow-200 via-yellow-400 to-emerald-700" },
  { key: "best_defense", emoji: "🛡️", name: "Лучшая защита",     how: "Пропусти меньше всех", grad: "from-sky-200 via-sky-400 to-slate-700" },
  { key: "unbeaten",     emoji: "💯", name: "Непобеждённый",     how: "Пройди турнир без поражений (3+ побед)", grad: "from-fuchsia-200 via-fuchsia-400 to-purple-700" },
  { key: "golden_glove", emoji: "🧤", name: "Золотая перчатка",  how: "Больше всех сухих матчей (от 2)", grad: "from-amber-200 via-amber-400 to-amber-700" },
  { key: "best_diff",    emoji: "⚡", name: "Лучшая разница",    how: "Лучшая разница мячей турнира", grad: "from-lime-200 via-lime-400 to-green-700" },
  { key: "biggest_win",  emoji: "💥", name: "Разгром турнира",   how: "Самая крупная победа (от +5)", grad: "from-red-200 via-red-400 to-rose-700" },
  { key: "win_streak",   emoji: "🔥", name: "Победная серия",    how: "Самая длинная серия побед (от 3)", grad: "from-orange-200 via-orange-400 to-red-700" },
];

const ACHIEVEMENTS: CatalogItem[] = [
  { key: "first_win",   emoji: "🥇", name: "Первая победа",       how: "Выиграй свой первый матч", grad: "from-yellow-200 via-yellow-400 to-yellow-700" },
  { key: "hat_trick",   emoji: "⚽", name: "Хет-трик",             how: "Забей 3+ гола в одном матче", grad: "from-green-200 via-green-400 to-green-700" },
  { key: "poker_5",     emoji: "🎪", name: "Голевое шоу",          how: "Забей 8+ голов в одном матче", grad: "from-lime-200 via-lime-400 to-lime-700" },
  { key: "thriller_8",  emoji: "🧨", name: "Триллер",              how: "Выиграй перестрелку: 10+ голов на двоих при разнице ≤2 (например 6:5)", grad: "from-red-200 via-red-400 to-red-700" },
  { key: "streak_3",    emoji: "🔥", name: "3 победы подряд",      how: "Победная серия из 3 матчей", grad: "from-orange-200 via-orange-400 to-orange-700" },
  { key: "streak_5",    emoji: "💥", name: "5 побед подряд",       how: "Победная серия из 5 матчей", grad: "from-rose-200 via-rose-400 to-rose-700" },
  { key: "streak_10",   emoji: "👑", name: "10 побед подряд",      how: "Победная серия из 10 матчей", grad: "from-amber-200 via-amber-400 to-amber-700" },
  { key: "scorer_10",   emoji: "👟", name: "10 голов за сезон",    how: "Забей 10 голов в одной лиге", grad: "from-emerald-200 via-emerald-400 to-emerald-700" },
  { key: "clean_sheet_5", emoji: "🧤", name: "5 сухих подряд",     how: "5 матчей подряд без пропущенных", grad: "from-sky-200 via-sky-400 to-sky-700" },
  { key: "veteran",     emoji: "🎖️", name: "50 матчей",           how: "Сыграй 50 матчей", grad: "from-zinc-100 via-zinc-300 to-zinc-500" },
  { key: "veteran_100", emoji: "🏅", name: "100 матчей",           how: "Сыграй 100 матчей", grad: "from-yellow-200 via-yellow-400 to-yellow-700" },
  { key: "veteran_200", emoji: "🏛", name: "200 матчей",           how: "Сыграй 200 матчей", grad: "from-purple-200 via-purple-400 to-purple-700" },
  { key: "goals_100",   emoji: "💯", name: "Клуб 100",             how: "Забей 100 голов за карьеру", grad: "from-lime-200 via-lime-400 to-lime-700" },
  { key: "goals_250",   emoji: "🚀", name: "Клуб 250",             how: "Забей 250 голов за карьеру", grad: "from-cyan-200 via-cyan-400 to-cyan-700" },
  { key: "goals_500",   emoji: "🌋", name: "Клуб 500",             how: "Забей 500 голов за карьеру", grad: "from-red-200 via-red-400 to-red-700" },
  { key: "elo_1200",    emoji: "📈", name: "Рейтинг 1200",         how: "Подними ELO до 1200", grad: "from-blue-200 via-blue-400 to-blue-700" },
  { key: "elo_1300",    emoji: "🚁", name: "Рейтинг 1300",         how: "Подними ELO до 1300", grad: "from-indigo-200 via-indigo-400 to-indigo-700" },
  { key: "league_champion", emoji: "🏆", name: "Чемпион лиги",     how: "Выиграй любую лигу", grad: "from-yellow-200 via-yellow-400 to-yellow-700" },
  { key: "champ_2",     emoji: "👑", name: "Двукратный чемпион",   how: "Выиграй 2 турнира", grad: "from-amber-200 via-amber-400 to-amber-700" },
  { key: "champ_3",     emoji: "💎", name: "Трёхкратный чемпион",  how: "Выиграй 3 турнира", grad: "from-cyan-200 via-cyan-400 to-cyan-700" },
  { key: "champ_5",     emoji: "🌟", name: "Легенда — 5 титулов",  how: "Выиграй 5 турниров", grad: "from-fuchsia-200 via-fuchsia-400 to-fuchsia-700" },
];

// Имя/условие трофея на языке интерфейса; для неизвестных ключей — как в каталоге.
function trName(item: CatalogItem) {
  const v = tr(`trophyCat.${item.key}.name`);
  return v.startsWith("trophyCat.") ? item.name : v;
}
function trHow(item: CatalogItem) {
  const v = tr(`trophyCat.${item.key}.how`);
  return v.startsWith("trophyCat.") ? item.how : v;
}

function TrophyCard({ item, earned, count }: { item: CatalogItem; earned: boolean; count: number }) {
  return (
    <div className={cn(
      "relative flex items-center gap-3 rounded-xl border px-3.5 py-3 transition-colors",
      earned ? "border-yellow-400/25 bg-zinc-900" : "border-zinc-800 bg-zinc-900/60",
    )}>
      <div className={cn(
        "relative flex h-12 w-12 flex-shrink-0 items-center justify-center overflow-hidden rounded-full bg-gradient-to-br ring-1",
        item.grad,
        earned
          ? "ring-white/40 shadow-[0_0_18px_rgba(250,204,21,0.3)]"
          : "ring-white/10 opacity-40 grayscale shadow-md shadow-black/30",
      )}>
        {/* Блик — «стеклянная» медаль */}
        <span className="pointer-events-none absolute inset-0 rounded-full bg-[radial-gradient(circle_at_32%_22%,rgba(255,255,255,0.5),transparent_46%)]" />
        <span className="relative text-[24px] leading-none drop-shadow-[0_2px_3px_rgba(0,0,0,0.5)]">{item.emoji}</span>
      </div>
      <div className="min-w-0 flex-1">
        <p className={cn("text-sm font-bold leading-tight", earned ? "text-zinc-100" : "text-zinc-400")}>{trName(item)}</p>
        <p className="mt-0.5 text-[11px] leading-snug text-zinc-500">{trHow(item)}</p>
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
  const { t } = useLang();

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
        <p className="mb-1 text-xs font-semibold uppercase tracking-widest text-zinc-500">{t("trophies.kicker")}</p>
        <h1 className="flex items-center gap-2 font-display text-2xl font-bold text-zinc-100">
          <Trophy size={22} className="text-yellow-400" /> {t("trophies.title")}
        </h1>
        {user && (
          <p className="mt-1 text-sm text-zinc-500">
            {t("trophies.collected")} <span className="font-bold text-yellow-400">{earnedTotal}</span> {t("trophies.of")} {TOURNAMENT.length + ACHIEVEMENTS.length}
          </p>
        )}
      </div>

      <section className="space-y-2.5">
        <div>
          <h2 className="text-xs font-bold uppercase tracking-wider text-zinc-400">{t("trophies.tourTitle")}</h2>
          <p className="mt-0.5 text-[11px] text-zinc-500">{t("trophies.tourDesc")}</p>
        </div>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {TOURNAMENT.map((t) => (
            <TrophyCard key={t.key} item={t} earned={awardCount(t.key) > 0} count={awardCount(t.key)} />
          ))}
        </div>
      </section>

      <section className="space-y-2.5">
        <div>
          <h2 className="text-xs font-bold uppercase tracking-wider text-zinc-400">{t("trophies.achTitle")}</h2>
          <p className="mt-0.5 text-[11px] text-zinc-500">{t("trophies.achDesc")}</p>
        </div>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {ACHIEVEMENTS.map((a) => (
            <TrophyCard key={a.key} item={a} earned={achievedCodes.has(a.key)} count={1} />
          ))}
        </div>
      </section>

      {!user && (
        <p className="rounded-xl card-premium px-4 py-3 text-center text-sm text-zinc-500">
          {t("trophies.loginPrompt")}
        </p>
      )}
    </div>
  );
}
