"use client";

import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { Send, Trash2, MessageSquare, Archive } from "lucide-react";
import { api } from "@/lib/api";
import { useChatMembers, useChatRoom, useChatRooms, type ChatMessage } from "@/lib/chat";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { cn } from "@/lib/utils";

interface Props {
  leagueId: number;
  currentUserId?: number;
  isAdmin?: boolean;
  // "card" — ограниченная высота с рамкой (во вкладке); "full" — на всю высоту
  // родителя без рамки (отдельная страница чата).
  variant?: "card" | "full";
}

function fmtTime(iso: string) {
  return new Date(iso).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
}

function dayLabel(iso: string) {
  const d = new Date(iso);
  const today = new Date();
  const yest = new Date(); yest.setDate(today.getDate() - 1);
  if (d.toDateString() === today.toDateString()) return "Сегодня";
  if (d.toDateString() === yest.toDateString()) return "Вчера";
  return d.toLocaleDateString("ru-RU", { day: "2-digit", month: "long" });
}

// Подсветка @упоминаний.
function renderBody(body: string) {
  return body.split(/(@[\wА-Яа-яЁё]+)/g).map((p, i) =>
    p.startsWith("@")
      ? <span key={i} className="font-semibold text-yellow-400">{p}</span>
      : <span key={i}>{p}</span>
  );
}

