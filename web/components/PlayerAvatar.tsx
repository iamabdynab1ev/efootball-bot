"use client";

import { useState, type ReactNode } from "react";
import { getClub } from "@/lib/clubs";

interface Props {
  displayName?: string;
  favoriteClub?: string;
  size?: number;
  bgClassName?: string;
  /** Онлайн-индикатор: зелёная точка при true. undefined — индикатор не показывается. */
  online?: boolean;
}

// withDot оборачивает аватар, накладывая онлайн-точку в правом нижнем углу.
// Когда online не задан — возвращает аватар как есть (ноль накладных расходов
// для мест, где статус не нужен).
function withDot(avatar: ReactNode, size: number, online?: boolean): JSX.Element {
  if (!online) return <>{avatar}</>;
  const dot = Math.max(8, Math.round(size * 0.3));
  return (
    <span className="relative inline-flex flex-shrink-0" style={{ width: size, height: size }}>
      {avatar}
      <span
        className="absolute rounded-full bg-emerald-500 ring-2 ring-zinc-900"
        style={{ width: dot, height: dot, right: 0, bottom: 0 }}
        title="онлайн"
      />
    </span>
  );
}

// Светлый ли цвет — чтобы выбрать контрастный цвет текста на бейдже.
function isLight(hex: string) {
  const c = hex.replace("#", "");
  if (c.length < 6) return false;
  const r = parseInt(c.slice(0, 2), 16);
  const g = parseInt(c.slice(2, 4), 16);
  const b = parseInt(c.slice(4, 6), 16);
  return (r * 299 + g * 587 + b * 114) / 1000 > 160;
}

export function PlayerAvatar({ displayName, favoriteClub, size = 32, online }: Props) {
  const club = getClub(favoriteClub);
  const [imgError, setImgError] = useState(false);
  const fontSize = Math.max(9, Math.floor(size * 0.4));

  // 1. Национальная сборная — флаг emoji
  if (club?.isNational) {
    return withDot(
      <span
        className="flex-shrink-0 select-none leading-none"
        style={{ fontSize: Math.round(size * 0.82) }}
        title={club.name}
      >
        {club.logo}
      </span>,
      size, online,
    );
  }

  // 2. Клуб с крест-логотипом — изображение (битое → фолбэк на бейдж)
  if (club?.logoUrl && !imgError) {
    return withDot(
      <img
        src={club.logoUrl}
        alt={club.name}
        title={club.name}
        width={size}
        height={size}
        loading="lazy"
        decoding="async"
        fetchPriority="low"
        className="flex-shrink-0 object-contain"
        style={{ width: size, height: size }}
        onError={() => setImgError(true)}
      />,
      size, online,
    );
  }

  // 3. Клуб без креста — фирменный градиентный бейдж с инициалами клуба
  if (club) {
    const initials = club.name
      .split(" ")
      .map((w) => w[0])
      .join("")
      .slice(0, 2)
      .toUpperCase();
    return withDot(
      <div
        className="flex flex-shrink-0 items-center justify-center rounded-full font-black select-none ring-1 ring-black/10"
        title={club.name}
        style={{
          width: size,
          height: size,
          fontSize,
          background: `linear-gradient(135deg, ${club.color} 0%, ${club.color2} 100%)`,
          color: isLight(club.color) ? "#000" : "#fff",
        }}
      >
        {initials}
      </div>,
      size, online,
    );
  }

  // 4. Клуб не выбран — инициалы игрока в нейтральном кружке
  const initials = displayName
    ? displayName.split(" ").map((w) => w[0]).join("").slice(0, 2).toUpperCase()
    : "?";

  return withDot(
    <div
      className="flex flex-shrink-0 items-center justify-center rounded-full bg-zinc-700 text-zinc-300 font-bold select-none"
      style={{ width: size, height: size, fontSize }}
    >
      {initials}
    </div>,
    size, online,
  );
}
