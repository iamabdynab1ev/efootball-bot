"use client";

import { useEffect, useMemo, useState } from "react";
import { MessageSquare, Archive } from "lucide-react";
import { fetchRoomReads, useChatMembers, useChatRoom, useChatRooms } from "@/lib/chat";
import { ChatThread } from "@/components/ChatThread";
import { cn } from "@/lib/utils";

interface Props {
  leagueId: number;
  currentUserId?: number;
  isAdmin?: boolean;
  // "card" — ограниченная высота с рамкой (во вкладке); "full" — на всю высоту
  // родителя без рамки (отдельная страница чата).
  variant?: "card" | "full";
}

export function ChatPanel({ leagueId, currentUserId, isAdmin = false, variant = "card" }: Props) {
  const { rooms, loading: roomsLoading } = useChatRooms(leagueId);
  const [roomId, setRoomId] = useState<number | null>(null);
  const room = useMemo(() => rooms.find((r) => r.id === roomId) ?? null, [rooms, roomId]);

  useEffect(() => {
    if (roomId == null && rooms.length > 0) setRoomId(rooms[0].id);
  }, [rooms, roomId]);

  const { messages, hasMore, send, loadOlder } = useChatRoom(roomId);
  const members = useChatMembers(roomId); // @упоминания — участники ИМЕННО этой комнаты

  // Начальный прогресс прочтения участников комнаты (для отметок «прочитано»).
  const [initialReads, setInitialReads] = useState<Record<number, number>>({});
  useEffect(() => {
    if (roomId == null) { setInitialReads({}); return; }
    let on = true;
    fetchRoomReads(roomId).then((m) => { if (on) setInitialReads(m); }).catch(() => { /* без галочек */ });
    return () => { on = false; };
  }, [roomId]);

  if (roomsLoading) {
    return <div className="py-8 text-center text-sm text-zinc-500">Загрузка чата…</div>;
  }
  if (rooms.length === 0) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-4 py-10 text-center">
        <MessageSquare size={22} className="mx-auto mb-2 text-zinc-600" />
        <p className="text-sm text-zinc-500">Чат станет доступен после жеребьёвки и распределения по группам.</p>
      </div>
    );
  }

  return (
    <div className={cn(
      "flex flex-col bg-zinc-900 overflow-hidden",
      variant === "full" ? "h-full" : "h-[75dvh] sm:h-[600px] rounded-xl border border-zinc-800",
    )}>
      {/* Комнаты */}
      {rooms.length > 1 && (
        <div className="flex gap-1.5 border-b border-white/5 bg-zinc-950/40 px-2 py-2 overflow-x-auto scrollbar-none flex-shrink-0">
          {rooms.map((r) => (
            <button
              key={r.id}
              onClick={() => setRoomId(r.id)}
              className={cn(
                "flex flex-shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-3.5 py-1.5 text-xs font-semibold transition-all",
                r.id === roomId
                  ? "bg-gradient-to-br from-[#d9ff3d] to-[#a3cc1e] text-zinc-950 shadow-[0_2px_10px_rgba(200,241,53,0.25)]"
                  : "bg-white/[0.04] text-zinc-400 hover:bg-white/[0.08] hover:text-zinc-100"
              )}
            >
              {r.title}
              {r.archived && <Archive size={11} className="opacity-70" />}
              {r.id !== roomId && (r.unread ?? 0) > 0 && (
                <span className="flex h-4 min-w-[16px] items-center justify-center rounded-full bg-yellow-400 px-1 text-[10px] font-bold text-zinc-900">
                  {(r.unread ?? 0) > 99 ? "99+" : r.unread}
                </span>
              )}
            </button>
          ))}
        </div>
      )}

      <div className="flex-1 min-h-0">
        <ChatThread
          key={roomId ?? 0}
          messages={messages}
          hasMore={hasMore}
          loadOlder={loadOlder}
          send={send}
          currentUserId={currentUserId}
          isAdmin={isAdmin}
          archived={!!room?.archived}
          archivedNote="Турнир завершён — чат в архиве"
          members={members}
          resetKey={roomId ?? 0}
          roomId={roomId ?? undefined}
          showReceipts
          initialReads={initialReads}
        />
      </div>
    </div>
  );
}
