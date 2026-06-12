"use client";

import { useEffect, useRef, useState } from "react";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "";

export interface LeagueSSEState {
  /** Соединение открыто — события приходят в реальном времени. */
  live: boolean;
  /** Время последнего события по этой лиге (Date.now()), null если не было. */
  lastEventAt: number | null;
}

/**
 * Подписка на live-события лиги через SSE.
 * Соединение само переподключается с экспоненциальной задержкой (1s → 30s)
 * и заново открывается при возврате вкладки из фона (visibilitychange) —
 * прежняя версия закрывала EventSource навсегда после первой ошибки.
 *
 * onUpdate получает match_id изменившегося матча (0, если бэкенд его не прислал).
 */
export function useLeagueSSE(
  leagueId: number,
  onUpdate: (matchId: number) => void,
): LeagueSSEState {
  const [state, setState] = useState<LeagueSSEState>({ live: false, lastEventAt: null });
  const onUpdateRef = useRef(onUpdate);
  onUpdateRef.current = onUpdate;

  useEffect(() => {
    if (!leagueId) return;

    let es: EventSource | null = null;
    let retryMs = 1000;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let closed = false;

    const connect = () => {
      if (closed) return;
      es = new EventSource(`${API_URL}/api/events`);

      es.onopen = () => {
        retryMs = 1000;
        setState((s) => ({ ...s, live: true }));
      };

      es.onmessage = (e) => {
        try {
          const data = JSON.parse(e.data);
          if (data.league_id === leagueId) {
            setState({ live: true, lastEventAt: Date.now() });
            onUpdateRef.current(Number(data.match_id) || 0);
          }
        } catch {
          // не-JSON heartbeat — игнорируем
        }
      };

      es.onerror = () => {
        es?.close();
        setState((s) => ({ ...s, live: false }));
        if (closed) return;
        retryTimer = setTimeout(connect, retryMs);
        retryMs = Math.min(retryMs * 2, 30_000);
      };
    };

    const onVisible = () => {
      // Вкладка вернулась из фона: если соединение мертво — переоткрываем сразу.
      if (document.visibilityState === "visible" && es?.readyState === EventSource.CLOSED) {
        if (retryTimer) clearTimeout(retryTimer);
        retryMs = 1000;
        connect();
      }
    };

    // Открытый EventSource блокирует back/forward-кеш браузера —
    // закрываем при уходе со страницы и переоткрываем при возврате из bfcache.
    const onPageHide = () => {
      if (retryTimer) clearTimeout(retryTimer);
      es?.close();
      setState((s) => ({ ...s, live: false }));
    };
    const onPageShow = () => {
      if (!closed && es?.readyState === EventSource.CLOSED) {
        retryMs = 1000;
        connect();
      }
    };

    connect();
    document.addEventListener("visibilitychange", onVisible);
    window.addEventListener("pagehide", onPageHide);
    window.addEventListener("pageshow", onPageShow);

    return () => {
      closed = true;
      if (retryTimer) clearTimeout(retryTimer);
      document.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener("pagehide", onPageHide);
      window.removeEventListener("pageshow", onPageShow);
      es?.close();
    };
  }, [leagueId]);

  return state;
}
