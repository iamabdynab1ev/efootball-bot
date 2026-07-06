"use client";

import { useEffect } from "react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useUnreadTotal } from "@/lib/chat";
import { useNotifications } from "@/lib/notifications";
import { preloadSounds } from "@/lib/sound";

// Счётчик непрочитанных (ЛС + уведомления) в заголовке вкладки: «(3) eFootLeague».
// Виден даже когда вкладка в фоне — человек замечает новое, не открывая её.
function TitleUnread() {
  const unreadDM = useUnreadTotal(true);
  const { unread } = useNotifications();

  useEffect(() => {
    const base = document.title.replace(/^\(\d+\+?\)\s*/, "");
    const total = unreadDM + unread;
    document.title = total > 0 ? `(${total > 99 ? "99+" : total}) ${base}` : base;
  }, [unreadDM, unread]);

  return null;
}

// AppSignals — невидимая «нервная система» приложения: предзагружает звуковые
// ассеты (autoplay разблокируется первым жестом) и ведёт бейдж непрочитанных
// в title.
export function AppSignals() {
  const { user } = useAuth();

  useEffect(() => { preloadSounds(); }, []);

  // Видимость приложения для сервера: вкладка на переднем плане → события
  // доставляются внутри приложения (звук+тост), web-push не дублирует их.
  useEffect(() => {
    if (!user) return;
    const beat = (on: boolean) => { api.post("/api/app/focus", { on }).catch(() => { /* не критично */ }); };
    const sync = () => beat(document.visibilityState === "visible");
    sync();
    const iv = setInterval(() => { if (document.visibilityState === "visible") beat(true); }, 60_000);
    document.addEventListener("visibilitychange", sync);
    return () => { clearInterval(iv); document.removeEventListener("visibilitychange", sync); beat(false); };
  }, [user]);

  return user ? <TitleUnread /> : null;
}
