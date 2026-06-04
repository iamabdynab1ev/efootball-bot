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
  rounds_type: "single" | "double" | "league" | "cup" | "groups" | "groups_playoff" | "swiss" | "nations_league" | string;
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
  status: "scheduled" | "pending_confirm" | "disputed" | "confirmed" | "cancelled" | string;
  home_goals?: number;
  away_goals?: number;
  claimed_home?: number;
  claimed_away?: number;
  dispute_count: number;
  played_at?: string;
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
}

export interface TopScorersLeague {
  league: League;
  scorers: TopScorer[];
}

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

export const adminGeneratePlayoff = (id: number, top_k = 8) =>
  api.post(`/api/admin/leagues/${id}/playoff`, { top_k }).then((r) => r.data);
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
export const fetchClubs = () => api.get<Club[]>("/api/clubs").then((r) => r.data);
export const generateLinkCode = () =>
  api.post<{ code: string; expires_in: string; deep_link?: string }>("/api/me/link-telegram").then((r) => r.data);
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
) =>
  api.post<League>("/api/admin/leagues", {
    name,
    registration_deadline: deadline || undefined,
    rounds_type: roundsType || "double",
    num_groups: numGroups,
    group_advance: groupAdvance,
    best_runners_up: bestRunnersUp,
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
  global_stats?: GlobalStats;
  achievements?: UserAchievement[];
}
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

export const fetchHallOfFame = () =>
  api.get<HallOfFame>("/api/hall-of-fame").then((r) => r.data);

export const adminGetDeadlines = (leagueId: number) =>
  api.get<RoundDeadline[]>(`/api/admin/leagues/${leagueId}/deadlines`).then((r) => r.data);

export const adminSetDeadline = (leagueId: number, round: number, deadline: string) =>
  api.post(`/api/admin/leagues/${leagueId}/deadlines`, { round, deadline }).then((r) => r.data);

export const adminDeleteDeadline = (leagueId: number, round: number) =>
  api.delete(`/api/admin/leagues/${leagueId}/deadlines/${round}`).then((r) => r.data);

export const adminFinalizeLeague = (leagueId: number) =>
  api.post(`/api/admin/leagues/${leagueId}/finalize`).then((r) => r.data);
