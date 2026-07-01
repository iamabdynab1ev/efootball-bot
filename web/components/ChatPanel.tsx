"use client";

import { useEffect, useMemo, useState } from "react";
import { MessageSquare, Archive } from "lucide-react";
import { useChatMembers, useChatRoom, useChatRooms } from "@/lib/chat";
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
        <div className="flex gap-1 border-b border-zinc-800 px-2 py-2 overflow-x-auto scrollbar-none flex-shrink-0">
          {rooms.map((r) => (
            <button
              key={r.id}
              onClick={() => setRoomId(r.id)}
              className={cn(
                "flex flex-shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-3.5 py-1.5 text-xs font-semibold transition-colors",
                r.id === roomId ? "bg-yellow-400 text-zinc-900" : "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
              )}
            >
              {r.title}
              {r.archived && <Archive size={11} className="opacity-70" />}
            </button>
          ))}
        </div>
      )}

      <div className="flex-1 min-h-0">
        <ChatThread
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
        />
      </div>
    </div>
  );
}
