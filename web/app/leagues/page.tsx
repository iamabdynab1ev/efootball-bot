"use client";

import Link from "next/link";
import { useState } from "react";
import { motion } from "framer-motion";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronRight, Trophy, UserPlus } from "lucide-react";
import { toast } from "sonner";
import { EmptyState } from "@/components/EmptyState";
import { LeagueStatusBadge } from "@/components/StatusBadge";
import { RegistrationCountdown } from "@/components/RegistrationCountdown";
import { Button } from "@/components/ui/button";
import { SkeletonTable } from "@/components/ui/skeleton";
import { fetchLeagues, fetchMyLeagues, joinLeague } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useLang } from "@/lib/i18n";

const fadeUp = { hidden: { opacity: 0, y: 12 }, show: { opacity: 1, y: 0 } };
const stagger = { show: { transition: { staggerChildren: 0.05 } } };

export default function LeaguesPage() {
  const { user } = useAuth();
  const { t } = useLang();
  const qc = useQueryClient();
  const { data: leagues = [], isLoading } = useQuery({ queryKey: ["leagues"], queryFn: fetchLeagues, staleTime: 30000 });
  const { data: myLeagues = [], isLoading: myLeaguesLoading } = useQuery({ queryKey: ["me", "leagues"], queryFn: fetchMyLeagues, enabled: !!user, staleTime: 30000 });
  const joinedIds  = new Set(myLeagues.filter((m) => m.status === "approved").map((m) => m.league?.id).filter(Boolean));
  const pendingIds = new Set(myLeagues.filter((m) => m.status === "pending").map((m) => m.league?.id).filter(Boolean));

  // Оптимистичный pending: показываем "Ожидание" сразу после клика, не дожидаясь refetch
  const [optimisticPendingId, setOptimisticPendingId] = useState<number | null>(null);

  const joinMutation = useMutation({
    mutationFn: joinLeague,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["me", "leagues"] });
      qc.invalidateQueries({ queryKey: ["leagues"] });
      // Убираем оптимистичный pending когда данные обновятся
    },
    onError: () => {
      setOptimisticPendingId(null);
      toast.error(t("leagues.joinError"));
    },
  });

  const registrationCount = leagues.filter((l) => l.status === "registration").length;
  const activeCount = leagues.filter((l) => l.status === "active").length;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-zinc-500 mb-1">{t("leagues.subtitle")}</p>
          <h1 className="text-2xl font-bold text-zinc-100">{t("leagues.title")}</h1>
        </div>
        <div className="flex items-center gap-4">
          <div className="text-right">
            <p className="text-lg font-black text-yellow-400">{registrationCount}</p>
            <p className="text-[10px] uppercase text-zinc-600 tracking-wide">{t("leagues.registration")}</p>
          </div>
          <div className="text-right">
            <p className="text-lg font-black text-green-400">{activeCount}</p>
            <p className="text-[10px] uppercase text-zinc-600 tracking-wide">{t("common.active")}</p>
          </div>
        </div>
      </div>

      {isLoading ? (
        <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
          <SkeletonTable rows={4} />
        </div>
      ) : leagues.length === 0 ? (
        <div className="rounded-xl border border-zinc-800 bg-zinc-900">
          <EmptyState icon={Trophy} title={t("leagues.noLeagues")} text={t("leagues.noLeaguesText")} />
        </div>
      ) : (
        <motion.div variants={stagger} initial="hidden" animate="show" className="space-y-2">
          {leagues.map((league) => {
            const joined  = joinedIds.has(league.id);
            // Pending = реальный из БД ИЛИ оптимистичный (сразу после клика)
            const pending = pendingIds.has(league.id) || optimisticPendingId === league.id;
            const deadlinePassed = league.registration_deadline
              ? new Date(league.registration_deadline) <= new Date()
              : false;
            // Не показываем кнопку "Заявка" пока данные о членстве не загружены
            const membershipReady = !user || !myLeaguesLoading;
            const canJoin = !!user && membershipReady && league.status === "registration" && !joined && !pending && !deadlinePassed;
            const isRegistration = league.status === "registration";

            // Черновик — показываем как "скоро откроется"
            if (league.status === "draft") {
              return (
                <motion.div key={league.id} variants={fadeUp}
                  className="rounded-xl border border-zinc-700 bg-zinc-900 overflow-hidden"
                >
                  <div className="flex items-center gap-3 px-4 py-4">
                    <Link href={`/leagues/details?id=${league.id}`} className="flex items-center gap-4 flex-1 min-w-0 group">
                      <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-zinc-800 text-zinc-500">
                        <Trophy size={18} aria-hidden="true" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-semibold text-zinc-300 group-hover:text-yellow-400 transition-colors truncate">
                          {league.name}
                        </p>
                        <p className="text-xs text-zinc-500 mt-0.5">
                          ⚡ Hybrid Tournament
                          {league.registration_deadline && (
                            <span> · Набор откроется до {new Date(league.registration_deadline).toLocaleDateString("ru-RU", { day: "numeric", month: "short" })}</span>
                          )}
                        </p>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        <span className="text-[10px] font-bold text-zinc-500 bg-zinc-800 border border-zinc-700 px-2 py-0.5 rounded-full uppercase">
                          Скоро
                        </span>
                        <ChevronRight size={15} className="text-zinc-600 group-hover:text-zinc-400 transition-colors" />
                      </div>
                    </Link>
                  </div>
                </motion.div>
              );
            }

            if (isRegistration) {
              return (
                <motion.div key={league.id} variants={fadeUp}>
                  <RegistrationCountdown
                    league={league}
                    joined={joined}
                    pending={pending}
                    canJoin={canJoin}
                    onJoin={() => {
                      setOptimisticPendingId(league.id);
                      joinMutation.mutate(league.id);
                    }}
                    joining={optimisticPendingId === league.id && joinMutation.isPending}
                  />
                </motion.div>
              );
            }

            return (
              <motion.div key={league.id} variants={fadeUp}
                className="rounded-xl border border-zinc-800 bg-zinc-900"
              >
                <div className="flex items-center gap-3 px-4 py-4">
                  <Link href={`/leagues/details?id=${league.id}`} className="flex items-center gap-4 flex-1 min-w-0 group">
                    <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-yellow-500/10 text-yellow-400">
                      <Trophy size={18} aria-hidden="true" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-semibold text-zinc-100 group-hover:text-yellow-400 transition-colors truncate">
                        {league.name}
                      </p>
                      <p className="text-xs text-zinc-500 mt-0.5">
                        {league.rounds_type === "double" ? t("common.doubleRound") : t("common.singleRound")}
                        {" · "}{league.max_players} {t("common.players")}
                        {joined && <span className="ml-1 text-green-400">· ✓ {t("leagues.member")}</span>}
                      </p>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      <LeagueStatusBadge status={league.status} />
                      <ChevronRight size={15} className="text-zinc-600 group-hover:text-zinc-400 transition-colors" />
                    </div>
                  </Link>
                </div>
              </motion.div>
            );
          })}
        </motion.div>
      )}
    </div>
  );
}
