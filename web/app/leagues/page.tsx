"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronRight, Trophy, UserPlus } from "lucide-react";
import { toast } from "sonner";
import { EmptyState } from "@/components/EmptyState";
import { LeagueStatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import { SkeletonTable } from "@/components/ui/skeleton";
import { fetchLeagues, fetchMyLeagues, joinLeague } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const fadeUp = { hidden: { opacity: 0, y: 12 }, show: { opacity: 1, y: 0 } };
const stagger = { show: { transition: { staggerChildren: 0.05 } } };

export default function LeaguesPage() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const { data: leagues = [], isLoading } = useQuery({ queryKey: ["leagues"], queryFn: fetchLeagues });
  const { data: myLeagues = [] } = useQuery({ queryKey: ["me", "leagues"], queryFn: fetchMyLeagues, enabled: !!user });
  const joinedIds = new Set(myLeagues.map((m) => m.league?.id).filter(Boolean));

  const joinMutation = useMutation({
    mutationFn: joinLeague,
    onSuccess: () => {
      toast.success("Заявка подана! Ожидайте одобрения.");
      qc.invalidateQueries({ queryKey: ["me", "leagues"] });
      qc.invalidateQueries({ queryKey: ["leagues"] });
    },
    onError: () => toast.error("Не удалось подать заявку"),
  });

  const registrationCount = leagues.filter((l) => l.status === "registration").length;
  const activeCount = leagues.filter((l) => l.status === "active").length;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-zinc-500 mb-1">Соревнования</p>
          <h1 className="text-2xl font-bold text-zinc-100">Лиги</h1>
        </div>
        <div className="flex items-center gap-4">
          <div className="text-right">
            <p className="text-lg font-black text-yellow-400">{registrationCount}</p>
            <p className="text-[10px] uppercase text-zinc-600 tracking-wide">набор</p>
          </div>
          <div className="text-right">
            <p className="text-lg font-black text-green-400">{activeCount}</p>
            <p className="text-[10px] uppercase text-zinc-600 tracking-wide">активных</p>
          </div>
        </div>
      </div>

      {isLoading ? (
        <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
          <SkeletonTable rows={4} />
        </div>
      ) : leagues.length === 0 ? (
        <div className="rounded-xl border border-zinc-800 bg-zinc-900">
          <EmptyState icon={Trophy} title="Лиг пока нет" text="Администратор должен создать лигу, затем игроки смогут подавать заявки." />
        </div>
      ) : (
        <motion.div variants={stagger} initial="hidden" animate="show"
          className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden"
        >
          {leagues.map((league) => {
            const joined = joinedIds.has(league.id);
            const canJoin = !!user && league.status === "registration" && !joined;

            return (
              <motion.div key={league.id} variants={fadeUp} className="border-b border-zinc-800/50 last:border-0">
                <div className="flex items-center gap-3 px-4 py-4">
                  <Link href={`/leagues/details?id=${league.id}`} className="flex items-center gap-4 flex-1 min-w-0 group">
                    <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-yellow-500/10 text-yellow-500">
                      <Trophy size={18} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-semibold text-zinc-100 group-hover:text-yellow-400 transition-colors truncate">
                        {league.name}
                      </p>
                      <p className="text-xs text-zinc-500 mt-0.5">
                        {league.rounds_type === "double" ? "Двойной круг" : "Один круг"}
                        {" · "}{league.max_players} игроков
                        {joined && <span className="ml-1 text-green-400">· ✓ участник</span>}
                      </p>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      <LeagueStatusBadge status={league.status} />
                      <ChevronRight size={15} className="text-zinc-600 group-hover:text-zinc-400 transition-colors" />
                    </div>
                  </Link>

                  {canJoin && (
                    <Button size="sm" className="flex-shrink-0" disabled={joinMutation.isPending}
                      onClick={() => joinMutation.mutate(league.id)}
                    >
                      <UserPlus size={14} /> Заявка
                    </Button>
                  )}

                  {!user && league.status === "registration" && (
                    <Button asChild size="sm" variant="outline" className="flex-shrink-0">
                      <Link href="/login"><UserPlus size={14} /> Войти</Link>
                    </Button>
                  )}
                </div>
              </motion.div>
            );
          })}
        </motion.div>
      )}
    </div>
  );
}
