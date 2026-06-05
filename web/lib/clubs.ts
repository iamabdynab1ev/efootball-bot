export interface ClubInfo {
  id: string;
  name: string;
  color: string;
  logo: string;       // флаг emoji для нац. сборных
  logoUrl?: string;   // URL логотипа для клубов
  isNational: boolean;
}

// Логотипы клубов: football-data.org (бесплатный CDN)
const FD = "https://crests.football-data.org";

const CLUB_DATA: ClubInfo[] = [
  { id: "real_madrid", name: "Real Madrid",        color: "#1a3c8c", logo: "⚽", logoUrl: `${FD}/86.svg`,  isNational: false },
  { id: "barcelona",   name: "Barcelona",           color: "#a50044", logo: "⚽", logoUrl: `${FD}/81.svg`,  isNational: false },
  { id: "man_city",    name: "Man City",            color: "#6cabdd", logo: "⚽", logoUrl: `${FD}/65.svg`,  isNational: false },
  { id: "man_utd",     name: "Man United",          color: "#da291c", logo: "⚽", logoUrl: `${FD}/66.svg`,  isNational: false },
  { id: "liverpool",   name: "Liverpool",           color: "#c8102e", logo: "⚽", logoUrl: `${FD}/64.svg`,  isNational: false },
  { id: "chelsea",     name: "Chelsea",             color: "#034694", logo: "⚽", logoUrl: `${FD}/61.svg`,  isNational: false },
  { id: "arsenal",     name: "Arsenal",             color: "#ef0107", logo: "⚽", logoUrl: `${FD}/57.svg`,  isNational: false },
  { id: "psg",         name: "PSG",                 color: "#003170", logo: "⚽", logoUrl: `${FD}/524.svg`, isNational: false },
  { id: "bayern",      name: "Bayern",              color: "#dc052d", logo: "⚽", logoUrl: `${FD}/5.svg`,   isNational: false },
  { id: "dortmund",    name: "Dortmund",            color: "#fde100", logo: "⚽", logoUrl: `${FD}/4.svg`,   isNational: false },
  { id: "juventus",    name: "Juventus",            color: "#111111", logo: "⚽", logoUrl: `${FD}/109.svg`, isNational: false },
  { id: "inter",       name: "Inter",               color: "#003399", logo: "⚽", logoUrl: `${FD}/108.svg`, isNational: false },
  { id: "ac_milan",    name: "AC Milan",            color: "#fb090b", logo: "⚽", logoUrl: `${FD}/98.svg`,  isNational: false },
  { id: "atletico",    name: "Atlético",            color: "#ce3524", logo: "⚽", logoUrl: `${FD}/78.svg`,  isNational: false },
  { id: "chelsea_w",   name: "Ajax",                color: "#d2122e", logo: "⚽", logoUrl: `${FD}/678.svg`, isNational: false },
  // Национальные сборные — флаги emoji
  { id: "brazil",      name: "Brazil",       color: "#009c3b", logo: "🇧🇷", isNational: true },
  { id: "argentina",   name: "Argentina",    color: "#74acdf", logo: "🇦🇷", isNational: true },
  { id: "france",      name: "France",       color: "#002395", logo: "🇫🇷", isNational: true },
  { id: "england",     name: "England",      color: "#cf081f", logo: "🏴󠁧󠁢󠁥󠁮󠁧󠁿", isNational: true },
  { id: "germany",     name: "Germany",      color: "#111111", logo: "🇩🇪", isNational: true },
  { id: "spain",       name: "Spain",        color: "#c60b1e", logo: "🇪🇸", isNational: true },
  { id: "portugal",    name: "Portugal",     color: "#006600", logo: "🇵🇹", isNational: true },
  { id: "italy",       name: "Italy",        color: "#003399", logo: "🇮🇹", isNational: true },
  { id: "uzbekistan",  name: "Uzbekistan",   color: "#1eb53a", logo: "🇺🇿", isNational: true },
  { id: "tajikistan",  name: "Tajikistan",   color: "#cc0000", logo: "🇹🇯", isNational: true },
];

const CLUB_MAP = new Map<string, ClubInfo>(CLUB_DATA.map((c) => [c.id, c]));

export function getClub(id?: string | null): ClubInfo | undefined {
  if (!id) return undefined;
  return CLUB_MAP.get(id);
}

export default CLUB_DATA;
