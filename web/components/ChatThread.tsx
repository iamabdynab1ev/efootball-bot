"use client";

import { memo, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { Send, Trash2, ChevronDown, Check, CheckCheck } from "lucide-react";
import { api } from "@/lib/api";
import { markRead, sendTyping, type ChatMessage, type ChatMember } from "@/lib/chat";
import { useSSE } from "@/lib/sse";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { cn } from "@/lib/utils";

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

interface RowData {
  m: ChatMessage;
  own: boolean;
  grouped: boolean;  // подряд от того же автора в пределах 5 мин — тесная группа
  showDay: boolean;  // сменился день — показать разделитель
  day: string;
  showName: boolean; // показывать имя автора (в ЛС/у своих не нужно)
}

// MessageRow мемоизирован: при новом сообщении перерисовывается только оно, а не
// весь список (объекты старых сообщений сохраняют идентичность).
const MessageRow = memo(function MessageRow({
  m, own, grouped, showDay, day, showName, isAdmin, onDelete, showReceipts, otherReads,
}: RowData & { isAdmin: boolean; onDelete: (id: number) => void; showReceipts: boolean; otherReads: number[] }) {
  // Сколько собеседников прочитали это сообщение (для ✓/✓✓ и счётчика в группе).
  const receipts = own && showReceipts && !m.deleted;
  const total = otherReads.length;
  const readers = receipts ? otherReads.filter((v) => v >= m.id).length : 0;
  const allRead = receipts && total > 0 && readers === total;
  return (
    <div>
      {showDay && (
        <div className="flex justify-center py-2">
          <span className="rounded-full bg-zinc-800 px-3 py-0.5 text-[10px] font-medium text-zinc-500">{day}</span>
        </div>
      )}
      <div className={cn("flex items-end gap-2 group", own ? "flex-row-reverse" : "", grouped ? "mt-0.5" : "mt-2")}>
        {/* Аватар = логотип клуба автора; резервируем место у сгруппированных */}
        {grouped
          ? <div className="w-[26px] flex-shrink-0" />
          : <PlayerAvatar displayName={m.author_name} favoriteClub={m.author_club} size={26} />}
        <div className={cn(
          "min-w-0 max-w-[85%] sm:max-w-[70%] px-3 py-1.5 shadow-sm",
          own ? "bg-yellow-500/15 rounded-2xl rounded-br-md" : "bg-zinc-800 rounded-2xl rounded-bl-md",
          grouped && (own ? "rounded-tr-md" : "rounded-tl-md"),
        )}>
          {showName && !grouped && (
            <p className={cn("text-[11px] font-semibold mb-0.5", own ? "text-right text-yellow-600/90" : "text-yellow-500/90")}>
              {own ? "Вы" : m.author_name}
            </p>
          )}
          {m.deleted ? (
            <p className="text-xs italic text-zinc-500">сообщение удалено</p>
          ) : (
            <p className="text-[15px] leading-snug text-zinc-100 whitespace-pre-wrap break-words [overflow-wrap:anywhere]">
              {renderBody(m.body)}
            </p>
          )}
          <div className={cn("flex items-center gap-1 mt-0.5", own ? "justify-end" : "")}>
            <span className="text-[10px] text-zinc-500">{fmtTime(m.created_at)}</span>
            {receipts && (
              readers === 0
                ? <Check size={13} className="text-zinc-500" />
                : (
                  <span className={cn("flex items-center gap-0.5", allRead ? "text-sky-400" : "text-sky-400/60")}>
                    <CheckCheck size={13} />
                    {total > 1 && <span className="text-[9px] font-semibold">{readers}</span>}
                  </span>
                )
            )}
          </div>
        </div>
        {isAdmin && !m.deleted && (
          <button
            onClick={() => onDelete(m.id)}
            className="opacity-0 group-hover:opacity-100 transition-opacity rounded p-1 text-zinc-600 hover:text-red-400 self-center"
            title="Удалить сообщение"
          >
            <Trash2 size={13} />
          </button>
        )}
      </div>
    </div>
  );
});

export interface ChatThreadProps {
  messages: ChatMessage[];
  hasMore: boolean;
  loadOlder: () => Promise<void> | void;
  send: (body: string) => Promise<void>;
  currentUserId?: number;
  isAdmin?: boolean;
  archived?: boolean;
  archivedNote?: string;
  /** Участники для @упоминаний (в ЛС не передаём — упоминания не нужны). */
  members?: ChatMember[];
  /** Показывать имя автора над чужими сообщениями (в ЛС не нужно). */
  showAuthorNames?: boolean;
  /** Плейсхолдер поля ввода. */
  placeholder?: string;
  /** Активна ли комната (для сброса скролла/ввода при переключении). */
  resetKey?: string | number;
  /** id комнаты — включает отметки прочтения и «печатает…» (для ЛС). */
  roomId?: number;
  /** Показывать ✓/✓✓ на своих сообщениях. */
  showReceipts?: boolean;
  /** Начальный прогресс прочтения участников {userId: lastRead} (для ✓✓). */
  initialReads?: Record<number, number>;
  /** Слать/принимать «печатает…». */
  showTyping?: boolean;
  /** Колбэк: собеседник печатает / перестал (для статуса в шапке). */
  onPeerTyping?: (typing: boolean) => void;
}

// ChatThread — переиспользуемая лента диалога: сообщения (bottom-anchored),
// авто-скролл, кнопка «вниз», @упоминания, поле ввода. Используется и в чате
// турнира (на комнату), и в личных сообщениях.
export function ChatThread({
  messages, hasMore, loadOlder, send,
  currentUserId, isAdmin = false, archived = false,
  archivedNote = "Чат в архиве", members = [], showAuthorNames = true,
  placeholder = "Сообщение…", resetKey,
  roomId, showReceipts = false, initialReads, showTyping = false, onPeerTyping,
}: ChatThreadProps) {
  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const [mentionQuery, setMentionQuery] = useState<string | null>(null);
  const [showJump, setShowJump] = useState(false);
  // Прогресс прочтения по участникам {userId: lastRead}. Для ЛС — один собеседник,
  // для группы — все участники (сколько прочитали каждое сообщение).
  const [readsByUser, setReadsByUser] = useState<Record<number, number>>(initialReads ?? {});
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const typingSentRef = useRef(0);   // троттлинг отправки «печатает…»
  const readSentRef = useRef(0);     // до какого id уже отметили прочтение
  const typingTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Начальный прогресс может приехать позже монтирования — вливаем по максимуму.
  useEffect(() => {
    if (!initialReads) return;
    setReadsByUser((prev) => {
      let changed = false;
      const next = { ...prev };
      for (const [uid, lr] of Object.entries(initialReads)) {
        if ((next[+uid] ?? 0) < lr) { next[+uid] = lr; changed = true; }
      }
      return changed ? next : prev;
    });
  }, [initialReads]);

  // Живое обновление ✓✓: кто-то из участников дочитал.
  useSSE("chat_read", (d: any) => {
    if (!showReceipts || roomId == null || !d || d.room_id !== roomId) return;
    const uid = Number(d.user_id) || 0;
    const lr = Number(d.last_read_id) || 0;
    if (!uid) return;
    setReadsByUser((prev) => (prev[uid] >= lr ? prev : { ...prev, [uid]: lr }));
  }, showReceipts && roomId != null);

  // Массив last_read остальных участников (без себя) — для подсчёта прочтений.
  const otherReads = useMemo(
    () => Object.entries(readsByUser)
      .filter(([uid]) => Number(uid) !== currentUserId)
      .map(([, lr]) => lr),
    [readsByUser, currentUserId],
  );

  // «печатает…»: показываем и гасим через 3.5с тишины.
  useSSE("chat_typing", (d: any) => {
    if (!showTyping || roomId == null || !d || d.room_id !== roomId) return;
    onPeerTyping?.(true);
    if (typingTimerRef.current) clearTimeout(typingTimerRef.current);
    typingTimerRef.current = setTimeout(() => onPeerTyping?.(false), 3500);
  }, showTyping && roomId != null);

  useEffect(() => () => { if (typingTimerRef.current) clearTimeout(typingTimerRef.current); }, []);

  // Отмечаем прочитанным до последнего сообщения (при открытии и на новые входящие).
  useEffect(() => {
    if (roomId == null || messages.length === 0) return;
    const lastId = messages[messages.length - 1].id;
    if (lastId > readSentRef.current) {
      readSentRef.current = lastId;
      markRead(roomId, lastId);
    }
  }, [roomId, messages]);

  const scrollRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const lastIdRef = useRef(0);
  const stickRef = useRef(true);
  const prevHeightRef = useRef(0);

  const rows = useMemo<RowData[]>(() => {
    const out: RowData[] = [];
    let prevAuthor: number | null | undefined;
    let prevDay = "";
    let prevTime = 0;
    for (const m of messages) {
      const day = dayLabel(m.created_at);
      const t = new Date(m.created_at).getTime();
      const grouped = m.user_id === prevAuthor && day === prevDay && t - prevTime < 5 * 60 * 1000;
      out.push({
        m, own: m.user_id === currentUserId, grouped,
        showDay: day !== prevDay, day, showName: showAuthorNames,
      });
      prevAuthor = m.user_id; prevDay = day; prevTime = t;
    }
    return out;
  }, [messages, currentUserId, showAuthorNames]);

  const onScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 120;
    stickRef.current = nearBottom;
    setShowJump(!nearBottom);
  }, []);

  const jumpToBottom = useCallback(() => {
    stickRef.current = true;
    setShowJump(false);
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  // Автоскролл вниз при новом сообщении — только если пользователь у низа.
  useEffect(() => {
    const lastId = messages.length ? messages[messages.length - 1].id : 0;
    if (lastId === lastIdRef.current) return;
    const initial = lastIdRef.current === 0;
    const grew = lastId > lastIdRef.current;
    lastIdRef.current = lastId;
    if (grew && (initial || stickRef.current)) {
      bottomRef.current?.scrollIntoView({ behavior: initial ? "auto" : "smooth" });
    }
  }, [messages]);

  // Смена комнаты — сбрасываем состояние скролла/ввода/прочтения.
  useEffect(() => {
    lastIdRef.current = 0;
    stickRef.current = true;
    readSentRef.current = 0;
    setText(""); setMentionQuery(null); setShowJump(false);
    onPeerTyping?.(false);
  }, [resetKey]); // eslint-disable-line react-hooks/exhaustive-deps

  // Автовысота поля ввода.
  useLayoutEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    el.style.height = "0px";
    el.style.height = Math.min(el.scrollHeight, 120) + "px";
  }, [text]);

  const onChange = (v: string) => {
    setText(v);
    // «печатает…»: троттлим отправку до раза в 2с.
    if (showTyping && roomId != null && v.trim()) {
      const now = Date.now();
      if (now - typingSentRef.current > 2000) {
        typingSentRef.current = now;
        sendTyping(roomId);
      }
    }
    if (members.length === 0) return; // упоминания не нужны (напр. в ЛС)
    const caret = inputRef.current?.selectionStart ?? v.length;
    const m = v.slice(0, caret).match(/@([\wА-Яа-яЁё]*)$/);
    setMentionQuery(m ? m[1].toLowerCase() : null);
  };

  const mentionMatches = useMemo(() => {
    if (mentionQuery == null) return [];
    // Показываем всех подходящих участников (список прокручивается), а не первые 6.
    return members.filter((m) => m.display_name.toLowerCase().includes(mentionQuery)).slice(0, 50);
  }, [mentionQuery, members]);

  const applyMention = (name: string) => {
    const first = name.split(" ")[0];
    setText((t) => t.replace(/@([\wА-Яа-яЁё]*)$/, `@${first} `));
    setMentionQuery(null);
    inputRef.current?.focus();
  };

  const submit = async () => {
    const body = text.trim();
    if (!body || sending || archived) return;
    setSending(true);
    try {
      stickRef.current = true;
      await send(body);
      setText("");
      setMentionQuery(null);
      requestAnimationFrame(() => bottomRef.current?.scrollIntoView({ behavior: "smooth" }));
    } finally {
      setSending(false);
    }
  };

  const onLoadOlder = useCallback(async () => {
    prevHeightRef.current = scrollRef.current?.scrollHeight ?? 0;
    await loadOlder();
  }, [loadOlder]);

  // После подгрузки старых — сохраняем визуальную позицию (без прыжка).
  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (el && prevHeightRef.current) {
      el.scrollTop += el.scrollHeight - prevHeightRef.current;
      prevHeightRef.current = 0;
    }
  }, [messages]);

  const onDelete = useCallback((id: number) => {
    api.delete(`/api/admin/chat/messages/${id}`).catch(() => { /* no-op */ });
  }, []);

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Сообщения. Внутренний spacer flex-1 прижимает переписку к низу. */}
      <div className="relative flex-1 min-h-0">
        <div ref={scrollRef} onScroll={onScroll} className="h-full overflow-y-auto overscroll-contain">
          <div className="flex min-h-full flex-col px-3 py-3">
            <div className="flex-1" />
            {hasMore && (
              <div className="text-center pb-2">
                <button onClick={onLoadOlder} className="rounded-full border border-zinc-700 px-3 py-1 text-[11px] text-zinc-400 hover:text-zinc-200">
                  Показать ранние
                </button>
              </div>
            )}
            {rows.length === 0 ? (
              <div className="py-12 text-center text-sm text-zinc-600">Пока нет сообщений — начните общение</div>
            ) : (
              rows.map((row) => (
                <MessageRow key={row.m.id} {...row} isAdmin={isAdmin} onDelete={onDelete}
                  showReceipts={showReceipts} otherReads={otherReads} />
              ))
            )}
            <div ref={bottomRef} />
          </div>
        </div>
        {showJump && (
          <button
            onClick={jumpToBottom}
            aria-label="Вниз к последним сообщениям"
            className="absolute bottom-3 right-3 flex h-9 w-9 items-center justify-center rounded-full border border-zinc-700 bg-zinc-800/95 text-zinc-200 shadow-lg backdrop-blur-sm hover:bg-zinc-700 transition-colors"
          >
            <ChevronDown size={18} />
          </button>
        )}
      </div>

      {/* Ввод */}
      {archived ? (
        <div className="border-t border-zinc-800 px-4 py-3 text-center text-xs text-zinc-500 flex-shrink-0">
          {archivedNote}
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
              placeholder={placeholder}
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
