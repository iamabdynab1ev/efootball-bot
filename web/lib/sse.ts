"use client";

import { useEffect } from "react";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "";

export function useLeagueSSE(leagueId: number, onUpdate: () => void) {
  useEffect(() => {
    if (!leagueId) return;
    const url = `${API_URL}/api/events`;
    const es = new EventSource(url);
    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        if (data.league_id === leagueId) {
          onUpdate();
        }
      } catch {}
    };
    es.onerror = () => es.close();
    return () => es.close();
  }, [leagueId, onUpdate]);
}
