"use client";

import Link from "next/link";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Bot, CalendarDays, History, Save, ShieldCheck, User } from "lucide-react";
import { toast } from "sonner";
import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { fetchMe, fetchMyHistory, fetchMyLeagues, generateLinkCode, updateMe } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

type ProfileForm = {
  display_name: string;
  team_power?: string;
};

function ratingColor(rating: number) {
  if (rating >= 1200) return "text-yellow-400";
  if (rating >= 1100) return "text-blue-400";
  if (rating >= 1050) return "text-purple-400";
  return "text-green-400";
}

export default function ProfilePage() {
  const router = useRouter();
  const qc = useQueryClient();
  const { user, loading, refreshUser } = useAuth();
  const { t } = useLang();

  const profileSchema = z.object({
    display_name: z.string().min(1, t("validation.nameRequired")).max(64, t("validation.nameMax")),
    team_power: z.string().regex(/^\d*$/, t("validation.numbersOnly")).optional(),
  });

  useEffect(() => {
    if (!loading && !user) router.replace("/login");
  }, [loading, router, user]);

  const { data: me, isLoading } = useQuery({ queryKey: ["me"], queryFn: fetchMe, enabled: !!user });
  const { data: myLeagues = [] } = useQuery({ queryKey: ["me", "leagues"], queryFn: fetchMyLeagues, enabled: !!user });
  const { data: history = [] } = useQuery({ queryKey: ["me", "history"], queryFn: () => fetchMyHistory(), enabled: !!user });

  const { register, handleSubmit, reset, formState: { errors, isDirty } } = useForm<ProfileForm>({
    resolver: zodResolver(profileSchema),
  });

  useEffect(() => {
    if (me) reset({ display_name: me.display_name, team_power: String(me.team_power || "") });
  }, [me, reset]);

  const updateMutation = useMutation({
    mutationFn: (data: ProfileForm) => updateMe({
      display_name: data.display_name.trim(),
      team_power: data.team_power ? Number(data.team_power) : undefined,
    }),
    onSuccess: async () => {
      toast.success(t("profile.saveSuccess"));
      await refreshUser();
      qc.invalidateQueries({ queryKey: ["me"] });
      qc.invalidateQueries({ queryKey: ["players"] });
    },
    onError: () => toast.error(t("profile.saveError")),
  });

  const linkMutation = useMutation({
    mutationFn: generateLinkCode,
    onSuccess: (data) => {
      navigator.clipboard?.writeText(`/link ${data.code}`);
      toast.success(`${t("profile.codeCopied")}: ${data.code}`);
    },
    onError: () => toast.error("Не удалось получить код"),
  });

  if (loading || isLoading || !me) {
    return (
      <div className="space-y-4">
        <div className="h-40 rounded-xl bg-zinc-900 animate-pulse" />
        <div className="h-64 rounded-xl bg-zinc-900 animate-pulse" />
      </div>
    );
  }

  const totalWins   = myLeagues.reduce((s, m) => s + m.wins,   0);
  const totalDraws  = myLeagues.reduce((s, m) => s + m.draws,  0);
  const totalLosses = myLeagues.reduce((s, m) => s + m.losses, 0);
  const totalPoints = myLeagues.reduce((s, m) => s + m.points, 0);

  return (
    <div className="space-y-5">
      <div>
        <p className="text-xs font-semibold uppercase tracking-widest text-zinc-500 mb-1">{t("profile.title")}</p>
        <h1 className="text-2xl font-bold text-zinc-100">{me.display_name}</h1>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Profile card column */}
        <div className="flex flex-col gap-3">
          {/* Main profile card */}
          <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
            <div className="flex items-center gap-4 px-5 pt-5 pb-4 border-b border-zinc-800">
              <div className="flex h-14 w-14 flex-shrink-0 items-center justify-center rounded-full bg-yellow-400 text-zinc-900 text-xl font-black">
                {(me.display_name || "?").slice(0, 1).toUpperCase()}
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-base font-black text-zinc-100 truncate">{me.display_name}</p>
                <p className="text-sm text-zinc-500">{me.rank || "Игрок"}</p>
              </div>
              <div className="text-right">
                <p className={cn("text-2xl font-black tabular-nums", ratingColor(me.rating))}>{me.rating}</p>
                <p className="text-[9px] text-zinc-600 uppercase">ELO</p>
              </div>
            </div>

            {/* W/D/L */}
            <div className="grid grid-cols-3 divide-x divide-zinc-800 border-b border-zinc-800">
              <div className="py-3 text-center">
                <p className="text-lg font-black text-green-400">{totalWins}</p>
                <p className="text-[10px] text-zinc-600 uppercase">{t("profile.wins")}</p>
              </div>
              <div className="py-3 text-center">
                <p className="text-lg font-black text-zinc-400">{totalDraws}</p>
                <p className="text-[10px] text-zinc-600 uppercase">{t("profile.draws")}</p>
              </div>
              <div className="py-3 text-center">
                <p className="text-lg font-black text-red-400">{totalLosses}</p>
                <p className="text-[10px] text-zinc-600 uppercase">{t("profile.losses")}</p>
              </div>
            </div>

            {/* Leagues / Points */}
            <div className="grid grid-cols-2 divide-x divide-zinc-800">
              <div className="py-3 text-center">
                <p className="text-lg font-black text-zinc-100">{myLeagues.length}</p>
                <p className="text-[10px] text-zinc-600 uppercase">{t("profile.leaguesLabel")}</p>
              </div>
              <div className="py-3 text-center">
                <p className="text-lg font-black text-zinc-100">{totalPoints}</p>
                <p className="text-[10px] text-zinc-600 uppercase">{t("profile.pointsLabel")}</p>
              </div>
            </div>
          </div>

          {/* Telegram section */}
          <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-4">
            {me.has_telegram ? (
              <div className="flex items-center gap-3 rounded-lg bg-green-500/10 border border-green-500/20 px-3 py-2.5">
                <ShieldCheck size={16} className="text-green-400 flex-shrink-0" />
                <div>
                  <p className="text-xs font-semibold text-green-300">{t("profile.telegramLinked")}</p>
                  <p className="text-[10px] text-zinc-500">{me.username ? `@${me.username}` : t("profile.notificationsAvailable")}</p>
                </div>
              </div>
            ) : (
              <div className="space-y-2">
                <p className="text-xs text-zinc-500">{t("profile.linkTelegram")}</p>
                <Button variant="outline" size="sm" className="w-full" disabled={linkMutation.isPending}
                  onClick={() => linkMutation.mutate()}
                >
                  <Bot size={14} /> {t("profile.getCode")}
                </Button>
              </div>
            )}
          </div>
        </div>

        {/* Edit form */}
        <div className="lg:col-span-2 rounded-xl border border-zinc-800 bg-zinc-900 p-6">
          <div className="flex items-center gap-2 mb-6">
            <User size={16} className="text-zinc-500" />
            <h2 className="text-sm font-semibold uppercase tracking-wider text-zinc-400">{t("profile.editProfile")}</h2>
          </div>

          <form onSubmit={handleSubmit((data) => updateMutation.mutate(data))} className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-zinc-400">{t("profile.displayName")}</label>
              <Input {...register("display_name")} placeholder={t("profile.displayNamePlaceholder")} />
              {errors.display_name && <p className="text-xs text-red-400">{errors.display_name.message}</p>}
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-zinc-400">{t("profile.teamPower")}</label>
              <Input {...register("team_power")} placeholder={t("profile.teamPowerPlaceholder")} inputMode="numeric" />
              {errors.team_power && <p className="text-xs text-red-400">{errors.team_power.message}</p>}
            </div>

            <Button type="submit" disabled={!isDirty || updateMutation.isPending} className="w-full sm:w-auto">
              <Save size={15} /> {t("profile.saveProfile")}
            </Button>
          </form>
        </div>
      </div>

      {/* My leagues */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
        <div className="flex items-center gap-2 px-4 py-3 border-b border-zinc-800 text-xs font-semibold uppercase tracking-wider text-zinc-400">
          <CalendarDays size={13} /> {t("profile.myLeagues")}
        </div>
        {myLeagues.length === 0 ? (
          <EmptyState
            icon={CalendarDays}
            title={t("profile.notInLeague")}
            text={t("profile.notInLeagueText")}
            action={
              <Button asChild size="sm" variant="outline">
                <Link href="/leagues">{t("profile.openLeagues")}</Link>
              </Button>
            }
          />
        ) : (
          <div>
            {myLeagues.map((membership) => (
              <Link
                key={membership.league?.id}
                href={`/leagues/details?id=${membership.league?.id}`}
                className="flex items-center justify-between gap-3 px-4 py-3 border-b border-zinc-800/50 last:border-0 hover:bg-zinc-800/40 transition-colors"
              >
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-zinc-200 truncate">{membership.league?.name}</p>
                  <p className="text-xs text-zinc-500">{membership.wins}В · {membership.draws}Н · {membership.losses}П</p>
                </div>
                <div className="text-right flex-shrink-0">
                  <p className="text-base font-black text-yellow-400">{membership.points}</p>
                  <p className="text-[10px] text-zinc-600 uppercase">очков</p>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>

      {/* Match history */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
        <div className="flex items-center gap-2 px-4 py-3 border-b border-zinc-800 text-xs font-semibold uppercase tracking-wider text-zinc-400">
          <History size={13} /> {t("profile.history")}
        </div>
        {history.length === 0 ? (
          <EmptyState icon={History} title={t("profile.noHistory")} text={t("profile.noHistoryText")} />
        ) : (
          <div>
            {history.slice(0, 12).map((match) => (
              <div key={match.id} className="flex items-center gap-4 px-4 py-3 border-b border-zinc-800/50 last:border-0">
                <span className="text-xs text-zinc-600 flex-shrink-0">Тур {match.round}</span>
                <span className="flex-1 text-sm font-semibold text-zinc-200 truncate">
                  {match.home_name}{" "}
                  <span className="text-yellow-400">{match.home_goals}:{match.away_goals}</span>
                  {" "}{match.away_name}
                </span>
                <span className="text-xs text-zinc-600 flex-shrink-0">
                  {match.played_at ? new Date(match.played_at).toLocaleDateString("ru-RU") : "подтвержден"}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
