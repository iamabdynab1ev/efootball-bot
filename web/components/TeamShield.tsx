const COLORS = [
  "#e53935", "#1e88e5", "#43a047", "#f9a825",
  "#8e24aa", "#fb8c00", "#546e7a", "#00acc1",
  "#7cb342", "#3949ab", "#e91e63", "#00897b",
];

function colorFor(name: string): string {
  let h = 0;
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) >>> 0;
  return COLORS[h % COLORS.length];
}

export function TeamShield({ name, size = 40 }: { name?: string; size?: number }) {
  const label = name || "?";
  const color = colorFor(label);
  const letter = label.slice(0, 1).toUpperCase();
  const s = size;

  return (
    <svg
      width={s}
      height={Math.round(s * 1.1)}
      viewBox="0 0 40 44"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      style={{ flexShrink: 0 }}
    >
      <path
        d="M20 1L2 9V22C2 33 10.5 41.5 20 43C29.5 41.5 38 33 38 22V9L20 1Z"
        fill={color}
        stroke="rgba(255,255,255,0.18)"
        strokeWidth="1.5"
      />
      <path
        d="M20 5.5L6 12V22C6 31 13 38.5 20 40C27 38.5 34 31 34 22V12L20 5.5Z"
        fill="none"
        stroke="rgba(255,255,255,0.12)"
        strokeWidth="1"
      />
      <text
        x="20"
        y="27"
        textAnchor="middle"
        fill="white"
        fontSize="15"
        fontWeight="900"
        fontFamily="Inter, Arial, sans-serif"
      >
        {letter}
      </text>
    </svg>
  );
}
