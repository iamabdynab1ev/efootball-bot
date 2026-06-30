"use client";

import { useEffect, useState } from "react";
import { api } from "./api";
import { sse } from "./sse";

// Глобальный реестр онлайн-пользователей: один initial-снимок (/api/online) +
// presence-дельты из общего SSE-канала. Подписка на SSE — единственная на всё
// приложение (refcount), сколько бы компонентов ни вызвали usePresence.

let onlineSet = new Set<number>();
let loaded = false;
let unsub: (() => void) | null = null;
let refcount = 0;
const subscribers = new Set<() => void>();

function notify() {
  subscribers.forEach((f) => { try { f(); } catch { /* изолируем */ } });
}

function ensureSubscription() {
  if (unsub) return;
  unsub = sse.subscribe("presence", (d: any) => {
    if (!d || typeof d.user_id !== "number") return;
    const id = Number(d.user_id);
    if (d.online) onlineSet.add(id);
    else onlineSet.delete(id);
    notify();
  });
}

function loadOnce() {
  if (loaded) return;
  loaded = true;
  api.get("/api/online")
    .then((r) => {
      onlineSet = new Set<number>((r.data.online ?? []).map(Number));
      notify();
    })
    .catch(() => { loaded = false; }); // позволяем повторить при следующем mount
}

export interface Presence {
  isOnline: (userID: number) => boolean;
  online: Set<number>;
}

export function usePresence(): Presence {
  const [, force] = useState(0);

  useEffect(() => {
    const rerender = () => force((n) => n + 1);
    subscribers.add(rerender);
    refcount++;
    ensureSubscription();
    loadOnce();

    return () => {
      subscribers.delete(rerender);
      refcount--;
      if (refcount === 0 && unsub) {
        unsub();
        unsub = null;
      }
    };
  }, []);

  return {
    isOnline: (id: number) => onlineSet.has(id),
    online: onlineSet,
  };
}
