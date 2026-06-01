import axios from "axios";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "";

export const api = axios.create({ baseURL: API_URL });

const getToken = () =>
  typeof window !== "undefined" ? localStorage.getItem("efootball_jwt") : null;

api.interceptors.request.use((config) => {
  const token = getToken();
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

export interface User {
  id: number;
  display_name: string;
  rating: number;
  rank: string;
  team_power: number;
  language: string;
  has_telegram: boolean;
  email?: string;
  username?: string;
  position?: number;
  is_admin?: boolean;
  is_super_admin?: boolean;
}

export interface League {
  id: number;
  name: string;
  status: "registration" | "active" | "finished" | "archived" | string;
  level: number;
  max_players: number;
  rounds_type: "single" | "double" | string;
  country?: string;
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
}

export interface Match {
  id: number;
  league_id: number;
  home_user_id: number;
  away_user_id: number;
  home_name?: string;
  away_name?: string;
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
  match_id?: number;
  winner_user_id?: number;
  winner_name?: string;
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

export const adminGeneratePlayoff = (id: number, top_k = 8) =>
  api.post(`/api/admin/leagues/${id}/playoff`, { top_k }).then((r) => r.data);

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
export const updateMe = (data: { display_name?: string; team_power?: number }) =>
  api.put<User>("/api/me", data).then((r) => r.data);
export const generateLinkCode = () =>
  api.post<{ code: string; expires_in: string }>("/api/me/link-telegram").then((r) => r.data);
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
export const adminCreateLeague = (name: string) => api.post<League>("/api/admin/leagues", { name }).then((r) => r.data);
export const adminArchiveLeague = (id: number) => api.delete(`/api/admin/leagues/${id}`).then((r) => r.data);
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
