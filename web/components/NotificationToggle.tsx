"use client";

import { useEffect, useState } from "react";
import { Bell, BellRing, BellOff } from "lucide-react";
import { toast } from "sonner";
import { fetchVapidPublic, pushSubscribe, pushUnsubscribe, pushTest } from "@/lib/api";
import { useLang } from "@/lib/i18n";

// VAPID public key (base64url) → Uint8Array для pushManager.subscribe
function urlBase64ToUint8Array(base64: string) {
  const padding = "=".repeat((4 - (base64.length % 4)) % 4);
  const b64 = (base64 + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(b64);
  const arr = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) arr[i] = raw.charCodeAt(i);
  return arr;
}

type State = "loading" | "unsupported" | "off" | "on" | "busy";

export function NotificationToggle() {
  const { t } = useLang();
  const [state, setState] = useState<State>("loading");

  useEffect(() => {
    if (typeof window === "undefined" || !("serviceWorker" in navigator) || !("PushManager" in window)) {
      setState("unsupported");
      return;
    }
    navigator.serviceWorker
      .register("/sw.js")
      .then(async (reg) => {
        // Принудительно проверяем обновление SW (иначе браузер держит старую версию)
        try { await reg.update(); } catch { /* ignore */ }
        return reg.pushManager.getSubscription();
      })
      .then((sub) => setState(sub ? "on" : "off"))
      .catch(() => setState("unsupported"));
  }, []);

  async function enable() {
    setState("busy");
    try {
      const perm = await Notification.requestPermission();
      if (perm !== "granted") {
        toast.error(t("push.denied"));
        setState("off");
        return;
      }
      const reg = await navigator.serviceWorker.ready;
      const key = await fetchVapidPublic();
      if (!key) {
        toast.error(t("push.error"));
        setState("off");
        return;
      }
      const sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(key),
      });
      await pushSubscribe(sub.toJSON());
      setState("on");
      toast.success(t("push.enabled"));
    } catch {
      toast.error(t("push.error"));
      setState("off");
    }
  }

  async function disable() {
    setState("busy");
    try {
      const reg = await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.getSubscription();
      if (sub) {
        await pushUnsubscribe(sub.endpoint);
        await sub.unsubscribe();
      }
      setState("off");
      toast.success(t("push.disabled"));
    } catch {
      setState("off");
    }
  }

  if (state === "loading" || state === "unsupported") return null;

  const on = state === "on";

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-4">
      <div className="flex items-center gap-3">
        <div className={`flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl ${on ? "bg-green-500/15 text-green-400" : "bg-zinc-800 text-zinc-400"}`}>
          {on ? <BellRing size={20} /> : <Bell size={20} />}
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-bold text-zinc-100">{t("push.title")}</p>
          <p className="text-xs text-zinc-400 mt-0.5">{on ? t("push.onDesc") : t("push.offDesc")}</p>
        </div>
        <button
          onClick={on ? disable : enable}
          disabled={state === "busy"}
          className={`flex-shrink-0 rounded-lg px-3 py-2 text-sm font-bold transition-opacity hover:opacity-90 disabled:opacity-50 ${
            on ? "bg-zinc-800 text-zinc-300 border border-zinc-700" : "bg-yellow-400 text-zinc-950"
          }`}
        >
          {on ? <BellOff size={15} /> : t("push.enable")}
        </button>
      </div>
      {on && (
        <button
          onClick={async () => { try { await pushTest(); toast.success(t("push.testSent")); } catch { toast.error(t("push.error")); } }}
          className="mt-3 w-full rounded-lg border border-zinc-700 py-2 text-xs font-semibold text-zinc-300 hover:bg-zinc-800 transition-colors"
        >
          {t("push.testButton")}
        </button>
      )}
    </div>
  );
}
