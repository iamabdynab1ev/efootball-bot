"use client";

import Link from "next/link";
import { useEffect, useState, useMemo } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { CalendarDays, History, Save, User, Search, X, ChevronDown } from "lucide-react";
import { toast } from "sonner";
import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";
import { SkeletonProfile } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { fetchMe, fetchMyHistory, fetchMyLeagues, fetchClubs, generateLinkCode, updateMe, deleteMe, fetchPlayerProfile, Club, UserAchievement } from "@/lib/api";
import { AchievementBadge } from "@/components/AchievementBadge";
import { TrophyCabinet } from "@/components/TrophyCabinet";
import { useAuth } from "@/lib/auth";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { getClub } from "@/lib/clubs";

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

function isLightColor(hex: string) {
  const c = hex.replace("#", "");
  const r = parseInt(c.substring(0, 2), 16);
  const g = parseInt(c.substring(2, 4), 16);
  const b = parseInt(c.substring(4, 6), 16);
  return (r * 299 + g * 587 + b * 114) / 1000 > 160;
}

// ─── Club Logo ────────────────────────────────────────────────────────────────

function ClubLogo({ club, size = 32 }: { club: Club; size?: number }) {
  const [err, setErr] = useState(false);

  // Используем тот же источник логотипов что и PlayerAvatar
  const clubInfo = getClub(club.id);

  // Национальная сборная — флаг emoji
  if (clubInfo?.isNational) {
    return (
      <span style={{ fontSize: size * 0.82, lineHeight: 1, flexShrink: 0 }}>
        {clubInfo.logo}
      </span>
    );
  }

  // Клуб с URL логотипа
  if (clubInfo?.logoUrl && !err) {
    return (
      <img
        src={clubInfo.logoUrl}
        alt={club.name}
        width={size}
        height={size}
        loading="lazy"
        decoding="async"
        style={{ objectFit: "contain", flexShrink: 0, width: size, height: size }}
        onError={() => setErr(true)}
      />
    );
  }

  // Фолбэк — цветной кружок с инициалами
  return (
    <div
      style={{ width: size, height: size, backgroundColor: club.color, flexShrink: 0 }}
      className="rounded-full flex items-center justify-center"
    >
      <span style={{ fontSize: size * 0.35, fontWeight: 900, color: isLightColor(club.color) ? "#000" : "#fff" }}>
        {club.name.slice(0, 2).toUpperCase()}
      </span>
    </div>
  );
}

// ─── Club Selector Modal ──────────────────────────────────────────────────────

const REGIONS = ["europe", "south-america", "asia", "africa", "north-america"];

function useRegionLabel() {
  const { t } = useLang();
  return (r: string) => ({
    "europe": t("profile.regionEurope"),
    "south-america": t("profile.regionSouthAmerica"),
    "asia": t("profile.regionAsia"),
    "africa": t("profile.regionAfrica"),
    "north-america": t("profile.regionNorthAmerica"),
  }[r] ?? r);
}

interface ClubSelectorProps {
  clubs: Club[];
  current?: string;
  onSelect: (club: Club) => void;
  onClose: () => void;
}