export function ChatPanel({ leagueId, currentUserId, isAdmin, variant = "card" }: Props) {
  const { rooms, loading: roomsLoading } = useChatRooms(leagueId);
  const [roomId, setRoomId] = useState<number | null>(null);
  const room = useMemo(() => rooms.find((r) => r.id === roomId) ?? null, [rooms, roomId]);

  useEffect(() => {
    if (roomId == null && rooms.length > 0) setRoomId(rooms[0].id);
  }, [rooms, roomId]);

  const { messages, hasMore, send, loadOlder } = useChatRoom(roomId);
  const members = useChatMembers(roomId); // @упоминания — участники ИМЕННО этой комнаты

  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const [mentionQuery, setMentionQuery] = useState<string | null>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  const scrollRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const lastIdRef = useRef(0);
  const prevHeightRef = useRef(0); // сохранение позиции при подгрузке старых

  const nearBottom = () => {
    const el = scrollRef.current;
    if (!el) return true;
    return el.scrollHeight - el.scrollTop - el.clientHeight < 140;
  };

  // Автоскролл вниз при новом сообщении — только если пользователь у низа.
  useEffect(() => {
    const lastId = messages.length ? messages[messages.length - 1].id : 0;
    if (lastId !== lastIdRef.current) {
      const wasInitial = lastIdRef.current === 0;
      const jump = lastId > lastIdRef.current && (wasInitial || nearBottom());
      lastIdRef.current = lastId;
      if (jump) bottomRef.current?.scrollIntoView({ behavior: wasInitial ? "auto" : "smooth" });
    }
  }, [messages]);

  useEffect(() => {
    lastIdRef.current = 0;
    setText(""); setMentionQuery(null);
  }, [roomId]);

  // Автоувеличение высоты поля ввода (как в мессенджерах).
  useLayoutEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    el.style.height = "0px";
    el.style.height = Math.min(el.scrollHeight, 120) + "px";
  }, [text]);

  const onChange = (v: string) => {
    setText(v);
    const caret = inputRef.current?.selectionStart ?? v.length;
    const m = v.slice(0, caret).match(/@([\wА-Яа-яЁё]*)$/);
    setMentionQuery(m ? m[1].toLowerCase() : null);
  };

  const mentionMatches = useMemo(() => {
    if (mentionQuery == null) return [];
    return members.filter((m) => m.display_name.toLowerCase().includes(mentionQuery)).slice(0, 6);
  }, [mentionQuery, members]);

  const applyMention = (name: string) => {
    const first = name.split(" ")[0];
    setText((t) => t.replace(/@([\wА-Яа-яЁё]*)$/, `@${first} `));
    setMentionQuery(null);
    inputRef.current?.focus();
  };

  const submit = async () => {
    const body = text.trim();
    if (!body || sending || !room || room.archived) return;
    setSending(true);
    try {
      await send(body);
      setText("");
      setMentionQuery(null);
      requestAnimationFrame(() => bottomRef.current?.scrollIntoView({ behavior: "smooth" }));
    } finally {
      setSending(false);
    }
  };

  const onLoadOlder = async () => {
    prevHeightRef.current = scrollRef.current?.scrollHeight ?? 0;
    await loadOlder();
  };
  // После подгрузки старых — сохраняем визуальную позицию (без прыжка).
  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (el && prevHeightRef.current) {
      el.scrollTop += el.scrollHeight - prevHeightRef.current;
      prevHeightRef.current = 0;
    }
  }, [messages]);

  const onDelete = async (m: ChatMessage) => {
    try { await api.delete(`/api/admin/chat/messages/${m.id}`); } catch { /* no-op */ }
  };

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

  let prevAuthor: number | null | undefined;
  let prevDay = "";
  let prevTime = 0;

  return (
    <div className={cn(
      "flex flex-col bg-zinc-900 overflow-hidden",
      variant === "full" ? "h-full" : "h-[75dvh] sm:h-[600px] rounded-xl border border-zinc-800",
    )}>
      {/* Комнаты */}
      {rooms.length > 1 && (
        <div className="flex gap-1 border-b border-zinc-800 px-2 py-2 overflow-x-auto flex-shrink-0">
          {rooms.map((r) => (
            <button
              key={r.id}
              onClick={() => setRoomId(r.id)}
              className={cn(
                "flex items-center gap-1.5 whitespace-nowrap rounded-full px-3.5 py-1.5 text-xs font-semibold transition-colors",
                r.id === roomId ? "bg-yellow-400 text-zinc-900" : "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
              )}
            >
              {r.title}
              {r.archived && <Archive size={11} className="opacity-70" />}
            </button>
          ))}
        </div>
      )}

      {/* Сообщения */}
      <div ref={scrollRef} className="flex-1 min-h-0 overflow-y-auto px-3 py-3 overscroll-contain">
        {hasMore && (
          <div className="text-center pb-2">
            <button onClick={onLoadOlder} className="rounded-full border border-zinc-700 px-3 py-1 text-[11px] text-zinc-400 hover:text-zinc-200">
              Показать ранние
            </button>
          </div>
        )}
        {messages.length === 0 ? (
          <div className="py-12 text-center text-sm text-zinc-600">Пока нет сообщений — начните общение</div>
        ) : (
          messages.map((m) => {
            const own = m.user_id === currentUserId;
            const t = new Date(m.created_at).getTime();
            const day = dayLabel(m.created_at);
            const grouped = m.user_id === prevAuthor && day === prevDay && t - prevTime < 5 * 60 * 1000;
            const showDay = day !== prevDay;
            prevAuthor = m.user_id; prevDay = day; prevTime = t;

            return (
              <div key={m.id}>
                {showDay && (
                  <div className="flex justify-center py-2">
                    <span className="rounded-full bg-zinc-800 px-3 py-0.5 text-[10px] font-medium text-zinc-500">{day}</span>
                  </div>
                )}
                <div className={cn("flex items-end gap-2 group", own ? "flex-row-reverse" : "", grouped ? "mt-0.5" : "mt-2")}>
                  {/* Аватар = логотип клуба автора, с обеих сторон — сразу видно кто пишет */}
                  {grouped
                    ? <div className="w-[26px] flex-shrink-0" />
                    : <PlayerAvatar displayName={m.author_name} favoriteClub={m.author_club} size={26} />}
                  <div className={cn(
                    "max-w-[80%] px-3 py-1.5 shadow-sm",
                    own
                      ? "bg-yellow-500/15 rounded-2xl rounded-br-md"
                      : "bg-zinc-800 rounded-2xl rounded-bl-md",
                    grouped && (own ? "rounded-tr-md" : "rounded-tl-md"),
                  )}>
                    {!grouped && (
                      <p className={cn("text-[11px] font-semibold mb-0.5", own ? "text-right text-yellow-600/90" : "text-yellow-500/90")}>
                        {own ? "Вы" : m.author_name}
                      </p>
                    )}
                    {m.deleted ? (
                      <p className="text-xs italic text-zinc-500">сообщение удалено</p>
                    ) : (
                      <p className="text-[15px] leading-snug text-zinc-100 break-words whitespace-pre-wrap">{renderBody(m.body)}</p>
                    )}
                    <span className={cn("block text-[10px] text-zinc-500 mt-0.5", own ? "text-right" : "")}>{fmtTime(m.created_at)}</span>
                  </div>
                  {isAdmin && !m.deleted && (
                    <button
                      onClick={() => onDelete(m)}
                      className="opacity-0 group-hover:opacity-100 transition-opacity rounded p-1 text-zinc-600 hover:text-red-400 self-center"
                      title="Удалить сообщение"
                    >
                      <Trash2 size={13} />
                    </button>
                  )}
                </div>
              </div>
            );
          })
        )}
        <div ref={bottomRef} />
      </div>

      {/* Ввод */}
      {room?.archived ? (
        <div className="border-t border-zinc-800 px-4 py-3 text-center text-xs text-zinc-500 flex-shrink-0">
          <Archive size={13} className="inline mr-1" /> Турнир завершён — чат в архиве
        </div>
      ) : (
        <div className="relative border-t border-zinc-800 p-2 flex-shrink-0 pb-[max(0.5rem,env(safe-area-inset-bottom))]">
          {mentionMatches.length > 0 && (
            <div className="absolute bottom-full left-2 right-2 mb-1 max-h-52 overflow-y-auto rounded-xl border border-zinc-700 bg-zinc-900 shadow-2xl">
              {mentionMatches.map((m) => (
                <button
                  key={m.user_id}
                  onClick={() => applyMention(m.display_name)}
                  className="flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm text-zinc-200 hover:bg-zinc-800"
                >
                  <PlayerAvatar displayName={m.display_name} favoriteClub={m.favorite_club} size={22} />
                  <span className="truncate">{m.display_name}</span>
                </button>
              ))}
            </div>
          )}
          <div className="flex items-end gap-2">
            <textarea
              ref={inputRef}
              rows={1}
              value={text}
              onChange={(e) => onChange(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); submit(); } }}
              placeholder="Сообщение…"
              maxLength={2000}
              className="flex-1 resize-none rounded-2xl border border-zinc-700 bg-zinc-950 px-3.5 py-2 text-[15px] text-zinc-100 placeholder-zinc-600 focus:border-yellow-400 focus:outline-none leading-snug max-h-[120px]"
            />
            <button
              onClick={submit}
              disabled={sending || !text.trim()}
              className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-yellow-400 text-zinc-900 disabled:opacity-40 active:scale-95 transition-transform"
              aria-label="Отправить"
            >
              <Send size={17} />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
