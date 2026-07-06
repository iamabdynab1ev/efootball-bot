"use client";

import { useEffect } from "react";
import { useAuth } from "@/lib/auth";
import { useUnreadTotal } from "@/lib/chat";
import { useNotifications } from "@/lib/notifications";
import { loadServerSoundPrefs, preloadSounds } from "@/lib/sound";

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
// ассеты (autoplay разблокируется первым жестом), подтягивает сохранённые в
// профиле настройки звука и ведёт бейдж непрочитанных в title.
export function AppSignals() {
  const { user } = useAuth();

  useEffect(() => { preloadSounds(); }, []);
  useEffect(() => { if (user) void loadServerSoundPrefs(); }, [user]);

  return user ? <TitleUnread /> : null;
}
