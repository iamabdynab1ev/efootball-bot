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
}

// openDirect — найти/создать ЛС с соперником, вернуть комнату. Кидает ошибку
// (403), если пользователь не был соперником по матчу.
export async function openDirect(userId: number): Promise<ChatRoom> {
  const r = await api.post("/api/chat/direct", { user_id: userId });
  return r.data as ChatRoom;
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
    return () => { on = false; };
  }, []);

  // Живо обновляем превью/порядок при любом входящем сообщении.
  useSSE("chat", () => { reload(); }, true);

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
  useEffect(() => {
    let on = true;
    setLoading(true);
    api.get(`/api/leagues/${leagueId}/chat/rooms`)
      .then((r) => { if (on) setRooms(r.data.rooms ?? []); })
      .finally(() => { if (on) setLoading(false); });
    return () => { on = false; };
  }, [leagueId]);
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
