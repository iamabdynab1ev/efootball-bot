"use client";

import { useState } from "react";
import { getClub } from "@/lib/clubs";

interface Props {
  displayName?: string;
  favoriteClub?: string;
  size?: number;
  bgClassName?: string;
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

export function PlayerAvatar({ displayName, favoriteClub, size = 32 }: Props) {
  const club = getClub(favoriteClub);
  const [imgError, setImgError] = useState(false);
  const fontSize = Math.max(9, Math.floor(size * 0.4));

  // 1. Национальная сборная — флаг emoji
  if (club?.isNational) {
    return (
      <span
        className="flex-shrink-0 select-none leading-none"
        style={{ fontSize: Math.round(size * 0.82) }}
        title={club.name}
      >
        {club.logo}
      </span>
    );
  }

  // 2. Клуб с крест-логотипом — изображение (битое → фолбэк на бейдж)
  if (club?.logoUrl && !imgError) {
    return (
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
      />
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
    return (
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
      </div>
    );
  }

  // 4. Клуб не выбран — инициалы игрока в нейтральном кружке
  const initials = displayName
    ? displayName.split(" ").map((w) => w[0]).join("").slice(0, 2).toUpperCase()
    : "?";

  return (
    <div
      className="flex flex-shrink-0 items-center justify-center rounded-full bg-zinc-700 text-zinc-300 font-bold select-none"
      style={{ width: size, height: size, fontSize }}
    >
      {initials}
    </div>
  );
}
