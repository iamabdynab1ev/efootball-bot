"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangle, Archive, Bell, BellOff, Check, Crown, Gavel, GitBranch, LayoutDashboard,
  Megaphone, Pencil, Plus, RotateCcw, ScrollText, Search, Shield, Shuffle, Star, Trash2,
  UserCheck, UserMinus, UserPlus, Users, Wifi, X,
} from "lucide-react";
import { AuditPanel } from "./AuditPanel";
import { IntegrationsPanel } from "./IntegrationsPanel";
import { usePresence } from "@/lib/presence";
import { toast } from "sonner";
import { EmptyState } from "@/components/EmptyState";
import { LeagueStatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import {
  adminAdd, adminApprove, adminArchiveLeague, adminCreateLeague,
  adminDraw, adminOpenLeague, adminFetchAdmins, adminFetchDisputed, adminFetchLeagues,
  adminFetchMembers, adminFetchUsers, adminGeneratePlayoff, adminPurgeLeague, adminDeleteUser,
  adminReject, adminRemove, adminResetRatings, adminResolve, adminUpdateLeague,
  adminGetDeadlines, adminSetDeadline, adminDeleteDeadline, adminFinalizeLeague,
  fetchLeagueProgress, fetchBracket, fetchPlayoffOptions, League, UserWithRole, RoundDeadline, PlayoffOptions,
  stageLabelKey,
} from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

type Tab = "leagues" | "requests" | "disputes" | "users" | "audit" | "integrations";

function LeaguePicker({ leagues, selected, onSelect, t }: { leagues: League[]; selected: number | null; onSelect: (id: number) => void; t: (k: any) => string }) {
  if (leagues.length === 0) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-900">
        <EmptyState icon={UserPlus} title={t("admin.noActiveLeagues")} text={t("admin.noActiveLeaguesText")} />
      </div>
    );
  }
  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-3">
      <p className="text-xs font-semibold uppercase tracking-wider text-zinc-400 mb-2.5 px-1">{t("admin.selectLeague")}</p>
      <div className="flex flex-wrap gap-2">
        {leagues.map((l) => (
          <button
            key={l.id}
            onClick={() => onSelect(l.id)}
            className={cn(
              "flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-semibold transition-all border",
              selected === l.id
                ? "bg-yellow-500 border-yellow-500 text-zinc-900"
                : "border-zinc-600 bg-zinc-800 text-zinc-200 hover:border-zinc-400 hover:text-white"
            )}
          >
            <span>{l.name}</span>
            <LeagueStatusBadge status={l.status} />
          </button>
        ))}
      </div>
    </div>
  );
}

function roleColor(role: UserWithRole["admin_role"]) {
  if (role === "super_admin") return "text-yellow-400 bg-yellow-400/10 border-yellow-400/30";
  if (role === "admin")       return "text-blue-400 bg-blue-400/10 border-blue-400/30";
  return "";
}
function roleLabel(role: UserWithRole["admin_role"], t: (k: any) => string) {
  if (role === "super_admin") return t("admin.roleSuperAdmin");
  if (role === "admin")       return t("admin.roleAdmin");
  return "";
}
function cardGradient(rating: number) {
  if (rating >= 1200) return { border: "#f0b90b", bg: "from-[#7c5a00] to-[#0c1525]" };
  if (rating >= 1100) return { border: "#4a90d9", bg: "from-[#1a3a6c] to-[#0c1525]" };
  if (rating >= 1050) return { border: "#9b59b6", bg: "from-[#2a1a4a] to-[#0c1525]" };
  return { border: "#3a7a3a", bg: "from-[#1a2f1a] to-[#0c1525]" };
}

