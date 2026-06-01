"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangle, Archive, Check, Crown, Gavel, GitBranch, LayoutDashboard,
  Plus, RotateCcw, Search, Shield, Shuffle, Star, Trash2,
  UserCheck, UserMinus, UserPlus, Users, X,
} from "lucide-react";
import { toast } from "sonner";
import { EmptyState } from "@/components/EmptyState";
import { LeagueStatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  adminAdd, adminApprove, adminArchiveLeague, adminCreateLeague,
  adminDraw, adminFetchAdmins, adminFetchDisputed, adminFetchLeagues,
  adminFetchMembers, adminFetchUsers, adminGeneratePlayoff, adminReject, adminRemove,
  adminResetRatings, adminResolve, UserWithRole,
} from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

type Tab = "leagues" | "requests" | "disputes" | "users";

function roleColor(role: UserWithRole["admin_role"]) {
  if (role === "super_admin") return "text-yellow-400 bg-yellow-400/10 border-yellow-400/30";
  if (role === "admin")       return "text-blue-400 bg-blue-400/10 border-blue-400/30";
  return "";
}
// eslint-disable-next-line @typescript-eslint/no-explicit-any
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
  const [selectedLeague, setSelectedLeague] = useState<number | null>(null);
  const [playoffK, setPlayoffK] = useState(8);
  const [resolveMatch, setResolveMatch] = useState<number | null>(null);
  const [homeGoals, setHomeGoals] = useState("");
  const [awayGoals, setAwayGoals] = useState("");
  const [userSearch, setUserSearch] = useState("");

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

  const filteredUsers = useMemo(() => {
    const q = userSearch.toLowerCase();
    return allUsers.filter((u) =>
      !q || u.display_name.toLowerCase().includes(q) || String(u.id).includes(q)
    );
  }, [allUsers, userSearch]);

  const createMutation = useMutation({
    mutationFn: () => adminCreateLeague(newLeague.trim()),
    onSuccess: () => {
      toast.success(t("admin.leagueCreated"));
      setNewLeague("");
      qc.invalidateQueries({ queryKey: ["admin", "leagues"] });
      qc.invalidateQueries({ queryKey: ["leagues"] });
    },
    onError: () => toast.error(t("admin.leagueCreateError")),
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
    mutationFn: () => adminResolve(resolveMatch!, Number(homeGoals), Number(awayGoals), "Решено через web"),
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
  const playoffMutation = useMutation({
    mutationFn: ({ id, k }: { id: number; k: number }) => adminGeneratePlayoff(id, k),
    onSuccess: () => {
      toast.success("Плей-офф сгенерирован!");
      qc.invalidateQueries({ queryKey: ["bracket"] });
    },
    onError: (e: any) => toast.error(e?.response?.data?.error || "Ошибка"),
  });

  if (loading) return (
    <div className="flex items-center justify-center py-20">
      <p className="text-sm text-zinc-500">{t("admin.loading")}</p>
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
    ...(user.is_super_admin ? [{ key: "users" as Tab, label: t("admin.tabUsers"), icon: Users }] : []),
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-zinc-500 mb-1">{t("admin.subtitle")}</p>
          <h1 className="text-2xl font-bold text-zinc-100">{t("admin.title")}</h1>
        </div>
        <div className="flex items-center gap-4">
          <div className="text-right">
            <p className="text-lg font-black text-zinc-100">{leagues.length}</p>
            <p className="text-[10px] uppercase text-zinc-600 tracking-wide">{t("admin.leaguesCount")}</p>
          </div>
          {disputed.length > 0 && (
            <div className="text-right">
              <p className="text-lg font-black text-red-400">{disputed.length}</p>
              <p className="text-[10px] uppercase text-zinc-600 tracking-wide">{t("admin.disputesCount")}</p>
            </div>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1 border-b border-zinc-800 pb-0">
        {TABS.map((t) => {
          const Icon = t.icon;
          return (
            <button key={t.key} onClick={() => setTab(t.key)}
              className={cn(
                "flex items-center gap-2 px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px",
                tab === t.key ? "border-yellow-400 text-yellow-400" : "border-transparent text-zinc-400 hover:text-zinc-200"
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

      {/* ── Leagues ── */}
      {tab === "leagues" && (
        <div className="space-y-4">
          <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-4">
            <h2 className="text-sm font-semibold text-zinc-300 mb-3 flex items-center gap-2">
              <Plus size={15} /> {t("admin.createLeague")}
            </h2>
            <div className="flex gap-2">
              <Input value={newLeague} onChange={(e) => setNewLeague(e.target.value)}
                placeholder={t("admin.leagueName")} className="flex-1"
                onKeyDown={(e) => e.key === "Enter" && newLeague.trim() && createMutation.mutate()}
              />
              <Button disabled={!newLeague.trim() || createMutation.isPending} onClick={() => createMutation.mutate()}>
                <Plus size={15} /> {t("admin.create")}
              </Button>
            </div>
          </div>
          {leagues.length === 0 ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900">
              <EmptyState icon={LayoutDashboard} title="Лиг пока нет" text="Создайте первую лигу выше." />
            </div>
          ) : (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
              {leagues.map((league) => (
                <div key={league.id} className="flex items-center gap-4 px-4 py-3.5 border-b border-zinc-800/50 last:border-0">
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-semibold text-zinc-200">{league.name}</p>
                    <p className="text-xs text-zinc-500">
                      {league.rounds_type === "double" ? t("common.doubleRound") : t("common.singleRound")} · {league.max_players} {t("common.players")}
                    </p>
                  </div>
                  <LeagueStatusBadge status={league.status} />
                  <div className="flex items-center gap-1.5 flex-shrink-0">
                    <Button size="sm" variant="outline" onClick={() => { setSelectedLeague(league.id); setTab("requests"); }}>
                      <Users size={13} /> {t("admin.tabRequests")}
                    </Button>
                    {league.status === "registration" && (
                      <Button size="sm" disabled={drawMutation.isPending} onClick={() => drawMutation.mutate(league.id)}>
                        <Shuffle size={13} /> {t("admin.draw")}
                      </Button>
                    )}
                    {league.status === "active" && (
                      <div className="flex items-center gap-1">
                        <select
                          value={playoffK}
                          onChange={(e) => setPlayoffK(Number(e.target.value))}
                          className="h-8 rounded-md border border-zinc-700 bg-zinc-800 px-1.5 text-xs text-zinc-200 focus:outline-none"
                        >
                          {[4, 8, 16].map((k) => (
                            <option key={k} value={k}>Топ-{k}</option>
                          ))}
                        </select>
                        <Button size="sm" variant="outline"
                          disabled={playoffMutation.isPending}
                          onClick={() => playoffMutation.mutate({ id: league.id, k: playoffK })}
                        >
                          <GitBranch size={13} /> Плей-офф
                        </Button>
                      </div>
                    )}
                    <Button size="sm" variant="destructive" className="h-8 w-8 p-0"
                      disabled={archiveMutation.isPending} onClick={() => archiveMutation.mutate(league.id)}>
                      <Archive size={13} />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
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
          <select value={selectedLeague ?? ""} onChange={(e) => setSelectedLeague(e.target.value ? Number(e.target.value) : null)}
            className="rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 focus:outline-none focus:ring-2 focus:ring-yellow-500/40"
          >
            <option value="">Выберите лигу</option>
            {leagues.map((l) => <option key={l.id} value={l.id}>{l.name}</option>)}
          </select>
          {!selectedLeague ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900">
              <EmptyState icon={UserPlus} title="Лига не выбрана" text="Выберите лигу для просмотра заявок." />
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
                  <EmptyState icon={UserPlus} title="Новых заявок нет" text="" />
                ) : (
                  <div>
                    {members.pending.map((member) => (
                      <div key={member.user_id} className="flex items-center gap-3 px-4 py-3 border-b border-zinc-800/50 last:border-0">
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-semibold text-zinc-200 truncate">{member.display_name || `User #${member.user_id}`}</p>
                          <p className="text-xs text-zinc-500">{member.rank || "заявка"} · {member.rating ?? 1000} ELO</p>
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
                  <EmptyState icon={Users} title="Участников пока нет" text="" />
                ) : (
                  <div>
                    {members.approved.map((member) => (
                      <div key={member.user_id} className="flex items-center gap-3 px-4 py-3 border-b border-zinc-800/50 last:border-0">
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-semibold text-zinc-200 truncate">{member.display_name || `User #${member.user_id}`}</p>
                          <p className="text-xs text-zinc-500">{member.points} очк. · {member.wins + member.draws + member.losses} игр</p>
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
                  <span className="text-xs font-semibold text-red-400">{t("admin.resolve")} #{match.dispute_count}</span>
                </div>
                <p className="text-sm font-bold text-zinc-200">{match.home_name} vs {match.away_name}</p>
                <p className="text-xs text-zinc-500">Тур {match.round} · матч #{match.id}</p>
              </div>
              {resolveMatch === match.id ? (
                <div className="flex items-center gap-2 pt-2 border-t border-zinc-800">
                  <Input value={homeGoals} onChange={(e) => setHomeGoals(e.target.value.replace(/\D/g, ""))} placeholder="ГЗ" className="w-16 text-center" />
                  <span className="text-zinc-600 font-bold">:</span>
                  <Input value={awayGoals} onChange={(e) => setAwayGoals(e.target.value.replace(/\D/g, ""))} placeholder="ГП" className="w-16 text-center" />
                  <Button disabled={!homeGoals || !awayGoals || resolveMutation.isPending} onClick={() => resolveMutation.mutate()}>
                    <Check size={14} /> {t("admin.resolveBtn")}
                  </Button>
                  <Button variant="ghost" onClick={() => setResolveMatch(null)}>{t("common.cancel")}</Button>
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
                        title="Снять роль"
                      >
                        <UserMinus size={14} />
                      </button>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Search */}
          <div className="relative">
            <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500" />
            <input
              value={userSearch}
              onChange={(e) => setUserSearch(e.target.value)}
              placeholder={t("admin.searchPlaceholder")}
              className="w-full rounded-lg border border-zinc-700 bg-zinc-900 py-2 pl-9 pr-3 text-sm text-zinc-200 placeholder-zinc-600 outline-none focus:border-yellow-500 focus:ring-1 focus:ring-yellow-500/30"
            />
          </div>

          {/* User cards grid */}
          {filteredUsers.length === 0 ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900">
              <EmptyState icon={Users} title="Пользователей нет" text="Зарегистрированные игроки появятся здесь." />
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
                    {/* Avatar */}
                    <div className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-full text-base font-black text-white shadow"
                      style={{ background: `radial-gradient(circle at 40% 35%, ${g.border}60, #0c1525)`, border: `2px solid ${g.border}50` }}
                    >
                      {(u.display_name || "?").slice(0, 1).toUpperCase()}
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
                      <div className="flex items-center gap-2 mt-0.5">
                        <span className="text-xs font-black" style={{ color: g.border }}>{u.rating}</span>
                        <span className="text-[10px] text-zinc-600">{u.rank || "ELO"}</span>
                        {u.has_telegram && (
                          <span className="text-[9px] text-blue-400 font-semibold">{t("admin.tgYes")}</span>
                        )}
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
