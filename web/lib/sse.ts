"use client";

import { useEffect, useRef, useState } from "react";

// Единый канал реального времени для всего приложения: одно SSE-соединение на
// вкладку, мультиплексирующее именованные события (match_update, audit,
// notification, presence, chat). Компоненты подписываются на нужный тип через
// useSSE; соединение поднимается лениво по первому подписчику и закрывается,
// когда подписчиков не осталось. Токен (для личных/админских топиков) уходит
// query-параметром, т.к. EventSource не умеет слать заголовки.

const API_URL = process.env.NEXT_PUBLIC_API_URL || "";

type Handler = (data: any) => void;
type StatusHandler = (live: boolean) => void;

function getToken(): string | null {
  return typeof window !== "undefined" ? localStorage.getItem("efootball_jwt") : null;
}

class SSEClient {
  private es: EventSource | null = null;
  private listeners = new Map<string, Set<Handler>>();
  private attached = new Set<string>(); // типы, навешенные на текущий es
  private statusListeners = new Set<StatusHandler>();
  private live = false;
  private token: string | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private backoff = 1000;

  constructor() {
    if (typeof window !== "undefined") {
      // Открытый EventSource блокирует bfcache — закрываем при уходе со
      // страницы и переоткрываем при возврате/появлении вкладки.
      window.addEventListener("pagehide", () => this.close());
      window.addEventListener("pageshow", () => this.ensure());
      document.addEventListener("visibilitychange", () => {
        if (document.visibilityState === "visible") this.ensure();
      });
    }
  }

  subscribe(type: string, handler: Handler): () => void {
    let set = this.listeners.get(type);
    if (!set) {
      set = new Set();
      this.listeners.set(type, set);
    }
    set.add(handler);
    this.ensure();
    if (this.es && !this.attached.has(type)) this.attach(type);

    return () => {
      const s = this.listeners.get(type);
      if (!s) return;
      s.delete(handler);
      if (s.size === 0) this.listeners.delete(type);
      if (this.listeners.size === 0) this.close();
    };
  }

  // Подписка на статус соединения (для live-индикатора). Сразу отдаёт текущее.
  subscribeStatus(handler: StatusHandler): () => void {
    this.statusListeners.add(handler);
    handler(this.live);
    return () => { this.statusListeners.delete(handler); };
  }

  private setLive(v: boolean) {
    if (this.live === v) return;
    this.live = v;
    this.statusListeners.forEach((h) => { try { h(v); } catch { /* изолируем */ } });
  }

  private ensure() {
    if (typeof window === "undefined" || this.listeners.size === 0) return;
    const token = getToken();
    if (this.es && this.token !== token) this.close(); // токен сменился (логин/логаут)
    if (this.es) return;

    const url = `${API_URL}/api/events${token ? `?token=${encodeURIComponent(token)}` : ""}`;
    const es = new EventSource(url);
    this.es = es;
    this.token = token;
    this.attached.clear();
    this.listeners.forEach((_set, type) => this.attach(type));

    es.onopen = () => { this.backoff = 1000; this.setLive(true); };
    es.onerror = () => {
      this.setLive(false);
      // EventSource сам переподключается, пока он в CONNECTING. Если браузер
      // закрыл соединение (CLOSED) — пересоздаём с нарастающей паузой.
      if (es.readyState === EventSource.CLOSED) {
        this.es = null;
        this.attached.clear();
        this.scheduleReconnect();
      }
    };
  }

  private attach(type: string) {
    if (!this.es) return;
    this.attached.add(type);
    this.es.addEventListener(type, (ev: MessageEvent) => {
      let data: any = ev.data;
      try { data = JSON.parse(ev.data); } catch { /* оставляем строкой */ }
      this.listeners.get(type)?.forEach((h) => {
        try { h(data); } catch { /* один обработчик не должен ронять остальные */ }
      });
    });
  }

  private scheduleReconnect() {
    if (this.reconnectTimer || this.listeners.size === 0) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.ensure();
    }, this.backoff);
    this.backoff = Math.min(this.backoff * 2, 30000);
  }

  private close() {
    if (this.reconnectTimer) { clearTimeout(this.reconnectTimer); this.reconnectTimer = null; }
    if (this.es) { this.es.close(); this.es = null; }
    this.attached.clear();
    this.token = null;
    this.setLive(false);
  }
}

export const sse = new SSEClient();

// useSSE подписывает компонент на событие type. handler хранится в ref, поэтому
// перерисовки не пересоздают подписку, а обработчик всегда актуален.
export function useSSE(type: string, handler: Handler, enabled = true) {
  const ref = useRef(handler);
  ref.current = handler;
  useEffect(() => {
    if (!enabled) return;
    return sse.subscribe(type, (d) => ref.current(d));
  }, [type, enabled]);
}

export interface LeagueSSEState {
  /** Соединение открыто — события приходят в реальном времени. */
  live: boolean;
  /** Время последнего события по этой лиге (Date.now()), null если не было. */
  lastEventAt: number | null;
}

/**
 * Подписка на live-события конкретной лиги поверх общего канала.
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
    const unsubEvent = sse.subscribe("match_update", (data) => {
      if (data && Number(data.league_id) === leagueId) {
        setState({ live: true, lastEventAt: Date.now() });
        onUpdateRef.current(Number(data.match_id) || 0);
      }
    });
    const unsubStatus = sse.subscribeStatus((live) => setState((s) => ({ ...s, live })));
    return () => { unsubEvent(); unsubStatus(); };
  }, [leagueId]);

  return state;
}