export default function AdminPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const { user, loading } = useAuth();
  const { t } = useLang();
  const [tab, setTab] = useState<Tab>("leagues");
  const [newLeague, setNewLeague] = useState("");
  const [newDeadline, setNewDeadline] = useState("");
  const [newRoundsType, setNewRoundsType] = useState<"hybrid" | "single" | "double" | "groups" | "cup" | "swiss" | "nations_league" | "double_elim">("hybrid");
  const [hybridK, setHybridK] = useState(4); // мин. размер группы
  const [hybridP, setHybridP] = useState(4); // выходят из группы
  const [deBestOf, setDeBestOf] = useState(1); // длина серии для double_elim
  const [newNumGroups, setNewNumGroups] = useState(4);
  const [newGroupAdvance, setNewGroupAdvance] = useState(1);
  const [newBestRunnersUp, setNewBestRunnersUp] = useState(0);
  const [selectedLeague, setSelectedLeague] = useState<number | null>(null);
  const [playoffAdvance, setPlayoffAdvance] = useState<Record<number, number>>({});
  const [playoffRandom, setPlayoffRandom] = useState<Record<number, boolean>>({});
  const [resolveMatch, setResolveMatch] = useState<number | null>(null);
  const [homeGoals, setHomeGoals] = useState("");
  const [awayGoals, setAwayGoals] = useState("");
  const [editLeagueId, setEditLeagueId] = useState<number | null>(null);
  const [editName, setEditName] = useState("");
  const [editDeadline, setEditDeadline] = useState("");
  const [userSearch, setUserSearch] = useState("");
  const [deadlineLeagueId, setDeadlineLeagueId] = useState<number | null>(null);
  const [deadlineRound, setDeadlineRound] = useState(1);
  const [deadlineValue, setDeadlineValue] = useState("");

  useEffect(() => {
    if (!loading && !user) router.replace("/login");
  }, [loading, router, user]);

  const enabled = !!user?.is_admin;
  const { data: leagues = [] }  = useQuery({ queryKey: ["admin", "leagues"],  queryFn: adminFetchLeagues,  enabled });
  const { data: disputed = [] } = useQuery({ queryKey: ["admin", "disputed"], queryFn: adminFetchDisputed, enabled });
  const { data: admins = [] }   = useQuery({ queryKey: ["admin", "admins"],   queryFn: adminFetchAdmins,   enabled });
  const { data: allUsers = [] } = useQuery({ queryKey: ["admin", "users"],    queryFn: adminFetchUsers,    enabled: enabled && !!user?.is_super_admin });
  const { data: members } = useQuery({
    queryKey: ["admin", "members", selectedLeague],
    queryFn: () => adminFetchMembers(selectedLeague!),
    enabled: enabled && !!selectedLeague,
  });
  const { data: deadlines = [], refetch: refetchDeadlines } = useQuery({
    queryKey: ["admin", "deadlines", deadlineLeagueId],
    queryFn: () => adminGetDeadlines(deadlineLeagueId!),
    enabled: enabled && !!deadlineLeagueId,
  });

  // Прогресс матчей для каждой активной лиги (нужен для блокировки кнопки плей-офф)
  const activeLeagueIds = useMemo(
    () => leagues.filter((l) => l.status === "active").map((l) => l.id),
    [leagues]
  );
  const progressQueries = useQueries({
    queries: activeLeagueIds.map((id) => ({
      queryKey: ["league-progress", id],
      queryFn: () => fetchLeagueProgress(id),
      enabled,
      staleTime: 30000,
    })),
  });
  const remainingByLeague = useMemo(() => {
    const map: Record<number, number> = {};
    activeLeagueIds.forEach((id, i) => {
      map[id] = progressQueries[i]?.data?.remaining ?? 1;
    });
    return map;
  }, [activeLeagueIds, progressQueries]);

  // Проверяем существование плей-офф для каждой активной лиги
  const bracketQueries = useQueries({
    queries: activeLeagueIds.map((id) => ({
      queryKey: ["bracket", id],
      queryFn: () => fetchBracket(id),
      enabled,
      staleTime: 60000,
    })),
  });
  const bracketByLeague = useMemo(() => {
    const map: Record<number, boolean> = {};
    activeLeagueIds.forEach((id, i) => {
      map[id] = (bracketQueries[i]?.data?.length ?? 0) > 0;
    });
    return map;
  }, [activeLeagueIds, bracketQueries]);

  // Лиги, готовые к генерации плей-офф (все матчи группового этапа сыграны, бракет ещё не создан)
  const readyForPlayoffIds = useMemo(
    () => activeLeagueIds.filter((id) => (remainingByLeague[id] ?? 1) === 0 && !bracketByLeague[id]),
    [activeLeagueIds, remainingByLeague, bracketByLeague]
  );
  const playoffOptionsQueries = useQueries({
    queries: readyForPlayoffIds.map((id) => ({
      queryKey: ["playoff-options", id],
      queryFn: () => fetchPlayoffOptions(id),
      enabled,
      staleTime: 30000,
    })),
  });
  const playoffOptionsByLeague = useMemo(() => {
    const map: Record<number, PlayoffOptions | undefined> = {};
    readyForPlayoffIds.forEach((id, i) => {
      map[id] = playoffOptionsQueries[i]?.data;
    });
    return map;
  }, [readyForPlayoffIds, playoffOptionsQueries]);

  const filteredUsers = useMemo(() => {
    const q = userSearch.toLowerCase();
    return allUsers.filter((u) =>
      !q || u.display_name.toLowerCase().includes(q) || String(u.id).includes(q)
    );
  }, [allUsers, userSearch]);

  // Presence в реальном времени (initial снимок + SSE-дельты).
  const { isOnline, online } = usePresence();
  const onlineCount = online.size;
  const pushCount = useMemo(() => allUsers.filter((u) => u.push_enabled).length, [allUsers]);

  const createMutation = useMutation({
    mutationFn: () => {
      const dl = newDeadline ? new Date(newDeadline).toISOString() : undefined;
      // Для hybrid передаём K и P как num_groups и group_advance (система их использует в CalculateHybrid)
      return adminCreateLeague(
        newLeague.trim(), dl, newRoundsType,
        newRoundsType === "hybrid" ? hybridK :
        newRoundsType === "groups" || newRoundsType === "nations_league" ? newNumGroups : undefined,
        newRoundsType === "hybrid" ? hybridP :
        newRoundsType === "groups" ? newGroupAdvance : undefined,
        newRoundsType === "groups" && newBestRunnersUp > 0 ? newBestRunnersUp : undefined,
        newRoundsType === "double_elim" ? deBestOf : undefined,
      );
    },
    onSuccess: () => {
      toast.success(t("admin.leagueCreated"));
      setNewLeague("");
      setNewDeadline("");
      setNewRoundsType("hybrid");
      qc.invalidateQueries({ queryKey: ["admin", "leagues"] });
      qc.invalidateQueries({ queryKey: ["leagues"] });
    },
    onError: () => toast.error(t("admin.leagueCreateError")),
  });
  const openMutation = useMutation({
    mutationFn: adminOpenLeague,
    onSuccess: () => {
      toast.success(t("admin.openSuccess"));
      qc.invalidateQueries({ queryKey: ["admin", "leagues"] });
      qc.invalidateQueries({ queryKey: ["leagues"] });
    },
    onError: () => toast.error(t("admin.openError")),
  });
  const archiveMutation = useMutation({
    mutationFn: adminArchiveLeague,
    onSuccess: () => {
      toast.success(t("admin.archiveSuccess"));
      qc.invalidateQueries({ queryKey: ["admin", "leagues"] });
    },
  });
  const drawMutation = useMutation({
    mutationFn: adminDraw,
    onSuccess: () => {
      toast.success(t("admin.drawSuccess"));
      qc.invalidateQueries({ queryKey: ["admin", "leagues"] });
    },
    onError: () => toast.error(t("admin.drawError")),
  });
  const approveMutation = useMutation({
    mutationFn: ({ leagueId, userId }: { leagueId: number; userId: number }) => adminApprove(leagueId, userId),
    onSuccess: () => { toast.success(t("admin.approveSuccess")); qc.invalidateQueries({ queryKey: ["admin", "members", selectedLeague] }); },
  });
  const rejectMutation = useMutation({
    mutationFn: ({ leagueId, userId }: { leagueId: number; userId: number }) => adminReject(leagueId, userId),
    onSuccess: () => { toast.success(t("admin.rejectSuccess")); qc.invalidateQueries({ queryKey: ["admin", "members", selectedLeague] }); },
  });
  const resolveMutation = useMutation({
    mutationFn: () => adminResolve(resolveMatch!, Number(homeGoals), Number(awayGoals), "web"),
    onSuccess: () => {
      toast.success(t("admin.resolveSuccess"));
      setResolveMatch(null); setHomeGoals(""); setAwayGoals("");
      qc.invalidateQueries({ queryKey: ["admin", "disputed"] });
    },
  });
  const addAdminMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: number; role: "admin" | "super_admin" }) =>
      adminAdd({ user_id: userId, role }),
    onSuccess: () => {
      toast.success(t("admin.roleAssigned"));
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
      qc.invalidateQueries({ queryKey: ["admin", "admins"] });
    },
    onError: () => toast.error(t("common.error")),
  });
  const removeAdminMutation = useMutation({
    mutationFn: (userId: number) => adminRemove(userId),
    onSuccess: () => {
      toast.success(t("admin.roleRemoved"));
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
      qc.invalidateQueries({ queryKey: ["admin", "admins"] });
    },
  });
  const resetMutation = useMutation({
    mutationFn: adminResetRatings,
    onSuccess: () => toast.success(t("admin.resetSuccess")),
  });
  const deleteUserMutation = useMutation({
    mutationFn: (userId: number) => adminDeleteUser(userId),
    onSuccess: () => {
      toast.success(t("admin.userDeleted"));
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
      qc.invalidateQueries({ queryKey: ["players"] });
    },
    onError: (e: any) => toast.error(e?.response?.data?.error || t("common.error")),
  });
  const playoffMutation = useMutation({
    mutationFn: ({ id, group_advance, random_draw }: { id: number; group_advance?: number; random_draw?: boolean }) =>
      adminGeneratePlayoff(id, group_advance ? { group_advance, random_draw } : { top_k: 8, random_draw }),
    onSuccess: () => {
      toast.success(t("admin.playoffSuccess"));
      qc.invalidateQueries({ queryKey: ["bracket"] });
    },
    onError: (e: any) => toast.error(e?.response?.data?.error || t("common.error")),
  });

  const updateLeagueMutation = useMutation({
    mutationFn: ({ id, name, deadline }: { id: number; name: string; deadline: string }) =>
      adminUpdateLeague(id, { name, registration_deadline: deadline || undefined }),
    onSuccess: () => {
      toast.success(t("admin.editLeagueSuccess"));
      setEditLeagueId(null);
      qc.invalidateQueries({ queryKey: ["admin", "leagues"] });
      qc.invalidateQueries({ queryKey: ["leagues"] });
    },
    onError: () => toast.error(t("admin.editLeagueError")),
  });

  const purgeMutation = useMutation({
    mutationFn: adminPurgeLeague,
    onSuccess: () => {
      toast.success(t("admin.purgeLeagueSuccess"));
      qc.invalidateQueries({ queryKey: ["admin", "leagues"] });
      qc.invalidateQueries({ queryKey: ["leagues"] });
    },
    onError: () => toast.error(t("admin.purgeLeagueError")),
  });

  const setDeadlineMutation = useMutation({
    mutationFn: ({ leagueId, round, deadline }: { leagueId: number; round: number; deadline: string }) =>
      adminSetDeadline(leagueId, round, new Date(deadline).toISOString()),
    onSuccess: () => {
      toast.success(t("admin.deadlineSaved"));
      setDeadlineValue("");
      refetchDeadlines();
    },
    onError: () => toast.error(t("common.error")),
  });

  const deleteDeadlineMutation = useMutation({
    mutationFn: ({ leagueId, round }: { leagueId: number; round: number }) =>
      adminDeleteDeadline(leagueId, round),
    onSuccess: () => {
      toast.success(t("admin.deadlineDeleted"));
      refetchDeadlines();
    },
    onError: () => toast.error(t("common.error")),
  });

  const finalizeMutation = useMutation({
    mutationFn: (leagueId: number) => adminFinalizeLeague(leagueId),
    onSuccess: () => {
      toast.success(t("admin.finalizeSuccess"));
      qc.invalidateQueries({ queryKey: ["admin", "leagues"] });
      qc.invalidateQueries({ queryKey: ["leagues"] });
    },
    onError: (e: any) => toast.error(e?.response?.data?.error || t("common.error")),
  });

  // Скелетон повторяет реальную структуру вкладки «Лиги» с теми же высотами —
  // при окончании загрузки контент встаёт на место без сдвига макета (CLS).
  if (loading) return (
    <div className="space-y-6 min-h-screen">
      <div className="space-y-2">
        <div className="h-3.5 w-28 rounded bg-zinc-800 animate-pulse" />
        <div className="h-7 w-44 rounded bg-zinc-800 animate-pulse" />
      </div>
      <div className="flex gap-1 border-b border-zinc-700 pb-2">
        {[1, 2, 3].map(i => <div key={i} className="h-8 w-28 rounded bg-zinc-800 animate-pulse" />)}
      </div>
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-4 space-y-3">
        <div className="h-5 w-40 rounded bg-zinc-800 animate-pulse" />
        <div className="h-9 w-full rounded bg-zinc-800 animate-pulse" />
        <div className="h-44 w-full rounded-xl bg-zinc-800/60 animate-pulse" />
        <div className="h-8 w-full rounded bg-zinc-800 animate-pulse" />
      </div>
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden min-h-[200px]">
        {[1, 2, 3].map(i => <div key={i} className="h-16 border-b border-zinc-800/50 last:border-0 bg-zinc-900 animate-pulse" />)}
      </div>
    </div>
  );

  if (!user?.is_admin) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-900">
        <EmptyState icon={Shield} title={t("admin.noAccess")} text={t("admin.noAccessText")} />
      </div>
    );
  }

  const TABS = [
    { key: "leagues"  as Tab, label: t("admin.tabLeagues"),  icon: LayoutDashboard },
    { key: "requests" as Tab, label: t("admin.tabRequests"), icon: UserPlus },
    { key: "disputes" as Tab, label: t("admin.tabDisputes"), icon: Gavel, count: disputed.length },
    { key: "audit" as Tab, label: "Аудит", icon: ScrollText },
    { key: "integrations" as Tab, label: "Интеграции", icon: Megaphone },
    ...(user.is_super_admin ? [{ key: "users" as Tab, label: t("admin.tabUsers"), icon: Users }] : []),
  ];

  return (
    <div className="space-y-6 min-h-screen">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-zinc-400 mb-1">{t("admin.subtitle")}</p>
          <h1 className="font-display text-2xl font-bold text-zinc-100">{t("admin.title")}</h1>
        </div>
        <div className="flex items-center gap-4 flex-wrap">
          <div className="text-right">
            <p className="flex items-center justify-end gap-1 text-lg font-black text-green-400">
              <span className="inline-block h-2 w-2 rounded-full bg-green-400 animate-pulse" />
              {onlineCount}
            </p>
            <p className="text-[10px] uppercase text-zinc-400 tracking-wide">В сети</p>
          </div>
          <div className="text-right">
            <p className="text-lg font-black text-zinc-100">{leagues.length}</p>
            <p className="text-[10px] uppercase text-zinc-400 tracking-wide">{t("admin.leaguesCount")}</p>
          </div>
          {disputed.length > 0 && (
            <div className="text-right">
              <p className="text-lg font-black text-red-400">{disputed.length}</p>
              <p className="text-[10px] uppercase text-zinc-400 tracking-wide">{t("admin.disputesCount")}</p>
            </div>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="relative -mx-4">
      {/* Правый фейд — подсказка, что таб-бар скроллится по горизонтали
          (на мобиле часть табов уходит за край). На desktop все влезают. */}
      <div className="pointer-events-none absolute inset-y-0 right-0 z-10 w-10 bg-gradient-to-l from-zinc-950 to-transparent md:hidden" />
      <div className="flex items-center gap-1 border-b border-zinc-700 pb-0 overflow-x-auto scrollbar-none px-4">
        {TABS.map((t) => {
          const Icon = t.icon;
          return (
            <button key={t.key} onClick={() => setTab(t.key)}
              className={cn(
                "flex flex-shrink-0 items-center gap-2 px-3 sm:px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px whitespace-nowrap",
                tab === t.key ? "border-yellow-400 text-yellow-400" : "border-transparent text-zinc-300 hover:text-white"
              )}
            >
              <Icon size={15} />
              {t.label}
              {"count" in t && t.count ? (
                <span className="flex h-4 min-w-[16px] items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white">
                  {t.count}
                </span>
              ) : null}
            </button>
          );
        })}
      </div>
      </div>

      {/* ── Leagues ── */}
      {tab === "leagues" && (
        <div className="space-y-4">
          <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-4">
            <h2 className="text-sm font-semibold text-zinc-300 mb-3 flex items-center gap-2">
              <Plus size={15} /> {t("admin.createLeague")}
            </h2>
            <div className="space-y-2">
              <div className="flex gap-2">
                <Input value={newLeague} onChange={(e) => setNewLeague(e.target.value)}
                  placeholder={t("admin.leagueName")} className="flex-1 min-w-0"
                  onKeyDown={(e) => e.key === "Enter" && newLeague.trim() && createMutation.mutate()}
                />
                <Button className="flex-shrink-0" disabled={!newLeague.trim() || createMutation.isPending} onClick={() => createMutation.mutate()}>
                  <Plus size={15} /> {t("admin.create")}
                </Button>
              </div>
              {/* Формат турнира */}
              <div className="flex items-center gap-2">
                <label htmlFor="rounds-type" className="text-xs text-zinc-400 flex-shrink-0">{t("admin.formatLabel")}:</label>
                <Select
                  id="rounds-type"
                  value={newRoundsType}
                  onChange={(v) => setNewRoundsType(v as typeof newRoundsType)}
                  className="h-8"
                  containerClassName="flex-1 min-w-0"
                  options={[
                    { value: "hybrid", label: "⚡ Гибрид (группы + плей-офф)" },
                    { value: "double_elim", label: `🔁 ${t("admin.formatDoubleElim")}` },
                  ]}
                />
              </div>

              {/* Hybrid-параметры */}
              {newRoundsType === "hybrid" && (
              <div className="rounded-xl border border-yellow-500/30 bg-yellow-500/5 px-4 py-3 space-y-3">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-black text-yellow-400 uppercase tracking-wide">⚡ Гибридный формат</span>
                  <span className="text-[9px] text-zinc-400 bg-zinc-800 px-1.5 py-0.5 rounded">автоматические группы + плей-офф</span>
                </div>
                <p className="text-[10px] text-zinc-400 leading-relaxed">
                  Система автоматически делит команды на равные группы, проводит круговой этап,
                  отбирает лучших в плей-офф с кросс-посевом (1A vs 4B, 2A vs 3B…)
                </p>
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <label htmlFor="hybrid-k" className="text-[10px] font-semibold text-zinc-400">
                      K — мин. размер группы
                    </label>
                    <Select id="hybrid-k" value={String(hybridK)} onChange={(v) => setHybridK(Number(v))}
                      ariaLabel="Минимальный размер группы" className="h-8"
                      options={[3, 4, 5, 6, 7, 8].map(n => ({ value: String(n), label: n === 4 ? `${n} — лучший` : String(n) }))}
                    />
                    <p className="text-[9px] text-zinc-400">Минимум команд в одной группе</p>
                  </div>
                  <div className="space-y-1">
                    <label htmlFor="hybrid-p" className="text-[10px] font-semibold text-zinc-400">
                      P — выходят в плей-офф
                    </label>
                    <Select id="hybrid-p" value={String(hybridP)} onChange={(v) => setHybridP(Number(v))}
                      ariaLabel="Команд из группы в плей-офф" className="h-8"
                      options={[2, 3, 4, 6, 8].map(n => ({ value: String(n), label: n === 4 ? `${n} — лучший` : String(n) }))}
                    />
                    <p className="text-[9px] text-zinc-400">Команд из каждой группы в плей-офф</p>
                  </div>
                </div>
              </div>
              )}

              {/* Double elimination — пояснение + длина серии */}
              {newRoundsType === "double_elim" && (
                <div className="rounded-xl border border-amber-500/30 bg-amber-500/5 px-4 py-3 space-y-3">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-xs font-black text-amber-400 uppercase tracking-wide">🔁 {t("admin.formatDoubleElim")}</span>
                  </div>
                  <p className="text-[10px] text-zinc-400 leading-relaxed">{t("admin.doubleElimNote")}</p>
                  <div className="space-y-1">
                    <label htmlFor="de-bestof" className="text-[10px] font-semibold text-zinc-400">{t("admin.seriesLength")}</label>
                    <Select id="de-bestof" value={String(deBestOf)} onChange={(v) => setDeBestOf(Number(v))}
                      className="h-8" accent="amber"
                      options={[
                        { value: "1", label: t("admin.seriesSingle") as string },
                        { value: "3", label: "Best-of-3" },
                        { value: "5", label: "Best-of-5" },
                        { value: "7", label: "Best-of-7" },
                      ]}
                    />
                  </div>
                </div>
              )}
              <div className="flex items-center gap-2">
                <label className="text-xs text-zinc-400 flex-shrink-0">{t("admin.deadlineLabel")}:</label>
                <input
                  type="datetime-local"
                  value={newDeadline}
                  onChange={(e) => setNewDeadline(e.target.value)}
                  className="flex-1 rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-yellow-500 focus:ring-1 focus:ring-yellow-500/30 transition-colors"
                />
              </div>
            </div>
          </div>
          {leagues.length === 0 ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900">
              <EmptyState icon={LayoutDashboard} title={t("admin.noLeagues")} text={t("admin.noLeaguesText")} />
            </div>
          ) : (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden min-h-[200px]">
              {leagues.map((league) => (
                <div key={league.id} className="border-b border-zinc-800/50 last:border-0">
                  {/* ── Карточка лиги — мобильный дизайн ── */}
                  <div className="p-4 space-y-3">

                    {/* Строка 1: Название + статус */}
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-bold text-zinc-100 leading-tight">{league.name}</p>
                        <p className="text-[11px] text-zinc-400 mt-0.5">
                          {({
                            hybrid: "⚡ Hybrid", single: "Один круг", double: "Двойной круг",
                            cup: "🏆 Кубок", groups: "🗂 Группы+ПО",
                            swiss: "🔄 Швейцарская", nations_league: "🌍 Лига Наций",
                            double_elim: "🔁 Двойная элиминация",
                          } as Record<string,string>)[league.rounds_type] ?? league.rounds_type}
                          {" · "}{league.players_count ?? 0}/{league.max_players} игроков
                        </p>
                        {league.registration_deadline && (
                          <p className="text-[11px] text-zinc-500 mt-0.5">
                            📅 Набор до: {new Date(league.registration_deadline).toLocaleString("ru-RU", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" })}
                          </p>
                        )}
                      </div>
                      <div className="flex-shrink-0 flex flex-col items-end gap-1.5">
                        <LeagueStatusBadge status={league.status} />
                        {/* Плей-офф сгенерирован */}
                        {bracketByLeague[league.id] && (
                          <span className="flex items-center gap-1 text-[10px] font-bold text-green-400 bg-green-500/10 border border-green-500/20 px-2 py-0.5 rounded-full">
                            <GitBranch size={10} /> Плей-офф ✓
                          </span>
                        )}
                      </div>
                    </div>

                    {/* Строка 2: Основные действия */}
                    <div className="flex flex-wrap gap-1.5">
                      {/* draft → открыть набор */}
                      {league.status === "draft" && (
                        <Button size="sm" className="bg-emerald-600 hover:bg-emerald-500 text-white border-0 flex-1 sm:flex-none"
                          disabled={openMutation.isPending}
                          onClick={() => openMutation.mutate(league.id)}
                        >
                          <UserPlus size={13} /> Открыть набор
                        </Button>
                      )}

                      {/* registration → жеребьёвка */}
                      {league.status === "registration" && (
                        <Button size="sm" className="flex-1 sm:flex-none"
                          disabled={drawMutation.isPending}
                          onClick={() => { if (confirm(t("admin.drawConfirm"))) drawMutation.mutate(league.id); }}
                        >
                          <Shuffle size={13} /> Жеребьёвка
                        </Button>
                      )}

                      {/* active → плей-офф */}
                      {league.status === "active" && !bracketByLeague[league.id] && (() => {
                        const remaining = remainingByLeague[league.id] ?? 1;
                        const allDone   = remaining === 0;
                        const options   = playoffOptionsByLeague[league.id];
                        const hasGroups = (options?.groups.length ?? 0) > 0;
                        const advance   = playoffAdvance[league.id] ?? options?.advance_default;
                        const cleanOptions = options?.options ?? [];
                        const stageName = (st: string) => {
                          const k = stageLabelKey(st);
                          return k ? (t(`leagueDetail.${k}` as never) as string) : st;
                        };
                        return (
                          <>
                            {allDone && hasGroups && options && (
                              cleanOptions.length > 0 ? (
                                // Ровные варианты сетки: 1/16, 1/8, 1/4… (степень двойки)
                                <Select
                                  value={String(advance ?? options.advance_default)}
                                  onChange={(v) => setPlayoffAdvance((prev) => ({ ...prev, [league.id]: Number(v) }))}
                                  ariaLabel="Формат сетки плей-офф"
                                  className="h-9"
                                  containerClassName="w-full sm:w-64"
                                  options={cleanOptions.map((o) => ({
                                    value: String(o.advance),
                                    label: `${stageName(o.stage)} · ${o.advance} из группы · ${o.qualifiers} команд`,
                                  }))}
                                />
                              ) : (
                                <Select
                                  value={String(advance ?? options.advance_default)}
                                  onChange={(v) => setPlayoffAdvance((prev) => ({ ...prev, [league.id]: Number(v) }))}
                                  ariaLabel="Команд из группы в плей-офф"
                                  className="h-9"
                                  containerClassName="w-full sm:w-48"
                                  options={Array.from(
                                    { length: options.advance_max - options.advance_min + 1 },
                                    (_, i) => options.advance_min + i
                                  ).map((n) => ({ value: String(n), label: `Из группы: ${n}` }))}
                                />
                              )
                            )}
                            {allDone && hasGroups && (
                              <label className="flex items-center gap-1.5 text-xs text-zinc-300 cursor-pointer select-none">
                                <input
                                  type="checkbox"
                                  checked={!!playoffRandom[league.id]}
                                  onChange={(e) => setPlayoffRandom((prev) => ({ ...prev, [league.id]: e.target.checked }))}
                                  className="h-3.5 w-3.5 accent-yellow-400"
                                />
                                {t("admin.randomDraw")}
                              </label>
                            )}
                            <Button size="sm" variant="outline"
                              className="flex-1 sm:flex-none"
                              disabled={playoffMutation.isPending || !allDone}
                              title={allDone ? t("admin.playoffReady") : t("admin.playoffNotReady").replace("{{n}}", String(remaining))}
                              onClick={() => playoffMutation.mutate({ id: league.id, group_advance: hasGroups ? advance : undefined, random_draw: !!playoffRandom[league.id] })}
                            >
                              <GitBranch size={13} />
                              {allDone ? t("admin.playoffBtn") : `${t("admin.playoffBtn")} (${remaining})`}
                            </Button>
                          </>
                        );
                      })()}

                      {/* Заявки */}
                      <Button size="sm" variant="outline"
                        className="flex-1 sm:flex-none"
                        onClick={() => { setSelectedLeague(league.id); setTab("requests"); }}
                      >
                        <Users size={13} /> Заявки
                      </Button>
                    </div>

                    {/* Строка 3: Второстепенные действия */}
                    <div className="flex items-center gap-1.5 flex-wrap">
                      {user.is_super_admin && league.status !== "archived" && (
                        <Button size="sm" variant="outline" className="h-8 gap-1 text-xs"
                          onClick={() => {
                            if (editLeagueId === league.id) {
                              setEditLeagueId(null);
                            } else {
                              setEditLeagueId(league.id);
                              setEditName(league.name);
                              const dl = league.registration_deadline;
                              setEditDeadline(dl ? dl.slice(0, 16) : "");
                            }
                          }}
                        >
                          <Pencil size={12} /> Изменить
                        </Button>
                      )}
                      {league.status !== "archived" && (
                        <Button size="sm" variant="outline" className="h-8 gap-1 text-xs text-zinc-500 border-zinc-700"
                          disabled={archiveMutation.isPending}
                          onClick={() => { if (confirm(t("admin.archiveConfirm"))) archiveMutation.mutate(league.id); }}
                        >
                          <Archive size={12} /> Архив
                        </Button>
                      )}
                      {user.is_super_admin && league.status === "archived" && (
                        <Button size="sm" variant="destructive" className="h-8 gap-1 text-xs"
                          disabled={purgeMutation.isPending}
                          onClick={() => { if (confirm(t("admin.purgeLeagueConfirm"))) purgeMutation.mutate(league.id); }}
                        >
                          <Trash2 size={12} /> Удалить
                        </Button>
                      )}
                    </div>
                  </div>

                                    {/* ── Inline форма редактирования ── */}
                  {editLeagueId === league.id && (
                    <div className="px-4 pb-3.5 pt-0 border-t border-zinc-800/60 bg-zinc-800/20">
                      <p className="text-[10px] font-bold uppercase tracking-wider text-zinc-400 mb-2.5 pt-3">
                        {t("admin.editLeague")}
                      </p>
                      <div className="flex flex-col sm:flex-row gap-2">
                        <Input
                          value={editName}
                          onChange={(e) => setEditName(e.target.value)}
                          placeholder={t("admin.editLeagueName")}
                          className="flex-1"
                          onKeyDown={(e) => {
                            if (e.key === "Enter" && editName.trim()) {
                              updateLeagueMutation.mutate({ id: league.id, name: editName.trim(), deadline: editDeadline });
                            }
                            if (e.key === "Escape") setEditLeagueId(null);
                          }}
                        />
                        <input
                          type="datetime-local"
                          value={editDeadline}
                          onChange={(e) => setEditDeadline(e.target.value)}
                          className="flex-1 rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-yellow-500 focus:ring-1 focus:ring-yellow-500/30 transition-colors"
                        />
                        <Button
                          size="sm"
                          disabled={!editName.trim() || updateLeagueMutation.isPending}
                          onClick={() => updateLeagueMutation.mutate({ id: league.id, name: editName.trim(), deadline: editDeadline })}
                        >
                          <Check size={13} /> {t("admin.editLeagueSave")}
                        </Button>
                        <Button size="sm" variant="ghost" onClick={() => setEditLeagueId(null)}>
                          <X size={13} /> {t("common.cancel")}
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
          {/* ── Round Deadlines ── */}
          <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-4 space-y-3">
            <h2 className="text-sm font-semibold text-zinc-300 flex items-center gap-2">
              <span>⏰</span> Дедлайны туров
            </h2>
            {/* League selector */}
            <div className="flex flex-wrap gap-2">
              {leagues.filter((l) => l.status !== "archived").map((l) => (
                <button
                  key={l.id}
                  onClick={() => setDeadlineLeagueId(deadlineLeagueId === l.id ? null : l.id)}
                  className={cn(
                    "rounded-lg px-3 py-1.5 text-xs font-semibold border transition-all",
                    deadlineLeagueId === l.id
                      ? "bg-yellow-500 border-yellow-500 text-zinc-900"
                      : "border-zinc-600 bg-zinc-800 text-zinc-200 hover:border-zinc-400"
                  )}
                >
                  {l.name}
                </button>
              ))}
            </div>

            {deadlineLeagueId && (
              <div className="space-y-3">
                {/* Existing deadlines */}
                {deadlines.length > 0 && (
                  <div className="space-y-1.5">
                    {deadlines.map((d: RoundDeadline) => (
                      <div key={d.id} className="flex items-center justify-between rounded-lg bg-zinc-800/50 px-3 py-2">
                        <span className="text-sm text-zinc-300">
                          Тур {d.round} — {new Date(d.deadline).toLocaleString("ru-RU")}
                        </span>
                        <button
                          onClick={() => deleteDeadlineMutation.mutate({ leagueId: deadlineLeagueId, round: d.round })}
                          disabled={deleteDeadlineMutation.isPending}
                          className="text-zinc-500 hover:text-red-400 transition-colors p-1"
                        >
                          <X size={14} />
                        </button>
                      </div>
                    ))}
                  </div>
                )}

                {/* Add deadline form */}
                <div className="flex flex-wrap gap-2 items-end">
                  <div className="flex flex-col gap-1">
                    <label className="text-[10px] text-zinc-400">Тур</label>
                    <Select
                      value={String(deadlineRound)}
                      onChange={(v) => setDeadlineRound(Number(v))}
                      ariaLabel="Номер тура"
                      className="h-8"
                      containerClassName="w-20"
                      options={Array.from({ length: 30 }, (_, i) => i + 1).map((n) => ({ value: String(n), label: String(n) }))}
                    />
                  </div>
                  <div className="flex flex-col gap-1 flex-1">
                    <label className="text-[10px] text-zinc-400">Дата и время</label>
                    <input
                      type="datetime-local"
                      value={deadlineValue}
                      onChange={(e) => setDeadlineValue(e.target.value)}
                      className="h-8 rounded-md border border-zinc-700 bg-zinc-800 px-3 text-xs text-zinc-100 outline-none focus:border-yellow-500 transition-colors"
                    />
                  </div>
                  <Button
                    size="sm"
                    disabled={!deadlineValue || setDeadlineMutation.isPending}
                    onClick={() => setDeadlineMutation.mutate({ leagueId: deadlineLeagueId, round: deadlineRound, deadline: deadlineValue })}
                  >
                    Сохранить
                  </Button>
                </div>

                {/* Finalize league */}
                <div className="flex justify-end pt-1 border-t border-zinc-800/60">
                  <Button
                    size="sm"
                    variant="destructive"
                    disabled={finalizeMutation.isPending}
                    onClick={() => { if (confirm(t("admin.finalizeConfirm"))) finalizeMutation.mutate(deadlineLeagueId); }}
                  >
                    {t("admin.finalizeLeague")}
                  </Button>
                </div>
              </div>
            )}
          </div>

          {user.is_super_admin && (
            <div className="flex justify-end">
              <Button variant="destructive" size="sm" disabled={resetMutation.isPending}
                onClick={() => { if (confirm(t("admin.resetConfirm"))) resetMutation.mutate(); }}>
                <RotateCcw size={14} /> {t("admin.resetRatings")}
              </Button>
            </div>
          )}
        </div>
      )}

      {/* ── Requests ── */}
      {tab === "requests" && (
        <div className="space-y-4">
          {/* League picker — только не-архивные */}
          <LeaguePicker
            leagues={leagues.filter((l) => l.status !== "archived")}
            selected={selectedLeague}
            onSelect={setSelectedLeague}
            t={t}
          />

          {!selectedLeague ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900">
              <EmptyState icon={UserPlus} title={t("admin.leagueNotSelected")} text={t("admin.leagueNotSelectedText")} />
            </div>
          ) : (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
                <div className="flex items-center gap-2 px-4 py-3 border-b border-zinc-800 text-xs font-semibold uppercase tracking-wider text-zinc-400">
                  <UserPlus size={13} /> {t("admin.pending")}
                  {!!members?.pending.length && (
                    <span className="ml-auto flex h-5 min-w-[20px] items-center justify-center rounded-full bg-amber-500/20 px-1.5 text-amber-400 text-[10px] font-bold">
                      {members.pending.length}
                    </span>
                  )}
                </div>
                {!members?.pending.length ? (
                  <EmptyState icon={UserPlus} title={t("admin.noPendingRequests")} text="" />
                ) : (
                  <div>
                    {members.pending.map((member) => (
                      <div key={member.user_id} className="flex items-center gap-3 px-4 py-3 border-b border-zinc-800/50 last:border-0">
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-semibold text-zinc-200 truncate">{member.display_name || `User #${member.user_id}`}</p>
                          <p className="text-xs text-zinc-400">{member.rank || t("admin.requestStatus")} · {member.rating ?? 1000} ELO</p>
                        </div>
                        <div className="flex gap-1.5">
                          <Button size="sm" className="h-7 w-7 p-0 rounded-md"
                            onClick={() => approveMutation.mutate({ leagueId: selectedLeague!, userId: member.user_id })}>
                            <Check size={14} />
                          </Button>
                          <Button size="sm" variant="destructive" className="h-7 w-7 p-0 rounded-md"
                            onClick={() => rejectMutation.mutate({ leagueId: selectedLeague!, userId: member.user_id })}>
                            <X size={14} />
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
              <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
                <div className="flex items-center gap-2 px-4 py-3 border-b border-zinc-800 text-xs font-semibold uppercase tracking-wider text-zinc-400">
                  <Check size={13} /> {t("admin.approved")}
                </div>
                {!members?.approved.length ? (
                  <EmptyState icon={Users} title={t("admin.noMembersYet")} text="" />
                ) : (
                  <div>
                    {members.approved.map((member) => (
                      <div key={member.user_id} className="flex items-center gap-3 px-4 py-3 border-b border-zinc-800/50 last:border-0">
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-semibold text-zinc-200 truncate">{member.display_name || `User #${member.user_id}`}</p>
                          <p className="text-xs text-zinc-400">{member.points} {t("admin.memberPoints")} · {member.wins + member.draws + member.losses} {t("admin.memberGames")}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {/* ── Disputes ── */}
      {tab === "disputes" && (
        <div className="space-y-3">
          {disputed.length === 0 ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900">
              <EmptyState icon={Gavel} title={t("admin.noDisputes")} text={t("admin.noDisputesText")} />
            </div>
          ) : disputed.map((match) => (
            <div key={match.id} className="rounded-xl border border-zinc-800 bg-zinc-900 p-4 space-y-3">
              <div>
                <div className="flex items-center gap-2 mb-1">
                  <AlertTriangle size={14} className="text-red-400" />
                  <span className="text-xs font-semibold text-red-400">
                    {t("admin.resolve")}
                    {match.dispute_count > 1 ? ` ×${match.dispute_count}` : ""}
                  </span>
                </div>
                <p className="text-sm font-bold text-zinc-200">{match.home_name} vs {match.away_name}</p>
                <p className="text-xs text-zinc-400">Тур {match.round} · матч #{match.id}</p>
              </div>
              {resolveMatch === match.id ? (
                <div className="flex flex-wrap items-center gap-2 pt-2 border-t border-zinc-800">
                  <div className="flex items-center gap-2">
                    <Input
                      type="number" min="0" max="50"
                      value={homeGoals}
                      onChange={(e) => setHomeGoals(e.target.value.replace(/[^0-9]/g, ""))}
                      placeholder={t("admin.homeGoals")}
                      className="w-16 text-center"
                    />
                    <span className="text-zinc-400 font-bold">:</span>
                    <Input
                      type="number" min="0" max="50"
                      value={awayGoals}
                      onChange={(e) => setAwayGoals(e.target.value.replace(/[^0-9]/g, ""))}
                      placeholder={t("admin.awayGoals")}
                      className="w-16 text-center"
                    />
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      disabled={
                        homeGoals === "" || awayGoals === "" ||
                        Number(homeGoals) < 0 || Number(awayGoals) < 0 ||
                        resolveMutation.isPending
                      }
                      onClick={() => resolveMutation.mutate()}
                    >
                      <Check size={14} /> {t("admin.resolveBtn")}
                    </Button>
                    <Button variant="ghost" onClick={() => setResolveMatch(null)}>{t("common.cancel")}</Button>
                  </div>
                </div>
              ) : (
                <Button size="sm" variant="outline" onClick={() => { setResolveMatch(match.id); setHomeGoals(String(match.claimed_home ?? "")); setAwayGoals(String(match.claimed_away ?? "")); }}>
                  <Gavel size={14} /> {t("admin.resolve")}
                </Button>
              )}
            </div>
          ))}
        </div>
      )}

      {/* ── Users (super_admin only) ── */}
      {tab === "audit" && <AuditPanel />}

      {/* ── Integrations (групповые уведомления) ── */}
      {tab === "integrations" && <IntegrationsPanel />}

      {tab === "users" && (
        <div className="space-y-4">
          {/* Current admins strip */}
          {admins.length > 0 && (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
              <div className="flex items-center gap-2 px-4 py-3 border-b border-zinc-800 text-xs font-semibold uppercase tracking-wider text-zinc-400">
                <Crown size={13} /> {t("admin.admins")}
              </div>
              <div>
                {admins.map((admin) => (
                  <div key={admin.id} className="flex items-center gap-3 px-4 py-2.5 border-b border-zinc-800/50 last:border-0">
                    <div className="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-full bg-yellow-500/10 text-yellow-400 text-xs font-bold">
                      {(admin.display_name || "?").slice(0, 1).toUpperCase()}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-semibold text-zinc-200 truncate">{admin.display_name || `User #${admin.user_id}`}</p>
                    </div>
                    <span className={cn("rounded-full border px-2 py-0.5 text-[10px] font-bold uppercase",
                      admin.role === "super_admin" ? "text-yellow-400 bg-yellow-400/10 border-yellow-400/30" : "text-blue-400 bg-blue-400/10 border-blue-400/30"
                    )}>
                      {roleLabel(admin.role as UserWithRole["admin_role"], t)}
                    </span>
                    {admin.user_id !== user.id && (
                      <button
                        onClick={() => removeAdminMutation.mutate(admin.user_id)}
                        disabled={removeAdminMutation.isPending}
                        className="flex-shrink-0 rounded-md p-1.5 text-zinc-600 hover:bg-red-500/10 hover:text-red-400 transition-colors"
                        title={t("admin.removeRole")}
                      >
                        <UserMinus size={14} />
                      </button>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Сводка: онлайн / push / всего */}
          <div className="grid grid-cols-3 gap-2">
            <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-3 py-2.5">
              <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-wide text-zinc-400">
                <Wifi size={12} className="text-green-400" /> В сети
              </div>
              <p className="mt-0.5 text-xl font-black text-green-400">{onlineCount}</p>
            </div>
            <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-3 py-2.5">
              <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-wide text-zinc-400">
                <Bell size={12} className="text-yellow-400" /> C push
              </div>
              <p className="mt-0.5 text-xl font-black text-zinc-100">{pushCount}</p>
            </div>
            <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-3 py-2.5">
              <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-wide text-zinc-400">
                <Users size={12} /> Всего
              </div>
              <p className="mt-0.5 text-xl font-black text-zinc-100">{allUsers.length}</p>
            </div>
          </div>

          {/* Search */}
          <div className="relative">
            <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500" />
            <input
              id="user-search"
              aria-label={t("admin.searchPlaceholder")}
              value={userSearch}
              onChange={(e) => setUserSearch(e.target.value)}
              placeholder={t("admin.searchPlaceholder")}
              className="w-full rounded-lg border border-zinc-700 bg-zinc-900 py-2 pl-9 pr-3 text-sm text-zinc-200 placeholder-zinc-600 outline-none focus:border-yellow-500 focus:ring-1 focus:ring-yellow-500/30"
            />
          </div>

          {/* User cards grid */}
          {filteredUsers.length === 0 ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900">
              <EmptyState icon={Users} title={t("admin.noUsers")} text={t("admin.noUsersText")} />
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
              {filteredUsers.map((u) => {
                const g = cardGradient(u.rating);
                const isCurrentUser = u.id === user.id;
                return (
                  <div key={u.id}
                    className={cn(
                      "relative rounded-xl border bg-zinc-900 p-4 flex items-center gap-3 transition-colors",
                      u.admin_role ? "border-yellow-500/20 bg-yellow-500/5" : "border-zinc-800 hover:border-zinc-700"
                    )}
                  >
                    {/* Avatar + индикатор онлайн */}
                    <div className="relative flex-shrink-0">
                      <div className="flex h-11 w-11 items-center justify-center rounded-full text-base font-black text-white shadow"
                        style={{ background: `radial-gradient(circle at 40% 35%, ${g.border}60, #0c1525)`, border: `2px solid ${g.border}50` }}
                      >
                        {(u.display_name || "?").slice(0, 1).toUpperCase()}
                      </div>
                      <span
                        title={isOnline(u.id) ? "В сети" : "Не в сети"}
                        className={cn(
                          "absolute -bottom-0.5 -right-0.5 h-3.5 w-3.5 rounded-full border-2 border-zinc-900",
                          isOnline(u.id) ? "bg-green-400" : "bg-zinc-600"
                        )}
                      />
                    </div>

                    {/* Info */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <p className="text-sm font-bold text-zinc-100 truncate">{u.display_name}</p>
                        {u.admin_role && (
                          <span className={cn("rounded-full border px-1.5 py-px text-[9px] font-bold uppercase flex-shrink-0", roleColor(u.admin_role))}>
                            {roleLabel(u.admin_role, t)}
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-2 mt-0.5 flex-wrap">
                        <span className="text-xs font-black" style={{ color: g.border }}>{u.rating}</span>
                        <span className="text-[10px] text-zinc-400">{u.rank || "ELO"}</span>
                        {u.has_telegram && (
                          <span className="text-[9px] text-blue-400 font-semibold">{t("admin.tgYes")}</span>
                        )}
                        <span
                          title={u.push_enabled ? "Push-уведомления включены" : "Push-уведомления выключены"}
                          className={cn(
                            "inline-flex items-center gap-0.5 text-[9px] font-semibold",
                            u.push_enabled ? "text-green-400" : "text-zinc-600"
                          )}
                        >
                          {u.push_enabled ? <Bell size={10} /> : <BellOff size={10} />}
                          {u.push_enabled ? "Push" : "Нет push"}
                        </span>
                      </div>
                    </div>

                    {/* Actions */}
                    {!isCurrentUser && (
                      <div className="flex flex-col gap-1 flex-shrink-0">
                        {u.admin_role === "" && (
                          <button
                            onClick={() => addAdminMutation.mutate({ userId: u.id, role: "admin" })}
                            disabled={addAdminMutation.isPending}
                            className="flex items-center gap-1 rounded-md px-2 py-1 text-[10px] font-bold text-blue-400 bg-blue-400/10 hover:bg-blue-400/20 transition-colors border border-blue-400/20"
                            title={t("admin.makeAdmin")}
                          >
                            <UserCheck size={11} /> {t("admin.roleAdmin")}
                          </button>
                        )}
                        {u.admin_role === "admin" && (
                          <>
                            <button
                              onClick={() => addAdminMutation.mutate({ userId: u.id, role: "super_admin" })}
                              disabled={addAdminMutation.isPending}
                              className="flex items-center gap-1 rounded-md px-2 py-1 text-[10px] font-bold text-yellow-400 bg-yellow-400/10 hover:bg-yellow-400/20 transition-colors border border-yellow-400/20"
                              title={t("admin.makeSuperAdmin")}
                            >
                              <Star size={11} /> {t("admin.roleSuperAdmin")}
                            </button>
                            <button
                              onClick={() => removeAdminMutation.mutate(u.id)}
                              disabled={removeAdminMutation.isPending}
                              className="flex items-center gap-1 rounded-md px-2 py-1 text-[10px] font-bold text-red-400 bg-red-400/10 hover:bg-red-400/20 transition-colors border border-red-400/20"
                              title={t("admin.removeRole")}
                            >
                              <UserMinus size={11} /> {t("admin.removeRole")}
                            </button>
                          </>
                        )}
                        {u.admin_role === "super_admin" && (
                          <button
                            onClick={() => removeAdminMutation.mutate(u.id)}
                            disabled={removeAdminMutation.isPending}
                            className="flex items-center gap-1 rounded-md px-2 py-1 text-[10px] font-bold text-red-400 bg-red-400/10 hover:bg-red-400/20 transition-colors border border-red-400/20"
                            title={t("admin.removeRole")}
                          >
                            <UserMinus size={11} /> {t("admin.removeRole")}
                          </button>
                        )}
                        {/* Полное удаление игрока (супер-админа удалить нельзя) */}
                        {u.admin_role !== "super_admin" && (
                          <button
                            onClick={() => {
                              if (window.confirm(`${t("admin.deleteUserConfirm")} «${u.display_name}»?`)) {
                                deleteUserMutation.mutate(u.id);
                              }
                            }}
                            disabled={deleteUserMutation.isPending}
                            className="flex items-center gap-1 rounded-md px-2 py-1 text-[10px] font-bold text-red-300 bg-red-600/15 hover:bg-red-600/25 transition-colors border border-red-600/30"
                            title={t("admin.deleteUser")}
                          >
                            <Trash2 size={11} /> {t("admin.deleteUser")}
                          </button>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
