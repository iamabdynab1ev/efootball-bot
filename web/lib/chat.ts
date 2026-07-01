"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "./api";
import { sse, useSSE } from "./sse";

export interface ChatRoom {
  id: number;
  league_id: number;
  group_name: string;
  title: string;
  archived: boolean;
  unread?: number;
}

// fetchRoomReads — прогресс прочтения участников комнаты как карта {userId: lastRead}
// (для отметок «прочитано» в групповом чате).
export async function fetchRoomReads(roomId: number): Promise<Record<number, number>> {
  const r = await api.get(`/api/chat/rooms/${roomId}/reads`);
  const map: Record<number, number> = {};
  for (const it of (r.data.reads ?? [])) map[it.user_id] = it.last_read;
  return map;
}

export interface ChatMessage {
  id: number;
  room_id: number;
  user_id?: number;
  author_name: string;
  author_club?: string;
  body: string;
  deleted: boolean;
  created_at: string;
}

export interface ChatMember {
  user_id: number;
  display_name: string;
  favorite_club?: string;
}

export interface DirectRoomView {
  room_id: number;
  other_id: number;
  other_name: string;
  other_club?: string;
  last_body: string;
  last_at?: string;
  last_author_id?: number;
  unread: number;
  other_last_read: number;
  other_last_seen?: string;
}

// openDirect — найти/создать ЛС с соперником, вернуть комнату. Кидает ошибку
// (403), если пользователь не был соперником по матчу.
export async function openDirect(userId: number): Promise<ChatRoom> {
  const r = await api.post("/api/chat/direct", { user_id: userId });
  return r.data as ChatRoom;
}

// markRead — отметить комнату прочитанной до сообщения upto; чистит счётчик
// непрочитанных и шлёт собеседнику ✓✓.
export async function markRead(roomId: number, upto: number): Promise<void> {
  if (!roomId || !upto) return;
  try {
    await api.post(`/api/chat/rooms/${roomId}/read`, { upto });
    // Локально просим бейджи (навбар/список) пересчитаться.
    if (typeof window !== "undefined") window.dispatchEvent(new Event("dm-read"));
  } catch { /* повторим при следующем сообщении/открытии */ }
}

// sendTyping — эфемерный сигнал «печатает…» (fire-and-forget, троттлится вызывающим).
export function sendTyping(roomId: number): void {
  if (!roomId) return;
  api.post(`/api/chat/rooms/${roomId}/typing`, {}).catch(() => { /* не критично */ });
}

// useUnreadTotal — всего непрочитанных ЛС; обновляется на новые сообщения и на
// локальное событие прочтения.
export function useUnreadTotal(enabled = true) {
  const [total, setTotal] = useState(0);

  const reload = useCallback(() => {
    if (!enabled) return;
    return api.get("/api/chat/unread")
      .then((r) => setTotal(r.data.total ?? 0))
      .catch(() => { /* оставляем прежнее */ });
  }, [enabled]);

  useEffect(() => {
    if (!enabled) { setTotal(0); return; }
    reload();
    const onRead = () => reload();
    window.addEventListener("dm-read", onRead);
    return () => window.removeEventListener("dm-read", onRead);
  }, [enabled, reload]);

  useSSE("chat", () => { reload(); }, enabled);

  return total;
}

// useDirectRooms — список личных диалогов; обновляется на новые сообщения (SSE).
export function useDirectRooms() {
  const [rooms, setRooms] = useState<DirectRoomView[]>([]);
  const [loading, setLoading] = useState(true);

  const reload = useCallback(() => {
    return api.get("/api/chat/direct")
      .then((r) => setRooms(r.data.rooms ?? []))
      .catch(() => { /* оставляем прежний список */ });
  }, []);

  useEffect(() => {
    let on = true;
    setLoading(true);
    api.get("/api/chat/direct")
      .then((r) => { if (on) setRooms(r.data.rooms ?? []); })
      .finally(() => { if (on) setLoading(false); });
    const onRead = () => reload();
    window.addEventListener("dm-read", onRead);
    return () => { on = false; window.removeEventListener("dm-read", onRead); };
  }, [reload]);

  // Живо обновляем превью/порядок/непрочитанные/галочки.
  useSSE("chat", () => { reload(); }, true);
  useSSE("chat_read", () => { reload(); }, true);

  return { rooms, loading, reload };
}

