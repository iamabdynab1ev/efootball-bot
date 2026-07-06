"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "./api";
import { playNotifySound } from "./sound";
import { sse } from "./sse";

export interface Notif {
  id: number;
  type: string;
  title: string;
  body?: string;
  link?: string;
  read: boolean;
  created_at: string;
}

// Единый источник правды для уведомлений: один initial-fetch + одна (refcount)
// SSE-подписка на «notification» для всего приложения. Колокольчик и выпадающий
// список читают одно и то же состояние, поэтому бейдж и лента всегда согласованы.

let items: Notif[] = [];
let unread = 0;
let loaded = false;
let unsub: (() => void) | null = null;
let refcount = 0;
const subscribers = new Set<() => void>();

function emit() {
  subscribers.forEach((f) => { try { f(); } catch { /* изолируем */ } });
}

function reset() {
  items = [];
  unread = 0;
  loaded = false;
  emit();
}

// Смена пользователя/выход — сбрасываем, чтобы чужие уведомления не утекли.
if (typeof window !== "undefined") {
  window.addEventListener("auth:unauthorized", reset);
}

function applyIncoming(n: Notif) {
  if (items.some((x) => x.id === n.id)) return;
  items = [n, ...items];
  if (!n.read) {
    unread++;
    playNotifySound(); // мягкий «динь» (настройка в Настройках, троттлинг внутри)
  }
  emit();
}

function ensureSubscription() {
  if (unsub) return;
  unsub = sse.subscribe("notification", (d: any) => {
    if (d && typeof d.id === "number") applyIncoming(d as Notif);
  });
}

async function loadInitial() {
  if (loaded) return;
  loaded = true;
  try {
    const r = await api.get("/api/notifications", { params: { limit: 30 } });
    // В колокольчике живут только непросмотренные: старые прочитанные не копятся.
    items = ((r.data.notifications ?? []) as Notif[]).filter((n) => !n.read);
    unread = r.data.unread ?? 0;
    emit();
  } catch {
    loaded = false; // позволяем повторить при следующем mount
  }
}

export interface NotificationsState {
  notifications: Notif[];
  unread: number;
  markRead: (id: number) => void;
  markAllRead: () => void;
  /** Убрать из списка уже просмотренные (вызывается при закрытии панели). */
  purgeRead: () => void;
  loadMore: () => Promise<void>;
}

export function useNotifications(): NotificationsState {
  const [, force] = useState(0);

  useEffect(() => {
    const rerender = () => force((n) => n + 1);
    subscribers.add(rerender);
    refcount++;
    ensureSubscription();
    loadInitial();
    return () => {
      subscribers.delete(rerender);
      refcount--;
      if (refcount === 0 && unsub) { unsub(); unsub = null; }
    };
  }, []);

  const markRead = useCallback((id: number) => {
    const target = items.find((n) => n.id === id);
    if (!target || target.read) return;
    items = items.map((n) => (n.id === id ? { ...n, read: true } : n));
    unread = Math.max(0, unread - 1);
    emit();
    api.post("/api/notifications/read", { ids: [id] }).catch(() => { /* оптимистично */ });
  }, []);

  const markAllRead = useCallback(() => {
    if (unread === 0) return;
    items = items.map((n) => ({ ...n, read: true }));
    unread = 0;
    emit();
    api.post("/api/notifications/read", { all: true }).catch(() => { /* оптимистично */ });
  }, []);

  const purgeRead = useCallback(() => {
    if (!items.some((n) => n.read)) return;
    items = items.filter((n) => !n.read);
    emit();
  }, []);

  const loadMore = useCallback(async () => {
    const last = items[items.length - 1];
    if (!last) return;
    try {
      const r = await api.get("/api/notifications", { params: { before: last.id, limit: 30 } });
      const more: Notif[] = (r.data.notifications ?? []).filter((n: Notif) => !n.read);
      const existing = new Set(items.map((n) => n.id));
      items = [...items, ...more.filter((n) => !existing.has(n.id))];
      emit();
    } catch { /* игнорируем сетевую ошибку дозагрузки */ }
  }, []);

  return { notifications: items, unread, markRead, markAllRead, purgeRead, loadMore };
}