function ClubSelector({ clubs, current, onSelect, onClose }: ClubSelectorProps) {
  const { t, lang } = useLang();
  const regionLabel = useRegionLabel();
  const [search, setSearch] = useState("");
  const [typeFilter, setTypeFilter] = useState<"all" | "club" | "national">("all");
  const [regionFilter, setRegionFilter] = useState("all");

  // Нормализуем регион: в данных он вида "South America", а фильтр — "south-america".
  const normRegion = (r?: string) => (r ?? "").toLowerCase().replace(/\s+/g, "-");

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return clubs.filter((c) => {
      if (typeFilter !== "all" && c.type !== typeFilter) return false;
      if (regionFilter !== "all" && normRegion(c.region) !== regionFilter) return false;
      if (!q) return true;
      const ru = (c.name_ru || "").toLowerCase();
      const en = (c.name || "").toLowerCase();
      return ru.includes(q) || en.includes(q);
    });
  }, [clubs, search, typeFilter, regionFilter, lang]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/70"
      onClick={onClose}
    >
      <div
        className="w-full sm:max-w-lg bg-zinc-950 border border-zinc-800 rounded-t-2xl sm:rounded-2xl flex flex-col overflow-hidden"
        style={{ maxHeight: "85vh" }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-800">
          <h2 className="text-sm font-semibold text-zinc-200">{t("profile.selectClub")}</h2>
          <button onClick={onClose} aria-label="Закрыть выбор клуба" className="text-zinc-500 hover:text-zinc-300 transition-colors">
            <X size={18} aria-hidden="true" />
          </button>
        </div>

        {/* Search */}
        <div className="px-4 pt-3 pb-2">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500" />
            <input
              className="w-full bg-zinc-900 border border-zinc-800 rounded-lg pl-8 pr-3 py-2 text-sm text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-zinc-600"
              placeholder={t("profile.searchClub")}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              autoFocus
            />
          </div>
        </div>

        {/* Filters */}
        <div className="px-4 pb-3 space-y-2">
          {/* Type row — 3 equal buttons */}
          <div className="grid grid-cols-3 gap-1.5">
            {(["all", "club", "national"] as const).map((type) => (
              <button
                key={type}
                onClick={() => setTypeFilter(type)}
                className={cn(
                  "text-xs py-1.5 rounded-lg border font-medium transition-colors",
                  typeFilter === type
                    ? "bg-yellow-400 border-yellow-400 text-zinc-900 font-semibold"
                    : "bg-zinc-800 border-zinc-700 text-zinc-300 hover:bg-zinc-700 hover:border-zinc-500 hover:text-white"
                )}
              >
                {type === "all" ? t("profile.typeAll") : type === "club" ? t("profile.typeClubs") : t("profile.typeNational")}
              </button>
            ))}
          </div>
          {/* Region row — scrollable */}
          <div className="flex gap-1.5 overflow-x-auto" style={{ scrollbarWidth: "none" }}>
            {["all", ...REGIONS].map((region) => (
              <button
                key={region}
                onClick={() => setRegionFilter(region)}
                className={cn(
                  "flex-shrink-0 text-xs px-3 py-1.5 rounded-lg border font-medium transition-colors",
                  regionFilter === region
                    ? "bg-blue-500 border-blue-500 text-white font-semibold"
                    : "bg-zinc-800 border-zinc-700 text-zinc-300 hover:bg-zinc-700 hover:border-zinc-500 hover:text-white"
                )}
              >
                {region === "all" ? t("profile.allRegions") : regionLabel(region)}
              </button>
            ))}
          </div>
        </div>

        {/* List */}
        <div className="overflow-y-auto flex-1">
          {filtered.length === 0 ? (
            <p className="text-center text-xs text-zinc-400 py-10">—</p>
          ) : (
            filtered.map((club) => {
              const name = lang === "ru" ? club.name_ru || club.name : club.name;
              const isSelected = club.id === current;
              return (
                <button
                  key={club.id}
                  onClick={() => { onSelect(club); onClose(); }}
                  className={cn(
                    "w-full flex items-center gap-3 px-4 py-3 border-b border-zinc-800/50 last:border-0 hover:bg-zinc-800/40 transition-colors text-left",
                    isSelected && "bg-zinc-800/60"
                  )}
                >
                  <ClubLogo club={club} size={36} />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-semibold text-zinc-200 truncate">{name}</p>
                    <p className="text-[11px] text-zinc-400">
                      {club.country.toUpperCase()} · {club.type === "national" ? t("profile.typeNational") : t("profile.typeClubs")}
                    </p>
                  </div>
                  {isSelected && <div className="w-2 h-2 rounded-full bg-yellow-400 flex-shrink-0" />}
                </button>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
}

// ─── Profile Card (full left column) ─────────────────────────────────────────

interface ProfileCardProps {
  displayName: string;
  rank: string;
  rating: number;
  club: Club | null;
  wins: number;
  draws: number;
  losses: number;
  leagues: number;
  points: number;
}

function ProfileCard({
  displayName, rank, rating, club,
  wins, draws, losses, leagues, points,
}: ProfileCardProps) {
  const { t } = useLang();

  const hasClub = !!club;
  const bgColor = hasClub ? club.color : "#18181b";
  const light = hasClub ? isLightColor(bgColor) : false;
  const textMain = light ? "text-black" : "text-white";
  const textSub  = light ? "text-black/80" : "text-white";

  return (
    <div className="flex flex-col gap-3">
      {/* Main card */}
      <div className="rounded-xl border border-zinc-800 overflow-hidden">

        {/* ── Club-colored banner ── */}
        <div
          style={{ backgroundColor: bgColor }}
          className="relative overflow-hidden px-5 pt-5 pb-4"
        >
          {/* Decorative circles */}
          {hasClub && (
            <>
              <div
                style={{ backgroundColor: club.color2 || "#fff", opacity: 0.10 }}
                className="absolute -right-6 -top-6 w-32 h-32 rounded-full pointer-events-none"
              />
              <div
                style={{ backgroundColor: club.color2 || "#fff", opacity: 0.06 }}
                className="absolute right-8 -bottom-10 w-40 h-40 rounded-full pointer-events-none"
              />
            </>
          )}

          <div className="relative flex items-center gap-4">
            {/* Аватар — логотип клуба или буква */}
            {hasClub ? (
              <ClubLogo club={club} size={56} />
            ) : (
              <div
                className="flex h-14 w-14 flex-shrink-0 items-center justify-center rounded-full text-xl font-black"
                style={{ backgroundColor: "#3f3f46", color: "#fff" }}
              >
                {(displayName || "?").slice(0, 1).toUpperCase()}
              </div>
            )}

            <div className="flex-1 min-w-0">
              <p className={cn("text-base font-black truncate", hasClub ? textMain : "text-zinc-100")}>
                {displayName}
              </p>
              {hasClub ? (
                <div className="flex items-center gap-1.5 mt-0.5">
                  <ClubLogo club={club} size={14} />
                  <p className={cn("text-xs font-semibold truncate", textSub)}>
                    {club.name_ru || club.name}
                  </p>
                </div>
              ) : (
                <p className="text-sm text-zinc-400">{rank || t("profile.noClubSelected")}</p>
              )}
            </div>

            <div className="text-right flex-shrink-0">
              <p className={cn("text-2xl font-black tabular-nums", hasClub ? textMain : ratingColor(rating))}>
                {rating}
              </p>
              <p className={cn("text-[9px] uppercase", hasClub ? textSub : "text-zinc-400")}>ELO</p>
            </div>
          </div>

          {/* Bottom accent strip */}
          {hasClub && (
            <div
              style={{ backgroundColor: club.color2 || "rgba(255,255,255,0.25)", height: 3 }}
              className="absolute bottom-0 left-0 right-0"
            />
          )}
        </div>

        {/* ── W / D / L ── */}
        <div className="grid grid-cols-3 divide-x divide-zinc-800 border-t border-zinc-800 bg-zinc-900">
          <div className="py-3 text-center">
            <p className="text-lg font-black text-green-400">{wins}</p>
            <p className="text-[10px] text-zinc-400 uppercase">{t("profile.wins")}</p>
          </div>
          <div className="py-3 text-center">
            <p className="text-lg font-black text-zinc-400">{draws}</p>
            <p className="text-[10px] text-zinc-400 uppercase">{t("profile.draws")}</p>
          </div>
          <div className="py-3 text-center">
            <p className="text-lg font-black text-red-400">{losses}</p>
            <p className="text-[10px] text-zinc-400 uppercase">{t("profile.losses")}</p>
          </div>
        </div>

        {/* ── Leagues / Points ── */}
        <div className="grid grid-cols-2 divide-x divide-zinc-800 border-t border-zinc-800 bg-zinc-900">
          <div className="py-3 text-center">
            <p className="text-lg font-black text-zinc-100">{leagues}</p>
            <p className="text-[10px] text-zinc-400 uppercase">{t("profile.leaguesLabel")}</p>
          </div>
          <div className="py-3 text-center">
            <p className="text-lg font-black text-zinc-100">{points}</p>
            <p className="text-[10px] text-zinc-400 uppercase">{t("profile.pointsLabel")}</p>
          </div>
        </div>
      </div>

    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function ProfilePage() {
  const router = useRouter();
  const qc = useQueryClient();
  const { user, loading, refreshUser, logout } = useAuth();
  const { t, lang } = useLang();
  const [showClubSelector, setShowClubSelector] = useState(false);
  const [selectedClubId, setSelectedClubId] = useState<string | undefined>(undefined);
  const [clubDirty, setClubDirty] = useState(false);

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
  const { data: clubs = [] } = useQuery({ queryKey: ["clubs"], queryFn: fetchClubs, staleTime: Infinity });
  const { data: playerProfile } = useQuery({
    queryKey: ["player-profile", user?.id],
    queryFn: () => fetchPlayerProfile(user!.id),
    enabled: !!user?.id,
  });
  const achievements: UserAchievement[] = playerProfile?.achievements ?? [];

  const { register, handleSubmit, reset, formState: { errors, isDirty } } = useForm<ProfileForm>({
    resolver: zodResolver(profileSchema),
  });

  // Initialise form and club selection from server data
  useEffect(() => {
    if (me) {
      reset({ display_name: me.display_name, team_power: String(me.team_power || "") });
      setSelectedClubId(me.favorite_club || undefined);
      setClubDirty(false);
    }
  }, [me, reset]);

  const handleClubSelect = (club: Club) => {
    setSelectedClubId(club.id);
    setClubDirty(club.id !== (me?.favorite_club || undefined));
  };

  const saveMutation = useMutation({
    mutationFn: async (data: ProfileForm) => {
      await updateMe({
        display_name: data.display_name.trim(),
        team_power: data.team_power ? Number(data.team_power) : undefined,
        ...(clubDirty && { favorite_club: selectedClubId ?? "" }),
      });
    },
    onSuccess: async () => {
      toast.success(t("profile.saveSuccess"));
      await refreshUser();
      qc.invalidateQueries({ queryKey: ["me"] });
      qc.invalidateQueries({ queryKey: ["players"] });
      setClubDirty(false);
    },
    onError: () => toast.error(t("profile.saveError")),
  });

  const [confirmDelete, setConfirmDelete] = useState(false);
  const deleteMutation = useMutation({
    mutationFn: deleteMe,
    onSuccess: () => {
      toast.success(t("profile.accountDeleted"));
      logout();
      router.push("/login");
    },
    onError: () => toast.error(t("profile.deleteError")),
  });

  if (loading || isLoading || !me) {
    return (
      <div className="space-y-4">
        <SkeletonProfile />
      </div>
    );
  }

  const totalWins   = myLeagues.reduce((s, m) => s + m.wins,   0);
  const totalDraws  = myLeagues.reduce((s, m) => s + m.draws,  0);
  const totalLosses = myLeagues.reduce((s, m) => s + m.losses, 0);
  const totalPoints = myLeagues.reduce((s, m) => s + m.points, 0);

  // Live preview: use selectedClubId (not me.favorite_club) so card updates before save
  const currentClub = selectedClubId ? (clubs.find((c) => c.id === selectedClubId) ?? null) : null;
  const anyDirty = isDirty || clubDirty;

  return (
    <>
      {showClubSelector && clubs.length > 0 && (
        <ClubSelector
          clubs={clubs}
          current={selectedClubId}
          onSelect={handleClubSelect}
          onClose={() => setShowClubSelector(false)}
        />
      )}

      <div className="space-y-5">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-zinc-400 mb-1">
            {t("profile.title")}
          </p>
          <h1 className="font-display text-2xl font-bold text-zinc-100">
            {me?.display_name ?? user?.display_name}
          </h1>
        </div>


        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">

          {/* ── Left: profile card (live preview) ── */}
          <ProfileCard
            displayName={me.display_name}
            rank={me.rank}
            rating={me.rating}
            club={currentClub}
            wins={totalWins}
            draws={totalDraws}
            losses={totalLosses}
            leagues={myLeagues.length}
            points={totalPoints}
          />

          {/* ── Right: edit form ── */}
          <div className="lg:col-span-2 rounded-xl card-premium p-6">
            <div className="flex items-center gap-2 mb-6">
              <User size={16} className="text-zinc-500" />
              <h2 className="text-sm font-semibold uppercase tracking-wider text-zinc-400">
                {t("profile.editProfile")}
              </h2>
            </div>

            <form
              onSubmit={handleSubmit((data) => saveMutation.mutate(data))}
              className="space-y-5"
            >
              {/* Display name */}
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-zinc-400">{t("profile.displayName")}</label>
                <Input {...register("display_name")} placeholder={t("profile.displayNamePlaceholder")} />
                {errors.display_name && (
                  <p className="text-xs text-red-400">{errors.display_name.message}</p>
                )}
              </div>

              {/* Team power */}
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-zinc-400">{t("profile.teamPower")}</label>
                <Input
                  {...register("team_power")}
                  placeholder={t("profile.teamPowerPlaceholder")}
                  inputMode="numeric"
                />
                <p className="text-[11px] text-zinc-400">{t("profile.teamPowerHint")}</p>
                {errors.team_power && (
                  <p className="text-xs text-red-400">{errors.team_power.message}</p>
                )}
              </div>

              {/* Favorite club — opens selector, updates live preview */}
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-zinc-400">{t("profile.favoriteClub")}</label>
                <button
                  type="button"
                  onClick={() => setShowClubSelector(true)}
                  className={cn(
                    "w-full flex items-center gap-3 px-3 py-2.5 rounded-md border transition-colors text-left",
                    clubDirty
                      ? "border-yellow-400/60 bg-yellow-500/5"
                      : "border-zinc-700 bg-zinc-800/50 hover:bg-zinc-800"
                  )}
                >
                  {currentClub ? (
                    <>
                      <ClubLogo club={currentClub} size={26} />
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-semibold text-zinc-100 truncate">
                          {lang === "ru" ? currentClub.name_ru || currentClub.name : currentClub.name}
                        </p>
                        <p className="text-[10px] text-zinc-400 uppercase">
                          {currentClub.country} · {currentClub.type === "national" ? t("profile.typeNational") : t("profile.typeClubs")}
                        </p>
                      </div>
                    </>
                  ) : (
                    <span className="text-sm text-zinc-400 flex-1">{t("profile.selectClub")}</span>
                  )}
                  <ChevronDown size={14} className="text-zinc-500 flex-shrink-0" />
                </button>
                {clubDirty && (
                  <p className="text-[10px] text-yellow-400">
                    ● {t("profile.clubNotSaved") ?? "Не сохранено — нажмите «Сохранить»"}
                  </p>
                )}
              </div>

              <Button
                type="submit"
                disabled={!anyDirty || saveMutation.isPending}
                className="w-full sm:w-auto"
              >
                <Save size={15} />
                {saveMutation.isPending ? t("profile.saving") : t("profile.saveProfile")}
              </Button>

              {/* Удаление профиля */}
              <div className="mt-6 pt-4 border-t border-red-500/15">
                {!confirmDelete ? (
                  <button
                    type="button"
                    onClick={() => setConfirmDelete(true)}
                    className="text-xs font-medium text-red-400 hover:text-red-300 transition-colors"
                  >
                    {t("profile.deleteAccount")}
                  </button>
                ) : (
                  <div className="space-y-2.5 rounded-lg border border-red-500/30 bg-red-500/5 p-3">
                    <p className="text-xs text-red-300">{t("profile.deleteConfirm")}</p>
                    <div className="flex gap-2">
                      <button
                        type="button"
                        disabled={deleteMutation.isPending}
                        onClick={() => deleteMutation.mutate()}
                        className="rounded-lg bg-red-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-red-500 disabled:opacity-50 transition-colors"
                      >
                        {deleteMutation.isPending ? t("profile.deleting") : t("profile.deleteYes")}
                      </button>
                      <button
                        type="button"
                        onClick={() => setConfirmDelete(false)}
                        className="rounded-lg border border-zinc-700 px-3 py-1.5 text-xs font-medium text-zinc-300 hover:bg-zinc-800 transition-colors"
                      >
                        {t("common.cancel")}
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </form>
          </div>
        </div>

        {/* My leagues */}
        <div className="rounded-xl card-premium overflow-hidden">
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
                    <p className="text-xs text-zinc-400">
                      {membership.wins}В · {membership.draws}Н · {membership.losses}П
                    </p>
                  </div>
                  <div className="text-right flex-shrink-0">
                    <p className="text-base font-black text-yellow-400">{membership.points}</p>
                    <p className="text-[10px] text-zinc-400 uppercase">очков</p>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>

        {/* Match history */}
        <div className="rounded-xl card-premium overflow-hidden">
          <div className="flex items-center gap-2 px-4 py-3 border-b border-zinc-800 text-xs font-semibold uppercase tracking-wider text-zinc-400">
            <History size={13} /> {t("profile.history")}
          </div>
          {history.length === 0 ? (
            <EmptyState icon={History} title={t("profile.noHistory")} text={t("profile.noHistoryText")} />
          ) : (
            <div>
              {history.slice(0, 12).map((match) => (
                <div
                  key={match.id}
                  className="flex items-center gap-4 px-4 py-3 border-b border-zinc-800/50 last:border-0"
                >
                  <span className="text-xs text-zinc-400 flex-shrink-0">{t("profile.round")} {match.round}</span>
                  <span className="flex-1 text-sm font-semibold text-zinc-200 truncate">
                    {match.home_name}{" "}
                    <span className="text-yellow-400">{match.home_goals}:{match.away_goals}</span>
                    {" "}{match.away_name}
                  </span>
                  <span className="text-xs text-zinc-400 flex-shrink-0">
                    {match.played_at
                      ? new Date(match.played_at).toLocaleDateString("ru-RU")
                      : t("profile.confirmed")}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Витрина трофеев — кубки за завершённые турниры */}
        {user && <TrophyCabinet userId={user.id} />}

        {/* Achievements */}
        <div className="rounded-xl card-premium overflow-hidden">
          <div className="flex items-center gap-2 px-4 py-3 border-b border-zinc-800 text-xs font-semibold uppercase tracking-wider text-zinc-400">
            <span className="text-base">🏅</span> Достижения
            <Link href="/trophies" className="ml-auto text-[10px] font-semibold normal-case tracking-normal text-zinc-500 hover:text-yellow-400 transition-colors">Все трофеи →</Link>
          </div>
          {achievements.length === 0 ? (
            <div className="px-4 py-6 text-center text-sm text-zinc-400">{t("leagueDetail.noAchievements")}</div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2 p-4">
              {achievements.map((a, i) => (
                <AchievementBadge key={a.id || `${a.achievement_id}-${a.league_id ?? 0}-${i}`} achievement={a} />
              ))}
            </div>
          )}
        </div>
      </div>
    </>
  );
}