// Участники ИМЕННО этой комнаты (для @упоминаний): в общей — вся лига, в
// групповой — только её игроки. Перезагружается при смене комнаты.
export function useChatMembers(roomId: number | null) {
  const [members, setMembers] = useState<ChatMember[]>([]);
  useEffect(() => {
    if (roomId == null) { setMembers([]); return; }
    let on = true;
    api.get(`/api/chat/rooms/${roomId}/members`)
      .then((r) => { if (on) setMembers(r.data.members ?? []); })
      .catch(() => { if (on) setMembers([]); });
    return () => { on = false; };
  }, [roomId]);
  return members;
}

export function useChatRooms(leagueId: number) {
  const [rooms, setRooms] = useState<ChatRoom[]>([]);
  const [loading, setLoading] = useState(true);

  const reload = useCallback(() => {
    return api.get(`/api/leagues/${leagueId}/chat/rooms`)
      .then((r) => setRooms(r.data.rooms ?? []))
      .catch(() => { /* оставляем прежний список */ });
  }, [leagueId]);

  useEffect(() => {
    let on = true;
    setLoading(true);
    api.get(`/api/leagues/${leagueId}/chat/rooms`)
      .then((r) => { if (on) setRooms(r.data.rooms ?? []); })
      .finally(() => { if (on) setLoading(false); });
    const onRead = () => reload();
    window.addEventListener("dm-read", onRead);
    return () => { on = false; window.removeEventListener("dm-read", onRead); };
  }, [leagueId, reload]);

  // Живо пересчитываем непрочитанные по комнатам.
  useSSE("chat", () => { reload(); }, true);
  useSSE("chat_read", () => { reload(); }, true);

  return { rooms, loading };
}

const PAGE = 50;

export function useChatRoom(roomId: number | null) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const lastIdRef = useRef(0); // максимальный полученный id — для catch-up

  const mergeAppend = useCallback((incoming: ChatMessage[]) => {
    if (incoming.length === 0) return;
    setMessages((prev) => {
      const seen = new Set(prev.map((m) => m.id));
      const add = incoming.filter((m) => !seen.has(m.id));
      if (add.length === 0) return prev;
      for (const m of add) if (m.id > lastIdRef.current) lastIdRef.current = m.id;
      return [...prev, ...add];
    });
  }, []);

  const loadLatest = useCallback(async (rid: number) => {
    setLoading(true);
    try {
      const r = await api.get(`/api/chat/rooms/${rid}/messages`, { params: { limit: PAGE } });
      const list: ChatMessage[] = r.data.messages ?? [];
      setMessages(list);
      setHasMore(list.length === PAGE);
      lastIdRef.current = list.length ? list[list.length - 1].id : 0;
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    setMessages([]);
    setHasMore(false);
    lastIdRef.current = 0;
    if (roomId != null) loadLatest(roomId);
  }, [roomId, loadLatest]);

  // Догрузка пропущенного при реконнекте SSE (no-loss): тянем всё новее lastId.
  useEffect(() => {
    if (roomId == null) return;
    let wasLive: boolean | null = null;
    const unsub = sse.subscribeStatus((live) => {
      if (live && wasLive === false) {
        api.get(`/api/chat/rooms/${roomId}/messages`, { params: { since: lastIdRef.current, limit: PAGE } })
          .then((r) => mergeAppend(r.data.messages ?? []))
          .catch(() => { /* следующий реконнект повторит */ });
      }
      wasLive = live;
    });
    return unsub;
  }, [roomId, mergeAppend]);

  // Живая доставка новых сообщений в личный топик.
  useSSE("chat", (m: ChatMessage) => {
    if (roomId == null || !m || m.room_id !== roomId) return;
    mergeAppend([m]);
  }, roomId != null);

  // Удаление админом.
  useSSE("chat_deleted", (d: any) => {
    if (roomId == null || !d || d.room_id !== roomId) return;
    setMessages((prev) => prev.map((m) => (m.id === d.id ? { ...m, deleted: true, body: "" } : m)));
  }, roomId != null);

  const send = useCallback(async (body: string) => {
    if (roomId == null) return;
    const r = await api.post(`/api/chat/rooms/${roomId}/messages`, { body });
    mergeAppend([r.data as ChatMessage]); // SSE тоже доставит — дедуп по id
  }, [roomId, mergeAppend]);

  const loadOlder = useCallback(async () => {
    if (roomId == null || messages.length === 0) return;
    const before = messages[0].id;
    const r = await api.get(`/api/chat/rooms/${roomId}/messages`, { params: { before, limit: PAGE } });
    const older: ChatMessage[] = r.data.messages ?? [];
    setHasMore(older.length === PAGE);
    setMessages((prev) => {
      const seen = new Set(prev.map((m) => m.id));
      return [...older.filter((m) => !seen.has(m.id)), ...prev];
    });
  }, [roomId, messages]);

  return { messages, loading, hasMore, send, loadOlder };
}
