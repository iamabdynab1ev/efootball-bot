"use client";

import { useState } from "react";
import { getClub } from "@/lib/clubs";

interface Props {
  displayName?: string;
  favoriteClub?: string;
  size?: number;
  bgClassName?: string;
}

export function PlayerAvatar({ displayName, favoriteClub, size = 32 }: Props) {
  const club = getClub(favoriteClub);
  const [imgError, setImgError] = useState(false);
  const fontSize = Math.max(9, Math.floor(size * 0.4));

  // Национальная сборная — флаг emoji
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

  // Клуб — логотип изображение
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

  // Фолбэк — инициалы в кружке
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
