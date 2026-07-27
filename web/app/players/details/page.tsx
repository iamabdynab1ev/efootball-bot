"use client";

import { Suspense, useState } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, MessageSquare, Swords, Trophy, Target, Zap } from "lucide-react";
import { toast } from "sonner";
import { openDirect } from "@/lib/chat";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { PlayerCard } from "@/components/PlayerCard";
import { AchievementBadge } from "@/components/AchievementBadge";
import { TrophyCabinet } from "@/components/TrophyCabinet";
import { EloChart } from "@/components/EloChart";
import { EmptyState } from "@/components/EmptyState";
import { SkeletonProfile } from "@/components/ui/skeleton";
import { api, fetchPlayerProfile, fetchHeadToHead } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { pluralize, useLang } from "@/lib/i18n";
import { getClub } from "@/lib/clubs";
import { cn } from "@/lib/utils";

function ratingColor(r: number) {
  if (r >= 1200) return "text-yellow-400";
  if (r >= 1100) return "text-blue-400";
  if (r >= 1050) return "text-purple-400";
  return "text-green-400";
}

function PlayerDetailsContent() {
  const params = useSearchParams();
  const router = useRouter();
  const { user } = useAuth();
  const { t, lang } = useLang();
  const id = Number(params.get("id"));

  const { data: p, isLoading, isError } = useQuery({
    queryKey: ["player-profile", id],
    queryFn: () => fetchPlayerProfile(id),
    enabled: !!id,
    retry: false,
  });
  const isMe = user?.id === id;

  // «Написать» — находит или создаёт личный чат и сразу открывает диалог.
  const [opening, setOpening] = useState(false);
  const writeTo = async () => {
    if (opening) return;
    setOpening(true);
    try {
      const room = await openDirect(id);
      router.push(`/messages?room=${room.id}`);
    } catch {
      toast.error("Не удалось открыть чат — попробуйте ещё раз");
      setOpening(false);
    }
  };

  // «Вызвать на матч» — товарищеский вызов; после отправки ведём в /friendlies.
  const [challenging, setChallenging] = useState(false);
  const challenge = async () => {
    if (challenging) return;
    setChallenging(true);
    try {
      await api.post("/api/friendlies", { opponent_id: id });
      toast.success("Вызов отправлен ⚔️");
      router.push("/friendlies");
    } catch (e: any) {
      toast.error(e?.response?.data?.error ?? "Не удалось отправить вызов");
      setChallenging(false);
    }
  };

  const { data: h2h } = useQuery({
    queryKey: ["h2h", id],
    queryFn: () => fetchHeadToHead(id),
    enabled: !!id && !!user && !isMe,
  });

  if (isLoading) {
    return <div className="space-y-4"><SkeletonProfile /></div>;
  }

  if (isError || !p) {
    return (
      <div className="max-w-2xl mx-auto space-y-4">
        <button
          onClick={() => router.back()}
          className="flex items-center gap-1.5 text-sm text-zinc-400 hover:text-zinc-200 transition-colors"
        >
          <ArrowLeft size={15} /> {t("playerPage.backToPlayers")}
        </button>
        <div className="rounded-xl card-premium">
          <EmptyState icon={Target} title={t("common.error")} text={t("playerPage.title")} />
        </div>
      </div>
    );
  }

  const club = getClub(p.favorite_club);
  const clubName = club ? (lang === "ru" ? club.nameRu || club.name : club.name) : null;
  const matches = p.total_matches ?? 0;
  const wins = p.total_wins ?? 0;
  const draws = p.total_draws ?? 0;
  const losses = p.total_losses ?? 0;
  const winRate = Math.round(p.win_rate ?? 0);

  const stat = (label: string, value: string | number, color = "text-zinc-100") => (
    <div className="rounded-xl card-premium p-3 text-center">
      <p className={cn("text-xl font-black tabular-nums", color)}>{value}</p>
      <p className="text-[10px] uppercase tracking-wide text-zinc-500 mt-0.5">{label}</p>
    </div>
  );

  return (
    <div className="space-y-5 max-w-2xl mx-auto">
      <button
        onClick={() => router.back()}
        className="flex items-center gap-1.5 text-sm text-zinc-400 hover:text-zinc-200 transition-colors"
      >
        <ArrowLeft size={15} /> {t("playerPage.backToPlayers")}
      </button>

      {/* Header */}
      <div className="rounded-2xl border border-zinc-800 overflow-hidden">
        <div
          className="relative px-5 pt-6 pb-5 flex items-center gap-4"
          style={{ background: club ? `linear-gradient(135deg, ${club.color} 0%, ${club.color2} 100%)` : "#18181b" }}
        >
          <div className="relative z-10">
            <PlayerAvatar displayName={p.display_name} favoriteClub={p.favorite_club} size={64} />
          </div>
          <div className="flex-1 min-w-0 relative z-10">
            <h1 className="text-xl font-black text-white truncate drop-shadow">{p.display_name}</h1>
            <p className="text-sm text-white/85 font-medium">{clubName || p.rank}</p>
          </div>
          <div className="text-right relative z-10">
            <p className="text-3xl font-black text-white tabular-nums drop-shadow">{p.rating}</p>
            <p className="text-[10px] uppercase text-white/70">ELO</p>
          </div>
        </div>
        {/* Написать игроку — личный чат создаётся автоматически при первом сообщении */}
        {!isMe && user && (
          <div className="flex divide-x divide-zinc-800">
            <button
              onClick={writeTo}
              disabled={opening}
              className="flex flex-1 items-center justify-center gap-2 bg-zinc-900 px-4 py-3 text-sm font-bold text-yellow-400 transition-colors hover:bg-zinc-800 disabled:opacity-60"
            >
              <MessageSquare size={16} />
              {opening ? "Открываем…" : "Написать"}
            </button>
            <button
              onClick={challenge}
              disabled={challenging}
              className="flex flex-1 items-center justify-center gap-2 bg-zinc-900 px-4 py-3 text-sm font-bold text-orange-400 transition-colors hover:bg-zinc-800 disabled:opacity-60"
            >
              <Swords size={16} />
              {challenging ? "Отправляем…" : "Вызвать на матч"}
            </button>
          </div>
        )}
      </div>

      {/* Stats */}
      {matches === 0 ? (
        <div className="rounded-xl card-premium">
          <EmptyState icon={Target} title={t("playerPage.noStats")} text="" />
        </div>
      ) : (
        <div className="grid grid-cols-3 gap-2">
          {stat(t("playerPage.matches"), matches)}
          {stat(t("playerPage.winRate"), `${winRate}%`, "text-yellow-400")}
          {stat("ELO", p.rating, ratingColor(p.rating))}
          {stat(t("profile.wins"), wins, "text-green-400")}
          {stat(t("profile.draws"), draws, "text-zinc-300")}
          {stat(t("profile.losses"), losses, "text-red-400")}
          {stat(t("playerPage.goalsFor"), p.total_goals_for ?? 0)}
          {stat(t("playerPage.goalsAgainst"), p.total_goals_against ?? 0)}
          {stat(t("common.teamPower"), (p.team_power || 0).toLocaleString())}
        </div>
      )}

      {/* Витрина трофеев — награды за завершённые турниры */}
      <EloChart userId={id} />
      <TrophyCabinet userId={id} playerName={p.display_name} />

      {/* Карточка игрока (рисуется в браузере с логотипом клуба).
          Смотреть могут все, «Поделиться» — только своей. */}
      <div className="rounded-xl card-premium p-4">
        <PlayerCard
          canShare={isMe}
          data={{
            name: p.display_name,
            rank: p.rank,
            rating: p.rating,
            favoriteClub: p.favorite_club,
            wins: p.total_wins ?? 0,
            draws: p.total_draws ?? 0,
            losses: p.total_losses ?? 0,
            goals: p.total_goals_for ?? 0,
            winRate: p.win_rate ?? 0,
          }}
        />
      </div>

      {/* Head-to-head (только для других игроков, если залогинен) */}
      {!isMe && user && (
        <div className="rounded-xl card-premium p-4">
          <div className="flex items-center gap-2 mb-3 text-sm font-bold uppercase tracking-wide text-zinc-300">
            <Swords size={15} className="text-yellow-400" /> {t("h2h.title")}
          </div>
          {!h2h || h2h.played === 0 ? (
            <p className="text-sm text-zinc-500 text-center py-3">{t("h2h.noMatches")}</p>
          ) : (
            <>
              <div className="grid grid-cols-3 gap-2 text-center">
                <div className="rounded-lg bg-green-500/10 border border-green-500/20 py-3">
                  <p className="text-2xl font-black text-green-400">{h2h.my_wins}</p>
                  <p className="text-[10px] uppercase text-zinc-500">{t("h2h.youWin")}</p>
                </div>
                <div className="rounded-lg bg-zinc-800 border border-zinc-700 py-3">
                  <p className="text-2xl font-black text-zinc-300">{h2h.draws}</p>
                  <p className="text-[10px] uppercase text-zinc-500">{t("h2h.draws")}</p>
                </div>
                <div className="rounded-lg bg-red-500/10 border border-red-500/20 py-3">
                  <p className="text-2xl font-black text-red-400">{h2h.opp_wins}</p>
                  <p className="text-[10px] uppercase text-zinc-500">{t("h2h.oppWin")}</p>
                </div>
              </div>
              <p className="text-center text-xs text-zinc-500 mt-3">
                {h2h.played} {lang === "ru" ? pluralize(lang, h2h.played, "матч", "матча", "матчей") : t("h2h.played")}
                {" · "}{t("h2h.goals")} {h2h.my_goals}:{h2h.opp_goals}
              </p>
              <div className="flex items-center justify-center gap-1.5 mt-3">
                {h2h.recent.map((m, i) => (
                  <span
                    key={i}
                    className={cn(
                      "flex h-7 w-7 items-center justify-center rounded-full text-[11px] font-black",
                      m.result === "W" ? "bg-green-500/20 text-green-400" :
                      m.result === "L" ? "bg-red-500/20 text-red-400" : "bg-zinc-700 text-zinc-300"
                    )}
                    title={`${m.my_goals}:${m.opp_goals}`}
                  >
                    {m.result}
                  </span>
                ))}
              </div>
            </>
          )}
        </div>
      )}

      {/* Achievements */}
      {p.achievements && p.achievements.length > 0 && (
        <div className="rounded-xl card-premium overflow-hidden">
          <div className="flex items-center gap-2 px-4 py-3 border-b border-zinc-800 text-xs font-bold uppercase tracking-wider text-zinc-400">
            <Trophy size={13} className="text-yellow-400" /> {t("playerPage.achievements")}
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 p-4">
            {p.achievements.map((a) => <AchievementBadge key={a.id} achievement={a} />)}
          </div>
        </div>
      )}
    </div>
  );
}

export default function PlayerDetailsPage() {
  return (
    <Suspense fallback={<div className="space-y-4"><SkeletonProfile /></div>}>
      <PlayerDetailsContent />
    </Suspense>
  );
}
