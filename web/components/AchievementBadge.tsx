"use client";

import { UserAchievement } from "@/lib/api";
import { useLang } from "@/lib/i18n";

interface Props {
  achievement: UserAchievement;
}

export function AchievementBadge({ achievement }: Props) {
  const { lang } = useLang();
  const a = achievement.achievement;
  if (!a) return null;
  const name = lang === "ru" ? a.name_ru : lang === "tg" ? a.name_tg : a.name_uz;
  const date = new Date(achievement.earned_at).toLocaleDateString();
  return (
    <div className="flex items-center gap-2 rounded-lg bg-zinc-800/60 px-3 py-2">
      <span className="text-2xl">{a.icon}</span>
      <div className="min-w-0">
        <p className="text-sm font-semibold text-zinc-200 truncate">{name}</p>
        <p className="text-xs text-zinc-500">{date}</p>
      </div>
    </div>
  );
}
