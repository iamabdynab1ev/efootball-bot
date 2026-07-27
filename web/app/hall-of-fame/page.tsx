"use client";

import Link from "next/link";

import { useEffect, useState } from "react";
import { Trophy } from "lucide-react";
import { fetchHallOfFame, SeasonAward } from "@/lib/api";
import { tr, useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export default function HallOfFamePage() {
  const { t } = useLang();
  const [awards, setAwards] = useState<SeasonAward[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchHallOfFame().then((d) => { setAwards(d.awards ?? []); setLoading(false); }).catch(() => setLoading(false));
  }, []);

  const labelFor = (type: string) => {
    if (type.startsWith("season_")) {
      const v = tr(`seasonNoms.${type.replace("season_", "")}`);
      return v.startsWith("seasonNoms.") ? type : v;
    }
    if (type === "champion") return t("hallOfFame.champion");
    if (type === "top_scorer") return t("hallOfFame.topScorer");
    if (type === "best_rating") return t("hallOfFame.bestRating");
    const emoji: Record<string, string> = {
      runner_up: "🥈", third_place: "🥉", best_defense: "🛡️", unbeaten: "💯",
      golden_glove: "🧤", best_diff: "⚡", biggest_win: "💥", win_streak: "🔥",
    };
    const name = tr(`trophyCat.${type}.name`);
    if (emoji[type] && !name.startsWith("trophyCat.")) return `${emoji[type]} ${name}`;
    return type;
  };

  // Group by season_id
  const seasons = Array.from(new Set(awards.map((a) => a.season_id))).sort((a, b) => b - a);

  if (loading) return <div className="flex min-h-screen items-center justify-center bg-zinc-950"><div className="text-zinc-400">{t("hallOfFame.loading")}</div></div>;

  return (
    <main className="min-h-screen bg-zinc-950 px-4 py-8">
      <div className="mx-auto max-w-2xl">
        <h1 className="mb-4 font-display text-3xl font-black text-gradient-gold">🏆 {t("hallOfFame.title")}</h1>
        <Link href="/trophies" className="mb-8 flex items-center justify-between rounded-xl border border-yellow-400/20 bg-yellow-400/5 px-4 py-3 text-sm font-semibold text-yellow-400 transition-colors hover:bg-yellow-400/10">
          <span>🎖 Трофейная комната — все награды и как их получить</span>
          <span>→</span>
        </Link>
        {seasons.length === 0 && <p className="text-zinc-400">{t("hallOfFame.noData")}</p>}
        {seasons.map((seasonId) => {
          const seasonAwards = awards.filter((a) => a.season_id === seasonId);
          const leagueNames = Array.from(new Set(seasonAwards.map((a) => a.league_name).filter(Boolean)));
          return (
            <div key={seasonId} className="mb-8">
              <h2 className="mb-3 text-xl font-bold text-zinc-300">{t("hallOfFame.season")} #{seasonId}</h2>

              {/* Герои сезона: номинации церемонии + повтор шоу */}
              {seasonAwards.some((a) => a.award_type.startsWith("season_")) && (
                <div className="mb-4 rounded-xl border border-amber-500/25 bg-amber-500/5 p-4 glow-gold">
                  <div className="mb-3 flex items-center justify-between gap-2">
                    <h3 className="text-base font-semibold text-amber-300/90">✨ {t("season.champions")}</h3>
                    <Link
                      href={`/season?id=${seasonId}`}
                      className="flex-shrink-0 rounded-lg border border-amber-500/40 px-2.5 py-1 text-[11px] font-bold text-amber-300 transition-colors hover:bg-amber-500/10"
                    >
                      ▶ {t("season.ceremony")}
                    </Link>
                  </div>
                  <div className="space-y-2">
                    {seasonAwards.filter((a) => a.award_type.startsWith("season_")).map((a) => (
                      <div key={a.id} className="flex items-center justify-between gap-2 min-w-0">
                        <span className="truncate text-sm font-semibold text-zinc-200">{labelFor(a.award_type)}</span>
                        <span className="flex-shrink-0 text-sm font-black text-amber-300">{a.display_name}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {leagueNames.map((ln) => (
                <div key={ln} className="mb-4 rounded-xl card-premium p-4 card-interactive">
                  <h3 className="mb-3 text-base font-semibold text-zinc-400">{ln}</h3>
                  <div className="space-y-2">
                    {seasonAwards.filter((a) => a.league_name === ln).map((a) => {
                      const isChampion = a.award_type === "champion";
                      return (
                        <div key={a.id} className="flex items-center justify-between gap-2 min-w-0">
                          <span className="flex items-center gap-1.5 text-sm text-zinc-400 flex-shrink-0">
                            {isChampion && <Trophy size={13} className="text-amber-400" />}
                            {labelFor(a.award_type)}
                          </span>
                          <span className={cn(
                            "truncate min-w-0",
                            isChampion
                              ? "font-display font-black text-gradient-gold"
                              : "font-semibold text-zinc-200",
                          )}>
                            {a.display_name}
                          </span>
                        </div>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          );
        })}
      </div>
    </main>
  );
}
