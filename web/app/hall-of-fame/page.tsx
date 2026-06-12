"use client";

import { useEffect, useState } from "react";
import { Trophy } from "lucide-react";
import { fetchHallOfFame, SeasonAward } from "@/lib/api";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export default function HallOfFamePage() {
  const { t } = useLang();
  const [awards, setAwards] = useState<SeasonAward[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchHallOfFame().then((d) => { setAwards(d.awards ?? []); setLoading(false); }).catch(() => setLoading(false));
  }, []);

  const labelFor = (type: string) => {
    if (type === "champion") return t("hallOfFame.champion");
    if (type === "top_scorer") return t("hallOfFame.topScorer");
    if (type === "best_rating") return t("hallOfFame.bestRating");
    return type;
  };

  // Group by season_id
  const seasons = Array.from(new Set(awards.map((a) => a.season_id))).sort((a, b) => b - a);

  if (loading) return <div className="flex min-h-screen items-center justify-center bg-zinc-950"><div className="text-zinc-400">{t("hallOfFame.loading")}</div></div>;

  return (
    <main className="min-h-screen bg-zinc-950 px-4 py-8">
      <div className="mx-auto max-w-2xl">
        <h1 className="mb-8 font-display text-3xl font-black text-gradient-gold">🏆 {t("hallOfFame.title")}</h1>
        {seasons.length === 0 && <p className="text-zinc-400">{t("hallOfFame.noData")}</p>}
        {seasons.map((seasonId) => {
          const seasonAwards = awards.filter((a) => a.season_id === seasonId);
          const leagueNames = Array.from(new Set(seasonAwards.map((a) => a.league_name).filter(Boolean)));
          return (
            <div key={seasonId} className="mb-8">
              <h2 className="mb-3 text-xl font-bold text-zinc-300">{t("hallOfFame.season")} #{seasonId}</h2>
              {leagueNames.map((ln) => (
                <div key={ln} className="mb-4 rounded-xl border border-zinc-800 bg-zinc-900 p-4 card-interactive">
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
