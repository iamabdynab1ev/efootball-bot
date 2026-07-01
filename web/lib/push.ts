"use client";

import { fetchVapidPublic, pushSubscribe, pushUnsubscribe } from "./api";

// Общая логика web-push (переиспользуется тоглом в настройках и авто-запросом
// при входе) — чтобы не дублировать подписку/отписку в двух местах.

// VAPID public key (base64url) → Uint8Array для pushManager.subscribe.
function urlBase64ToUint8Array(base64: string) {
  const padding = "=".repeat((4 - (base64.length % 4)) % 4);
  const b64 = (base64 + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(b64);
  const arr = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) arr[i] = raw.charCodeAt(i);
  return arr;
}

export type PushSupport = "ok" | "unsupported" | "ios-install";

// pushSupport определяет, доступен ли push. На iPhone push работает только для
// установленного на «Домой» приложения — тогда просим установить.
export function pushSupport(): PushSupport {
  if (typeof window === "undefined") return "unsupported";
  const ua = navigator.userAgent.toLowerCase();
  const isIOS = /iphone|ipad|ipod/.test(ua);
  const standalone =
    window.matchMedia("(display-mode: standalone)").matches ||
    // @ts-expect-error — iOS Safari
    window.navigator.standalone === true;
  if (!("serviceWorker" in navigator) || !("PushManager" in window)) {
    return isIOS && !standalone ? "ios-install" : "unsupported";
  }
  return "ok";
}

export function permissionDenied(): boolean {
  return typeof Notification !== "undefined" && Notification.permission === "denied";
}

// isPushEnabled — есть ли активная подписка (регистрирует SW и проверяет апдейт).
export async function isPushEnabled(): Promise<boolean> {
  if (pushSupport() !== "ok") return false;
  try {
    const reg = await navigator.serviceWorker.register("/sw.js");
    try { await reg.update(); } catch { /* ignore */ }
    const sub = await reg.pushManager.getSubscription();
    return !!sub;
  } catch {
    return false;
  }
}

// enablePush запрашивает разрешение и оформляет подписку. Возвращает true при
// успехе, "denied" если пользователь отказал, false при иной ошибке.
export async function enablePush(): Promise<true | "denied" | false> {
  if (pushSupport() !== "ok") return false;
  try {
    const perm = await Notification.requestPermission();
    if (perm !== "granted") return "denied";
    const reg = await navigator.serviceWorker.ready;
    const key = await fetchVapidPublic();
    if (!key) return false;
    const sub = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(key),
    });
    await pushSubscribe(sub.toJSON());
    return true;
  } catch {
    return false;
  }
}

export async function disablePush(): Promise<void> {
  const reg = await navigator.serviceWorker.ready;
  const sub = await reg.pushManager.getSubscription();
  if (sub) {
    await pushUnsubscribe(sub.endpoint);
    await sub.unsubscribe();
  }
}
