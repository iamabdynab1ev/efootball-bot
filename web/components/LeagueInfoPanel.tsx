"use client";

import { League, Standing, leagueFormatKeys } from "@/lib/api";
import { useLang } from "@/lib/i18n";

interface Props {
  league: League;
  standings: Standing[];
  hasPlayoff: boolean;
}

export function LeagueInfoPanel({ league, standings, hasPlayoff }: Props) {
  const { t } = useLang();
  const totalPlayed = standings.reduce((s, m) => s + m.wins + m.draws + m.losses, 0);
  const totalGoals = standings.reduce((s, m) => s + m.goals_for, 0);
  const fmt = leagueFormatKeys(league.rounds_type);

  return (
    <div className="rounded-xl card-premium p-4 space-y-4">
      {/* Что это за формат — понятным языком */}
      <div className="rounded-lg border border-yellow-500/20 bg-yellow-500/5 px-3 py-2.5">
        <p className="text-xs font-bold uppercase tracking-wide text-yellow-400">{t(fmt.label as never)}</p>
        <p className="mt-1 text-xs leading-relaxed text-zinc-400">{t(fmt.desc as never)}</p>
      </div>
      <div className="grid grid-cols-3 gap-3">
        <Stat label={t("leagueDetail.infoParticipants")} value={standings.length} />
        <Stat label={t("leagueDetail.infoPlayed")} value={totalPlayed} />
        <Stat label={t("leagueDetail.infoGoals")} value={totalGoals} />
      </div>
      {hasPlayoff && (
        <p className="text-xs text-zinc-500">{t("leagueDetail.playoffInCupTab")}</p>
      )}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg bg-zinc-800/60 px-3 py-2 text-center">
      <p className="text-xs text-zinc-500">{label}</p>
      <p className="mt-0.5 text-lg font-black text-zinc-200">{value}</p>
    </div>
  );
}
