"use client";

interface Props {
  displayName?: string;
  favoriteClub?: string;
  size?: number;
  bgClassName?: string;
}

export function PlayerAvatar({ displayName, size = 32, bgClassName = "bg-zinc-700" }: Props) {
  const initials = displayName
    ? displayName
        .split(" ")
        .map((w) => w[0])
        .join("")
        .slice(0, 2)
        .toUpperCase()
    : "?";
  const fontSize = Math.max(9, Math.floor(size * 0.4));
  return (
    <div
      className={`flex flex-shrink-0 items-center justify-center rounded-full ${bgClassName} text-zinc-300 font-bold select-none`}
      style={{ width: size, height: size, fontSize }}
    >
      {initials}
    </div>
  );
}
