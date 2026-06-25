import axios from "axios";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "";

export const api = axios.create({
  baseURL: API_URL,
  timeout: 15000, // 15 секунд — достаточно даже на плохом интернете
});

const getToken = () =>
  typeof window !== "undefined" ? localStorage.getItem("efootball_jwt") : null;

api.interceptors.request.use((config) => {
  const token = getToken();
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// Перехватываем сетевые ошибки и добавляем понятное сообщение
api.interceptors.response.use(
  (r) => r,
  (err) => {
    if (!err.response) {
      // Нет ответа от сервера: нет интернета или timeout
      err.userMessage = "Нет соединения. Проверьте интернет и попробуйте снова.";
    } else if (err.response.status === 401) {
      err.userMessage = "Сессия истекла. Войдите снова.";
      // Токен невалиден/истёк — разлогиниваем, чтобы не остаться в "залогиненном" UI с битым токеном
      if (typeof window !== "undefined" && getToken()) {
        localStorage.removeItem("efootball_jwt");
        window.dispatchEvent(new Event("auth:unauthorized"));
      }
    } else if (err.response.status === 403) {
      err.userMessage = "Нет доступа к этому действию.";
    } else if (err.response.status >= 500) {
      err.userMessage = "Ошибка сервера. Попробуйте позже.";
    }
    return Promise.reject(err);
  }
);

export interface User {
  id: number;
  display_name: string;
  rating: number;
  rank: string;
  team_power: number;
  language: string;
  has_telegram: boolean;
  favorite_club?: string;
  email?: string;
  username?: string;
  position?: number;
  is_admin?: boolean;
  is_super_admin?: boolean;
}

export interface Club {
  id: string;
  name: string;
  name_ru: string;
  type: "club" | "national";
  country: string;
  region: string;
  color: string;
  color2: string;
  logo: string;
}

export interface League {
  id: number;
  name: string;
  status: "draft" | "registration" | "active" | "finished" | "archived" | string;
  level: number;
  max_players: number;
  players_count?: number; // реальное число вступивших (одобренных)
  rounds_type: "single" | "double" | "league" | "cup" | "groups" | "groups_playoff" | "swiss" | "nations_league" | "double_elim" | string;
  num_groups?: number;
  group_advance?: number;
  current_round?: number;
  country?: string;
  registration_deadline?: string;
}

export interface Standing {
  user_id: number;
  display_name?: string;
  rating?: number;
  rank?: string;
  position?: number;
  points: number;
  wins: number;
  draws: number;
  losses: number;
  goals_for: number;
  goals_against: number;
  goal_diff: number;
  status: string;
  league?: League;
  favorite_club?: string;
  form?: string[];
}

export interface Match {
  id: number;
  league_id: number;
  home_user_id: number;
  away_user_id: number;
  home_name?: string;
  away_name?: string;
  home_club?: string;
  away_club?: string;
  round: number;
  stage?: string;
  bracket_slot?: number;
  status: "scheduled" | "pending_confirm" | "disputed" | "confirmed" | "cancelled" | string;
  home_goals?: number;
  away_goals?: number;
  claimed_home?: number;
  claimed_away?: number;
  dispute_count: number;
  played_at?: string;
  best_of?: number;
  home_wins?: number;
  away_wins?: number;
}

export function isPlayoffMatch(m: Match): boolean {
  const s = m.stage ?? "";
  return s === "qf" || s === "sf" || s === "final" || s === "r16" || s === "r32";
}

// Ключ перевода стадии плей-офф (leagueDetail.stageXX) или null для групп.
// Текст берётся через t() на месте рендера — стадии локализуются на ru/uz/tg.
export function stageLabelKey(stage?: string): string | null {
  const map: Record<string, string> = {
    qf: "stageQF", sf: "stageSF", final: "stageFinal",
    r16: "stageR16", r32: "stageR32",
  };
  return stage ? map[stage] ?? null : null;
}

// Ключи перевода краткого названия и одно-строчного описания формата лиги
// (namespace formats.*). Текст резолвится через t() на месте рендера.
export function leagueFormatKeys(roundsType?: string): { label: string; desc: string } {
  switch (roundsType) {
    case "double":
      return { label: "formats.doubleLabel", desc: "formats.doubleDesc" };
    case "groups":
    case "groups_playoff":
    case "hybrid":
      return { label: "formats.groupsLabel", desc: "formats.groupsDesc" };
    case "cup":
      return { label: "formats.cupLabel", desc: "formats.cupDesc" };
    case "swiss":
      return { label: "formats.swissLabel", desc: "formats.swissDesc" };
    case "nations_league":
      return { label: "formats.nationsLabel", desc: "formats.nationsDesc" };
    case "double_elim":
      return { label: "formats.deLabel", desc: "formats.deDesc" };
    default: // single | league
      return { label: "formats.singleLabel", desc: "formats.singleDesc" };
  }
}

export interface Round {
  round: number;
  matches: Match[];
}

export interface TopScorer {
  position: number;
  user_id: number;
  display_name: string;
  rating: number;
  team_power: number;
  goals_for: number;
  favorite_club?: string;
}

export interface TopScorersLeague {
  league: League;
  scorers: TopScorer[];
}

export interface StatEntry {
  user_id: number;
  display_name: string;
  favorite_club?: string;
  rank: string;
  rating: number;
  played: number;
  wins: number;
  draws: number;
  losses: number;
  goals_for: number;
  goals_against: number;
  team_power: number;
  win_rate: number;
  avg_goals: number;
  streak: number;
}

export const fetchStatWinRate   = () => api.get<StatEntry[]>("/api/stats/win-rate").then(r => r.data);
export const fetchStatStreaks   = () => api.get<StatEntry[]>("/api/stats/streaks").then(r => r.data);
export const fetchStatAvgGoals  = () => api.get<StatEntry[]>("/api/stats/avg-goals").then(r => r.data);
export const fetchStatTeamPower = () => api.get<StatEntry[]>("/api/stats/team-power").then(r => r.data);
export const fetchStatActivity  = () => api.get<StatEntry[]>("/api/stats/activity").then(r => r.data);

export interface AdminUser {
  id: number;
  user_id: number;
  telegram_id?: number;
  display_name?: string;
  username?: string;
  role: "admin" | "super_admin" | string;
}

// ── Bracket ──────────────────────────────────────────────────────────────────

export interface BracketSlot {
  slot: number;
  stage: string;
  home_user_id?: number;
  away_user_id?: number;
  home_name: string;
  away_name: string;
  home_club?: string;
  away_club?: string;
  match_id?: number;
  winner_user_id?: number;
  winner_name?: string;
  winner_club?: string;
  home_goals?: number;
  away_goals?: number;
  match_status?: string;
}

export interface BracketStage {
  stage: string;
  label: string;
  slots: BracketSlot[];
}

export const fetchBracket = (id: number) =>
  api.get<{ stages: BracketStage[] }>(`/api/leagues/${id}/bracket`).then((r) => r.data.stages);

// ── Double elimination ───────────────────────────────────────────────────────

export interface DENode {
  node_key: number;
  bracket: "de_w" | "de_l" | "de_gf";
  round: number;
  ord: number;
  is_reset: boolean;
  home_user_id?: number;
  away_user_id?: number;
  home_name: string;
  away_name: string;
  home_club?: string;
  away_club?: string;
  winner_user_id?: number;
  winner_name?: string;
  home_goals?: number;
  away_goals?: number;
  status?: string;
  best_of?: number;
  home_wins?: number;
  away_wins?: number;
}

export interface DERoundGroup {
  round: number;
  nodes: DENode[];
}

export interface DEBracketGroup {
  bracket: "de_w" | "de_l" | "de_gf";
  rounds: DERoundGroup[];
}

export const fetchDoubleElim = (id: number) =>
  api.get<{ brackets: DEBracketGroup[] }>(`/api/leagues/${id}/double-elim`).then((r) => r.data.brackets);

export interface GroupStanding extends Standing {
  group_name: string;
}

export interface GroupInfo {
  name: string;
  standings: GroupStanding[];
}

export const fetchLeagueGroups = (id: number) =>
  api.get<{ groups: GroupInfo[] }>(`/api/leagues/${id}/groups`).then((r) => r.data.groups);

export const fetchGroupStandings = (id: number, group: string) =>
  api.get<{ standings: GroupStanding[]; group: string }>(`/api/leagues/${id}/groups/${group}/standings`).then((r) => r.data);

export const fetchGroupSchedule = (id: number, group: string) =>
  api.get<{ rounds: Round[]; group: string }>(`/api/leagues/${id}/groups/${group}/schedule`).then((r) => r.data);

export const adminGeneratePlayoff = (id: number, opts: { top_k?: number; group_advance?: number; random_draw?: boolean } = { top_k: 8 }) =>
  api.post(`/api/admin/leagues/${id}/playoff`, opts).then((r) => r.data);

export interface PlayoffBracketOption {
  advance: number;     // сколько выходит из каждой группы
  qualifiers: number;  // итоговое число команд в сетке (степень двойки)
  stage: string;       // первая стадия: r32 | r16 | qf | sf | final
}

export interface PlayoffOptions {
  groups: { name: string; size: number }[];
  advance_min: number;
  advance_max: number;
  advance_default: number;
  options?: PlayoffBracketOption[]; // ровные варианты сетки (предпочтительные)
}

export const fetchPlayoffOptions = (id: number) =>
  api.get<PlayoffOptions>(`/api/admin/leagues/${id}/playoff-options`).then((r) => r.data);
export const adminNextRound = (id: number) =>
  api.post(`/api/admin/leagues/${id}/next-round`).then((r) => r.data);
export const adminFinalFour = (id: number) =>
  api.post(`/api/admin/leagues/${id}/final-four`).then((r) => r.data);
export const fetchLeagueProgress = (id: number) =>
  api.get<{ remaining: number }>(`/api/leagues/${id}/progress`).then((r) => r.data);

// ── Leagues ──────────────────────────────────────────────────────────────────
export const fetchLeagues = () => api.get<League[]>("/api/leagues").then((r) => r.data);
export const fetchLeague = (id: number) => api.get<League>(`/api/leagues/${id}`).then((r) => r.data);
export const fetchStandings = (id: number) =>
  api.get<{ standings: Standing[] }>(`/api/leagues/${id}/standings`).then((r) => r.data.standings);
export const fetchSchedule = (id: number) =>
  api.get<{ rounds: Round[] }>(`/api/leagues/${id}/schedule`).then((r) => r.data.rounds);
export const fetchMyMatches = (id: number) => api.get<Match[]>(`/api/leagues/${id}/my-matches`).then((r) => r.data);
export const fetchMyLeagues = () => api.get<Standing[]>("/api/me/leagues").then((r) => r.data);
export const fetchMyHistory = (leagueId?: number) => {
  const qs = leagueId ? `?league_id=${leagueId}` : "";
  return api.get<Match[]>(`/api/me/history${qs}`).then((r) => r.data);
};
export const fetchTopScorers = () =>
  api.get<{ leagues: TopScorersLeague[] }>("/api/top-scorers").then((r) => r.data.leagues);
export const joinLeague = (id: number) => api.post(`/api/leagues/${id}/join`).then((r) => r.data);
export const fetchPlayers = (limit = 100) => api.get<User[]>(`/api/players?limit=${limit}`).then((r) => r.data);
export const fetchMe = () => api.get<User>("/api/me").then((r) => r.data);
export const updateMe = (data: { display_name?: string; team_power?: number; favorite_club?: string }) =>
  api.patch<User>("/api/me", data).then((r) => r.data);
export const deleteMe = () => api.delete("/api/me").then((r) => r.data);
export const adminDeleteUser = (uid: number) => api.delete(`/api/admin/users/${uid}`).then((r) => r.data);
export const fetchClubs = () => api.get<Club[]>("/api/clubs").then((r) => r.data);
export const generateLinkCode = () =>
  api.post<{ code: string; expires_in: string; deep_link?: string }>("/api/me/link-telegram").then((r) => r.data);
export const unlinkTelegram = () => api.post("/api/me/unlink-telegram").then((r) => r.data);
export const submitResult = (id: number, home_goals: number, away_goals: number) =>
  api.post(`/api/matches/${id}/result`, { home_goals, away_goals }).then((r) => r.data);
export const confirmMatch = (id: number) => api.post(`/api/matches/${id}/confirm`).then((r) => r.data);
export const disputeMatch = (id: number) => api.post(`/api/matches/${id}/dispute`).then((r) => r.data);

export interface UserWithRole {
  id: number;
  display_name: string;
  rating: number;
  rank: string;
  team_power: number;
  has_telegram: boolean;
  admin_role: "" | "admin" | "super_admin";
}

export const adminFetchUsers = () => api.get<UserWithRole[]>("/api/admin/users").then((r) => r.data);
export const adminFetchLeagues = () => api.get<League[]>("/api/admin/leagues").then((r) => r.data);
export const adminCreateLeague = (
  name: string,
  deadline?: string,
  roundsType?: string,
  numGroups?: number,
  groupAdvance?: number,
  bestRunnersUp?: number,
  bestOf?: number,
) =>
  api.post<League>("/api/admin/leagues", {
    name,
    registration_deadline: deadline || undefined,
    rounds_type: roundsType || "double",
    num_groups: numGroups,
    group_advance: groupAdvance,
    best_runners_up: bestRunnersUp,
    best_of: bestOf,
  }).then((r) => r.data);
export const adminArchiveLeague = (id: number) => api.delete(`/api/admin/leagues/${id}`).then((r) => r.data);
export const adminPurgeLeague = (id: number) => api.delete(`/api/admin/leagues/${id}/purge`).then((r) => r.data);
export const adminUpdateLeague = (id: number, data: { name: string; registration_deadline?: string }) =>
  api.patch<League>(`/api/admin/leagues/${id}`, data).then((r) => r.data);
export const adminFetchMembers = (id: number) =>
  api.get<{ pending: Standing[]; approved: Standing[] }>(`/api/admin/leagues/${id}/members`).then((r) => r.data);
export const adminApprove = (lid: number, uid: number) =>
  api.post(`/api/admin/leagues/${lid}/members/${uid}/approve`).then((r) => r.data);
export const adminReject = (lid: number, uid: number) =>
  api.post(`/api/admin/leagues/${lid}/members/${uid}/reject`).then((r) => r.data);
export const adminDraw = (id: number) => api.post(`/api/admin/leagues/${id}/draw`).then((r) => r.data);
export const adminOpenLeague = (id: number) => api.post(`/api/admin/leagues/${id}/open`).then((r) => r.data);
export const adminResolve = (id: number, home_goals: number, away_goals: number, note?: string) =>
  api.post(`/api/admin/matches/${id}/resolve`, { home_goals, away_goals, note }).then((r) => r.data);
export const adminFetchDisputed = () => api.get<Match[]>("/api/admin/disputed").then((r) => r.data);
export const adminFetchAdmins = () => api.get<AdminUser[]>("/api/admin/admins").then((r) => r.data);
export const adminAdd = (data: { telegram_id?: number; user_id?: number; role: "admin" | "super_admin" }) =>
  api.post("/api/admin/admins", data).then((r) => r.data);
export const adminRemove = (userId: number) => api.delete(`/api/admin/admins/${userId}`).then((r) => r.data);
export const adminResetRatings = () => api.post("/api/admin/ratings/reset").then((r) => r.data);

export interface Achievement {
  id: number;
  code: string;
  icon: string;
  name_uz: string;
  name_ru: string;
  name_tg: string;
}
export interface UserAchievement {
  id: number;
  user_id: number;
  achievement_id: number;
  league_id?: number;
  earned_at: string;
  achievement?: Achievement;
}
export interface GlobalStats {
  total_wins: number;
  total_draws: number;
  total_losses: number;
  total_goals_for: number;
  total_goals_against: number;
}
export interface PlayerProfile {
  id: number;
  display_name: string;
  rating: number;
  rank: string;
  team_power: number;
  favorite_club?: string;
  // Бэкенд отдаёт плоские поля статистики (не nested global_stats)
  total_matches?: number;
  total_wins?: number;
  total_draws?: number;
  total_losses?: number;
  total_goals_for?: number;
  total_goals_against?: number;
  win_rate?: number;
  achievements?: UserAchievement[];
}

export interface HeadToHead {
  played: number;
  my_wins: number;
  opp_wins: number;
  draws: number;
  my_goals: number;
  opp_goals: number;
  recent: { my_goals: number; opp_goals: number; result: "W" | "D" | "L"; played_at?: string }[];
}

export const fetchHeadToHead = (opponentId: number) =>
  api.get<HeadToHead>(`/api/players/${opponentId}/h2h`).then((r) => r.data);

// ── Web Push ──────────────────────────────────────────────────────────────────
export const fetchVapidPublic = () =>
  api.get<{ key: string }>("/api/push/vapid-public").then((r) => r.data.key);
export const pushSubscribe = (sub: PushSubscriptionJSON) =>
  api.post("/api/me/push/subscribe", sub).then((r) => r.data);
export const pushUnsubscribe = (endpoint: string) =>
  api.post("/api/me/push/unsubscribe", { endpoint }).then((r) => r.data);
export const pushTest = () => api.post("/api/me/push/test").then((r) => r.data);

// ── Admin broadcast ──────────────────────────────────────────────────────────
export const adminBroadcast = (text: string, title?: string) =>
  api.post<{ pushed: number; telegram: number }>("/api/admin/broadcast", { text, title }).then((r) => r.data);
export const adminNotifyUser = (user_id: number, text: string, title?: string) =>
  api.post<{ pushed: number; telegram: number; name: string }>("/api/admin/notify", { user_id, text, title }).then((r) => r.data);

// ── Support contact (dynamic, admin-editable) ────────────────────────────────
export interface SupportContact {
  phone: string;
  whatsapp: string;
  telegram: string;
}
export const fetchSupport = () =>
  api.get<SupportContact>("/api/settings/support").then((r) => r.data);
export const adminSetSupport = (c: SupportContact) =>
  api.post<SupportContact>("/api/admin/settings/support", c).then((r) => r.data);

// URL карточки игрока (PNG). v= версия дизайна — меняем при обновлении дизайна,
// чтобы сбросить кэш браузера и показать новую карточку.
export const playerCardUrl = (id: number) => `${API_URL}/api/players/${id}/card.png?v=4`;
export interface SeasonAward {
  id: number;
  season_id: number;
  league_id?: number;
  award_type: string;
  user_id: number;
  display_name?: string;
  league_name?: string;
  value?: number;
  created_at: string;
}
export interface HallOfFame {
  awards: SeasonAward[];
}
export interface RoundDeadline {
  id: number;
  league_id: number;
  round: number;
  deadline: string;
  reminder_24h_sent: boolean;
  reminder_1h_sent: boolean;
}

export const fetchPlayerProfile = (id: number) =>
  api.get<PlayerProfile>(`/api/players/${id}`).then((r) => r.data);

// API отдаёт награды вложенно: seasons → leagues → awards. Разворачиваем в
// плоский список SeasonAward, который ожидает страница Зала Славы.
interface HofApiAward { award_type: string; user_id: number; display_name: string; value: number; created_at: string }
interface HofApiLeague { season_id: number; season_name: string; league_id: number; league_name: string; awards: HofApiAward[] }
interface HofApiSeason { season_id: number; season_name: string; leagues: HofApiLeague[] }

export const fetchHallOfFame = (): Promise<HallOfFame> =>
  api.get<{ seasons: HofApiSeason[] }>("/api/hall-of-fame").then((r) => {
    const awards: SeasonAward[] = [];
    let key = 0;
    for (const s of r.data.seasons ?? []) {
      for (const l of s.leagues ?? []) {
        for (const a of l.awards ?? []) {
          awards.push({
            id: ++key,
            season_id: s.season_id,
            league_id: l.league_id,
            league_name: l.league_name,
            award_type: a.award_type,
            user_id: a.user_id,
            display_name: a.display_name,
            value: a.value,
            created_at: a.created_at,
          });
        }
      }
    }
    return { awards };
  });

export const adminGetDeadlines = (leagueId: number) =>
  api.get<RoundDeadline[]>(`/api/admin/leagues/${leagueId}/deadlines`).then((r) => r.data);

export const adminSetDeadline = (leagueId: number, round: number, deadline: string) =>
  api.post(`/api/admin/leagues/${leagueId}/deadlines`, { round, deadline }).then((r) => r.data);

export const adminDeleteDeadline = (leagueId: number, round: number) =>
  api.delete(`/api/admin/leagues/${leagueId}/deadlines/${round}`).then((r) => r.data);

export const adminFinalizeLeague = (leagueId: number) =>
  api.post(`/api/admin/leagues/${leagueId}/finalize`).then((r) => r.data);
