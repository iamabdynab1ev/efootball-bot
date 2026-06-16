package data

// Club represents an eFootball club.
type Club struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	NameRu  string `json:"name_ru"`
	Type    string `json:"type"` // "club" | "national"
	Country string `json:"country"`
	Region  string `json:"region"`
	Color   string `json:"color"`
	Color2  string `json:"color2"`
	Logo    string `json:"logo"`
}

// Clubs is the static list of available clubs.
var Clubs = []Club{
	{ID: "real_madrid", Name: "Real Madrid", NameRu: "Реал Мадрид", Type: "club", Country: "Spain", Region: "Europe", Color: "#ffffff", Color2: "#1a3c8c", Logo: "⚽"},
	{ID: "barcelona", Name: "Barcelona", NameRu: "Барселона", Type: "club", Country: "Spain", Region: "Europe", Color: "#a50044", Color2: "#004d98", Logo: "⚽"},
	{ID: "man_city", Name: "Manchester City", NameRu: "Манчестер Сити", Type: "club", Country: "England", Region: "Europe", Color: "#6cabdd", Color2: "#ffffff", Logo: "⚽"},
	{ID: "man_utd", Name: "Manchester United", NameRu: "Манчестер Юнайтед", Type: "club", Country: "England", Region: "Europe", Color: "#da291c", Color2: "#fbe122", Logo: "⚽"},
	{ID: "liverpool", Name: "Liverpool", NameRu: "Ливерпуль", Type: "club", Country: "England", Region: "Europe", Color: "#c8102e", Color2: "#f6eb61", Logo: "⚽"},
	{ID: "chelsea", Name: "Chelsea", NameRu: "Челси", Type: "club", Country: "England", Region: "Europe", Color: "#034694", Color2: "#ffffff", Logo: "⚽"},
	{ID: "arsenal", Name: "Arsenal", NameRu: "Арсенал", Type: "club", Country: "England", Region: "Europe", Color: "#ef0107", Color2: "#ffffff", Logo: "⚽"},
	{ID: "psg", Name: "Paris Saint-Germain", NameRu: "ПСЖ", Type: "club", Country: "France", Region: "Europe", Color: "#003170", Color2: "#da291c", Logo: "⚽"},
	{ID: "bayern", Name: "Bayern München", NameRu: "Бавария", Type: "club", Country: "Germany", Region: "Europe", Color: "#dc052d", Color2: "#0066b2", Logo: "⚽"},
	{ID: "dortmund", Name: "Borussia Dortmund", NameRu: "Боруссия Дортмунд", Type: "club", Country: "Germany", Region: "Europe", Color: "#fde100", Color2: "#000000", Logo: "⚽"},
	{ID: "juventus", Name: "Juventus", NameRu: "Ювентус", Type: "club", Country: "Italy", Region: "Europe", Color: "#000000", Color2: "#ffffff", Logo: "⚽"},
	{ID: "inter", Name: "Inter Milan", NameRu: "Интер", Type: "club", Country: "Italy", Region: "Europe", Color: "#003399", Color2: "#000000", Logo: "⚽"},
	{ID: "ac_milan", Name: "AC Milan", NameRu: "Милан", Type: "club", Country: "Italy", Region: "Europe", Color: "#fb090b", Color2: "#000000", Logo: "⚽"},
	{ID: "atletico", Name: "Atlético Madrid", NameRu: "Атлетико", Type: "club", Country: "Spain", Region: "Europe", Color: "#ce3524", Color2: "#272e6b", Logo: "⚽"},
	{ID: "chelsea_w", Name: "Ajax", NameRu: "Аякс", Type: "club", Country: "Netherlands", Region: "Europe", Color: "#d2122e", Color2: "#ffffff", Logo: "⚽"},
	{ID: "brazil", Name: "Brazil", NameRu: "Бразилия", Type: "national", Country: "Brazil", Region: "South America", Color: "#009c3b", Color2: "#fedf00", Logo: "🇧🇷"},
	{ID: "argentina", Name: "Argentina", NameRu: "Аргентина", Type: "national", Country: "Argentina", Region: "South America", Color: "#74acdf", Color2: "#ffffff", Logo: "🇦🇷"},
	{ID: "france", Name: "France", NameRu: "Франция", Type: "national", Country: "France", Region: "Europe", Color: "#002395", Color2: "#ffffff", Logo: "🇫🇷"},
	{ID: "england", Name: "England", NameRu: "Англия", Type: "national", Country: "England", Region: "Europe", Color: "#ffffff", Color2: "#cf081f", Logo: "🏴󠁧󠁢󠁥󠁮󠁧󠁿"},
	{ID: "germany", Name: "Germany", NameRu: "Германия", Type: "national", Country: "Germany", Region: "Europe", Color: "#ffffff", Color2: "#000000", Logo: "🇩🇪"},
	{ID: "spain", Name: "Spain", NameRu: "Испания", Type: "national", Country: "Spain", Region: "Europe", Color: "#c60b1e", Color2: "#ffc400", Logo: "🇪🇸"},
	{ID: "portugal", Name: "Portugal", NameRu: "Португалия", Type: "national", Country: "Portugal", Region: "Europe", Color: "#006600", Color2: "#ff0000", Logo: "🇵🇹"},
	{ID: "italy", Name: "Italy", NameRu: "Италия", Type: "national", Country: "Italy", Region: "Europe", Color: "#003399", Color2: "#ffffff", Logo: "🇮🇹"},
	{ID: "uzbekistan", Name: "Uzbekistan", NameRu: "Узбекистан", Type: "national", Country: "Uzbekistan", Region: "Asia", Color: "#1eb53a", Color2: "#0099b5", Logo: "🇺🇿"},
	{ID: "tajikistan", Name: "Tajikistan", NameRu: "Таджикистан", Type: "national", Country: "Tajikistan", Region: "Asia", Color: "#cc0000", Color2: "#006600", Logo: "🇹🇯"},

	// ── England (Premier League) ──
	{ID: "tottenham", Name: "Tottenham Hotspur", NameRu: "Тоттенхэм", Type: "club", Country: "England", Region: "Europe", Color: "#ffffff", Color2: "#132257", Logo: "⚽"},
	{ID: "newcastle", Name: "Newcastle United", NameRu: "Ньюкасл", Type: "club", Country: "England", Region: "Europe", Color: "#241f20", Color2: "#ffffff", Logo: "⚽"},
	{ID: "aston_villa", Name: "Aston Villa", NameRu: "Астон Вилла", Type: "club", Country: "England", Region: "Europe", Color: "#670e36", Color2: "#95bfe5", Logo: "⚽"},
	{ID: "west_ham", Name: "West Ham United", NameRu: "Вест Хэм", Type: "club", Country: "England", Region: "Europe", Color: "#7a263a", Color2: "#1bb1e7", Logo: "⚽"},
	{ID: "brighton", Name: "Brighton & Hove Albion", NameRu: "Брайтон", Type: "club", Country: "England", Region: "Europe", Color: "#0057b8", Color2: "#ffffff", Logo: "⚽"},
	{ID: "wolves", Name: "Wolverhampton", NameRu: "Вулверхэмптон", Type: "club", Country: "England", Region: "Europe", Color: "#fdb913", Color2: "#231f20", Logo: "⚽"},
	{ID: "crystal_palace", Name: "Crystal Palace", NameRu: "Кристал Пэлас", Type: "club", Country: "England", Region: "Europe", Color: "#1b458f", Color2: "#c4122e", Logo: "⚽"},
	{ID: "fulham", Name: "Fulham", NameRu: "Фулхэм", Type: "club", Country: "England", Region: "Europe", Color: "#ffffff", Color2: "#000000", Logo: "⚽"},
	{ID: "brentford", Name: "Brentford", NameRu: "Брентфорд", Type: "club", Country: "England", Region: "Europe", Color: "#e30613", Color2: "#ffffff", Logo: "⚽"},
	{ID: "everton", Name: "Everton", NameRu: "Эвертон", Type: "club", Country: "England", Region: "Europe", Color: "#003399", Color2: "#ffffff", Logo: "⚽"},
	{ID: "nottingham", Name: "Nottingham Forest", NameRu: "Ноттингем Форест", Type: "club", Country: "England", Region: "Europe", Color: "#dd0000", Color2: "#ffffff", Logo: "⚽"},
	{ID: "leeds", Name: "Leeds United", NameRu: "Лидс", Type: "club", Country: "England", Region: "Europe", Color: "#ffffff", Color2: "#1d428a", Logo: "⚽"},

	// ── Spain (La Liga) ──
	{ID: "sevilla", Name: "Sevilla", NameRu: "Севилья", Type: "club", Country: "Spain", Region: "Europe", Color: "#d8112a", Color2: "#ffffff", Logo: "⚽"},
	{ID: "real_betis", Name: "Real Betis", NameRu: "Бетис", Type: "club", Country: "Spain", Region: "Europe", Color: "#00954c", Color2: "#ffffff", Logo: "⚽"},
	{ID: "villarreal", Name: "Villarreal", NameRu: "Вильярреал", Type: "club", Country: "Spain", Region: "Europe", Color: "#ffe667", Color2: "#005187", Logo: "⚽"},
	{ID: "real_sociedad", Name: "Real Sociedad", NameRu: "Реал Сосьедад", Type: "club", Country: "Spain", Region: "Europe", Color: "#0067b1", Color2: "#ffffff", Logo: "⚽"},
	{ID: "athletic", Name: "Athletic Bilbao", NameRu: "Атлетик Бильбао", Type: "club", Country: "Spain", Region: "Europe", Color: "#ee2523", Color2: "#ffffff", Logo: "⚽"},
	{ID: "valencia", Name: "Valencia", NameRu: "Валенсия", Type: "club", Country: "Spain", Region: "Europe", Color: "#ffffff", Color2: "#f18e00", Logo: "⚽"},
	{ID: "girona", Name: "Girona", NameRu: "Жирона", Type: "club", Country: "Spain", Region: "Europe", Color: "#d50032", Color2: "#ffffff", Logo: "⚽"},
	{ID: "celta", Name: "Celta Vigo", NameRu: "Сельта", Type: "club", Country: "Spain", Region: "Europe", Color: "#8ac3ee", Color2: "#ffffff", Logo: "⚽"},

	// ── Italy (Serie A) ──
	{ID: "napoli", Name: "Napoli", NameRu: "Наполи", Type: "club", Country: "Italy", Region: "Europe", Color: "#12a0d7", Color2: "#ffffff", Logo: "⚽"},
	{ID: "roma", Name: "AS Roma", NameRu: "Рома", Type: "club", Country: "Italy", Region: "Europe", Color: "#8e1f2f", Color2: "#f0bc42", Logo: "⚽"},
	{ID: "lazio", Name: "Lazio", NameRu: "Лацио", Type: "club", Country: "Italy", Region: "Europe", Color: "#87d8f7", Color2: "#ffffff", Logo: "⚽"},
	{ID: "atalanta", Name: "Atalanta", NameRu: "Аталанта", Type: "club", Country: "Italy", Region: "Europe", Color: "#1d1d1b", Color2: "#005ca9", Logo: "⚽"},
	{ID: "fiorentina", Name: "Fiorentina", NameRu: "Фиорентина", Type: "club", Country: "Italy", Region: "Europe", Color: "#592c82", Color2: "#ffffff", Logo: "⚽"},
	{ID: "bologna", Name: "Bologna", NameRu: "Болонья", Type: "club", Country: "Italy", Region: "Europe", Color: "#1a2f48", Color2: "#a01e20", Logo: "⚽"},
	{ID: "torino", Name: "Torino", NameRu: "Торино", Type: "club", Country: "Italy", Region: "Europe", Color: "#881600", Color2: "#ffffff", Logo: "⚽"},

	// ── Germany (Bundesliga) ──
	{ID: "leipzig", Name: "RB Leipzig", NameRu: "Лейпциг", Type: "club", Country: "Germany", Region: "Europe", Color: "#dd0741", Color2: "#001f47", Logo: "⚽"},
	{ID: "leverkusen", Name: "Bayer Leverkusen", NameRu: "Байер", Type: "club", Country: "Germany", Region: "Europe", Color: "#e32219", Color2: "#000000", Logo: "⚽"},
	{ID: "frankfurt", Name: "Eintracht Frankfurt", NameRu: "Айнтрахт", Type: "club", Country: "Germany", Region: "Europe", Color: "#000000", Color2: "#e1000f", Logo: "⚽"},
	{ID: "wolfsburg", Name: "VfL Wolfsburg", NameRu: "Вольфсбург", Type: "club", Country: "Germany", Region: "Europe", Color: "#65b32e", Color2: "#ffffff", Logo: "⚽"},
	{ID: "stuttgart", Name: "VfB Stuttgart", NameRu: "Штутгарт", Type: "club", Country: "Germany", Region: "Europe", Color: "#ffffff", Color2: "#e30613", Logo: "⚽"},
	{ID: "gladbach", Name: "Borussia M'gladbach", NameRu: "Боруссия М", Type: "club", Country: "Germany", Region: "Europe", Color: "#ffffff", Color2: "#000000", Logo: "⚽"},
	{ID: "freiburg", Name: "SC Freiburg", NameRu: "Фрайбург", Type: "club", Country: "Germany", Region: "Europe", Color: "#000000", Color2: "#e2001a", Logo: "⚽"},

	// ── France (Ligue 1) ──
	{ID: "marseille", Name: "Olympique Marseille", NameRu: "Марсель", Type: "club", Country: "France", Region: "Europe", Color: "#2faee0", Color2: "#ffffff", Logo: "⚽"},
	{ID: "lyon", Name: "Olympique Lyon", NameRu: "Лион", Type: "club", Country: "France", Region: "Europe", Color: "#ffffff", Color2: "#e2001a", Logo: "⚽"},
	{ID: "monaco", Name: "AS Monaco", NameRu: "Монако", Type: "club", Country: "France", Region: "Europe", Color: "#e51b22", Color2: "#ffffff", Logo: "⚽"},
	{ID: "lille", Name: "Lille", NameRu: "Лилль", Type: "club", Country: "France", Region: "Europe", Color: "#e01e24", Color2: "#001b50", Logo: "⚽"},
	{ID: "nice", Name: "OGC Nice", NameRu: "Ницца", Type: "club", Country: "France", Region: "Europe", Color: "#c8102e", Color2: "#000000", Logo: "⚽"},
	{ID: "rennes", Name: "Stade Rennais", NameRu: "Ренн", Type: "club", Country: "France", Region: "Europe", Color: "#e23737", Color2: "#000000", Logo: "⚽"},
	{ID: "lens", Name: "RC Lens", NameRu: "Ланс", Type: "club", Country: "France", Region: "Europe", Color: "#ffe500", Color2: "#e2001a", Logo: "⚽"},

	// ── Portugal / Netherlands / others (Europe) ──
	{ID: "benfica", Name: "Benfica", NameRu: "Бенфика", Type: "club", Country: "Portugal", Region: "Europe", Color: "#e30613", Color2: "#ffffff", Logo: "⚽"},
	{ID: "porto", Name: "FC Porto", NameRu: "Порту", Type: "club", Country: "Portugal", Region: "Europe", Color: "#00428c", Color2: "#ffffff", Logo: "⚽"},
	{ID: "sporting", Name: "Sporting CP", NameRu: "Спортинг", Type: "club", Country: "Portugal", Region: "Europe", Color: "#008057", Color2: "#ffffff", Logo: "⚽"},
	{ID: "braga", Name: "SC Braga", NameRu: "Брага", Type: "club", Country: "Portugal", Region: "Europe", Color: "#e30613", Color2: "#ffffff", Logo: "⚽"},
	{ID: "ajax", Name: "Ajax", NameRu: "Аякс", Type: "club", Country: "Netherlands", Region: "Europe", Color: "#d2122e", Color2: "#ffffff", Logo: "⚽"},
	{ID: "psv", Name: "PSV Eindhoven", NameRu: "ПСВ", Type: "club", Country: "Netherlands", Region: "Europe", Color: "#ed1c24", Color2: "#ffffff", Logo: "⚽"},
	{ID: "feyenoord", Name: "Feyenoord", NameRu: "Фейеноорд", Type: "club", Country: "Netherlands", Region: "Europe", Color: "#e30613", Color2: "#ffffff", Logo: "⚽"},
	{ID: "celtic", Name: "Celtic", NameRu: "Селтик", Type: "club", Country: "Scotland", Region: "Europe", Color: "#16973b", Color2: "#ffffff", Logo: "⚽"},
	{ID: "rangers", Name: "Rangers", NameRu: "Рейнджерс", Type: "club", Country: "Scotland", Region: "Europe", Color: "#1b458f", Color2: "#ffffff", Logo: "⚽"},
	{ID: "galatasaray", Name: "Galatasaray", NameRu: "Галатасарай", Type: "club", Country: "Turkey", Region: "Europe", Color: "#a90432", Color2: "#fbb800", Logo: "⚽"},
	{ID: "fenerbahce", Name: "Fenerbahçe", NameRu: "Фенербахче", Type: "club", Country: "Turkey", Region: "Europe", Color: "#093a7c", Color2: "#ffed00", Logo: "⚽"},
	{ID: "besiktas", Name: "Beşiktaş", NameRu: "Бешикташ", Type: "club", Country: "Turkey", Region: "Europe", Color: "#000000", Color2: "#ffffff", Logo: "⚽"},
	{ID: "shakhtar", Name: "Shakhtar Donetsk", NameRu: "Шахтёр", Type: "club", Country: "Ukraine", Region: "Europe", Color: "#f47b20", Color2: "#000000", Logo: "⚽"},
	{ID: "club_brugge", Name: "Club Brugge", NameRu: "Брюгге", Type: "club", Country: "Belgium", Region: "Europe", Color: "#005baa", Color2: "#000000", Logo: "⚽"},
	{ID: "salzburg", Name: "RB Salzburg", NameRu: "Зальцбург", Type: "club", Country: "Austria", Region: "Europe", Color: "#e2001a", Color2: "#ffffff", Logo: "⚽"},

	// ── South America (clubs) ──
	{ID: "boca", Name: "Boca Juniors", NameRu: "Бока Хуниорс", Type: "club", Country: "Argentina", Region: "South America", Color: "#003f88", Color2: "#fcb813", Logo: "⚽"},
	{ID: "river_plate", Name: "River Plate", NameRu: "Ривер Плейт", Type: "club", Country: "Argentina", Region: "South America", Color: "#ffffff", Color2: "#e6002d", Logo: "⚽"},
	{ID: "flamengo", Name: "Flamengo", NameRu: "Фламенго", Type: "club", Country: "Brazil", Region: "South America", Color: "#c52613", Color2: "#000000", Logo: "⚽"},
	{ID: "palmeiras", Name: "Palmeiras", NameRu: "Палмейрас", Type: "club", Country: "Brazil", Region: "South America", Color: "#006437", Color2: "#ffffff", Logo: "⚽"},
	{ID: "corinthians", Name: "Corinthians", NameRu: "Коринтианс", Type: "club", Country: "Brazil", Region: "South America", Color: "#000000", Color2: "#ffffff", Logo: "⚽"},
	{ID: "sao_paulo", Name: "São Paulo", NameRu: "Сан-Паулу", Type: "club", Country: "Brazil", Region: "South America", Color: "#ffffff", Color2: "#e30613", Logo: "⚽"},
	{ID: "santos", Name: "Santos", NameRu: "Сантос", Type: "club", Country: "Brazil", Region: "South America", Color: "#ffffff", Color2: "#000000", Logo: "⚽"},
	{ID: "gremio", Name: "Grêmio", NameRu: "Гремио", Type: "club", Country: "Brazil", Region: "South America", Color: "#0d80bf", Color2: "#000000", Logo: "⚽"},

	// ── National teams (Europe) ──
	{ID: "netherlands", Name: "Netherlands", NameRu: "Нидерланды", Type: "national", Country: "Netherlands", Region: "Europe", Color: "#ff7d00", Color2: "#ffffff", Logo: "🇳🇱"},
	{ID: "belgium", Name: "Belgium", NameRu: "Бельгия", Type: "national", Country: "Belgium", Region: "Europe", Color: "#e30613", Color2: "#ffe500", Logo: "🇧🇪"},
	{ID: "croatia", Name: "Croatia", NameRu: "Хорватия", Type: "national", Country: "Croatia", Region: "Europe", Color: "#e30613", Color2: "#ffffff", Logo: "🇭🇷"},
	{ID: "denmark", Name: "Denmark", NameRu: "Дания", Type: "national", Country: "Denmark", Region: "Europe", Color: "#c8102e", Color2: "#ffffff", Logo: "🇩🇰"},
	{ID: "switzerland", Name: "Switzerland", NameRu: "Швейцария", Type: "national", Country: "Switzerland", Region: "Europe", Color: "#d52b1e", Color2: "#ffffff", Logo: "🇨🇭"},
	{ID: "poland", Name: "Poland", NameRu: "Польша", Type: "national", Country: "Poland", Region: "Europe", Color: "#ffffff", Color2: "#dc143c", Logo: "🇵🇱"},
	{ID: "sweden", Name: "Sweden", NameRu: "Швеция", Type: "national", Country: "Sweden", Region: "Europe", Color: "#006aa7", Color2: "#fecc00", Logo: "🇸🇪"},
	{ID: "ukraine", Name: "Ukraine", NameRu: "Украина", Type: "national", Country: "Ukraine", Region: "Europe", Color: "#005bbb", Color2: "#ffd500", Logo: "🇺🇦"},
	{ID: "serbia", Name: "Serbia", NameRu: "Сербия", Type: "national", Country: "Serbia", Region: "Europe", Color: "#c6363c", Color2: "#ffffff", Logo: "🇷🇸"},
	{ID: "turkey", Name: "Turkey", NameRu: "Турция", Type: "national", Country: "Turkey", Region: "Europe", Color: "#e30a17", Color2: "#ffffff", Logo: "🇹🇷"},
	{ID: "wales", Name: "Wales", NameRu: "Уэльс", Type: "national", Country: "Wales", Region: "Europe", Color: "#c8102e", Color2: "#ffffff", Logo: "🏴󠁧󠁢󠁷󠁬󠁳󠁿"},
	{ID: "scotland", Name: "Scotland", NameRu: "Шотландия", Type: "national", Country: "Scotland", Region: "Europe", Color: "#0065bd", Color2: "#ffffff", Logo: "🏴󠁧󠁢󠁳󠁣󠁴󠁿"},
	{ID: "austria_nt", Name: "Austria", NameRu: "Австрия", Type: "national", Country: "Austria", Region: "Europe", Color: "#ef3340", Color2: "#ffffff", Logo: "🇦🇹"},
	{ID: "russia", Name: "Russia", NameRu: "Россия", Type: "national", Country: "Russia", Region: "Europe", Color: "#ffffff", Color2: "#d52b1e", Logo: "🇷🇺"},

	// ── National teams (South America) ──
	{ID: "uruguay", Name: "Uruguay", NameRu: "Уругвай", Type: "national", Country: "Uruguay", Region: "South America", Color: "#5cbfeb", Color2: "#ffffff", Logo: "🇺🇾"},
	{ID: "colombia", Name: "Colombia", NameRu: "Колумбия", Type: "national", Country: "Colombia", Region: "South America", Color: "#fcd116", Color2: "#003893", Logo: "🇨🇴"},
	{ID: "chile", Name: "Chile", NameRu: "Чили", Type: "national", Country: "Chile", Region: "South America", Color: "#d52b1e", Color2: "#0039a6", Logo: "🇨🇱"},
	{ID: "peru", Name: "Peru", NameRu: "Перу", Type: "national", Country: "Peru", Region: "South America", Color: "#d91023", Color2: "#ffffff", Logo: "🇵🇪"},
	{ID: "ecuador", Name: "Ecuador", NameRu: "Эквадор", Type: "national", Country: "Ecuador", Region: "South America", Color: "#ffd100", Color2: "#0072ce", Logo: "🇪🇨"},
	{ID: "paraguay", Name: "Paraguay", NameRu: "Парагвай", Type: "national", Country: "Paraguay", Region: "South America", Color: "#d52b1e", Color2: "#0038a8", Logo: "🇵🇾"},

	// ── National teams (Asia) ──
	{ID: "japan", Name: "Japan", NameRu: "Япония", Type: "national", Country: "Japan", Region: "Asia", Color: "#0a1f8f", Color2: "#ffffff", Logo: "🇯🇵"},
	{ID: "south_korea", Name: "South Korea", NameRu: "Южная Корея", Type: "national", Country: "South Korea", Region: "Asia", Color: "#c60c30", Color2: "#003478", Logo: "🇰🇷"},
	{ID: "saudi_arabia", Name: "Saudi Arabia", NameRu: "Саудовская Аравия", Type: "national", Country: "Saudi Arabia", Region: "Asia", Color: "#006c35", Color2: "#ffffff", Logo: "🇸🇦"},
	{ID: "iran", Name: "Iran", NameRu: "Иран", Type: "national", Country: "Iran", Region: "Asia", Color: "#239f40", Color2: "#da0000", Logo: "🇮🇷"},
	{ID: "australia", Name: "Australia", NameRu: "Австралия", Type: "national", Country: "Australia", Region: "Asia", Color: "#ffcd00", Color2: "#00843d", Logo: "🇦🇺"},
	{ID: "qatar", Name: "Qatar", NameRu: "Катар", Type: "national", Country: "Qatar", Region: "Asia", Color: "#8a1538", Color2: "#ffffff", Logo: "🇶🇦"},
	{ID: "iraq", Name: "Iraq", NameRu: "Ирак", Type: "national", Country: "Iraq", Region: "Asia", Color: "#ffffff", Color2: "#007a3d", Logo: "🇮🇶"},
	{ID: "china", Name: "China", NameRu: "Китай", Type: "national", Country: "China", Region: "Asia", Color: "#de2910", Color2: "#ffde00", Logo: "🇨🇳"},
	{ID: "kazakhstan", Name: "Kazakhstan", NameRu: "Казахстан", Type: "national", Country: "Kazakhstan", Region: "Asia", Color: "#00afca", Color2: "#fec50c", Logo: "🇰🇿"},
	{ID: "kyrgyzstan", Name: "Kyrgyzstan", NameRu: "Кыргызстан", Type: "national", Country: "Kyrgyzstan", Region: "Asia", Color: "#e8112d", Color2: "#ffef00", Logo: "🇰🇬"},
	{ID: "turkmenistan", Name: "Turkmenistan", NameRu: "Туркменистан", Type: "national", Country: "Turkmenistan", Region: "Asia", Color: "#00853e", Color2: "#ffffff", Logo: "🇹🇲"},

	// ── National teams (Africa) ──
	{ID: "morocco", Name: "Morocco", NameRu: "Марокко", Type: "national", Country: "Morocco", Region: "Africa", Color: "#c1272d", Color2: "#006233", Logo: "🇲🇦"},
	{ID: "senegal", Name: "Senegal", NameRu: "Сенегал", Type: "national", Country: "Senegal", Region: "Africa", Color: "#00853f", Color2: "#fdef42", Logo: "🇸🇳"},
	{ID: "nigeria", Name: "Nigeria", NameRu: "Нигерия", Type: "national", Country: "Nigeria", Region: "Africa", Color: "#008751", Color2: "#ffffff", Logo: "🇳🇬"},
	{ID: "egypt", Name: "Egypt", NameRu: "Египет", Type: "national", Country: "Egypt", Region: "Africa", Color: "#c8102e", Color2: "#ffffff", Logo: "🇪🇬"},
	{ID: "ghana", Name: "Ghana", NameRu: "Гана", Type: "national", Country: "Ghana", Region: "Africa", Color: "#ce1126", Color2: "#fcd116", Logo: "🇬🇭"},
	{ID: "cameroon", Name: "Cameroon", NameRu: "Камерун", Type: "national", Country: "Cameroon", Region: "Africa", Color: "#007a5e", Color2: "#ce1126", Logo: "🇨🇲"},
	{ID: "algeria", Name: "Algeria", NameRu: "Алжир", Type: "national", Country: "Algeria", Region: "Africa", Color: "#006233", Color2: "#ffffff", Logo: "🇩🇿"},
	{ID: "ivory_coast", Name: "Ivory Coast", NameRu: "Кот-д'Ивуар", Type: "national", Country: "Ivory Coast", Region: "Africa", Color: "#ff8200", Color2: "#009e60", Logo: "🇨🇮"},

	// ── National teams (North America) ──
	{ID: "usa", Name: "USA", NameRu: "США", Type: "national", Country: "USA", Region: "North America", Color: "#0a3161", Color2: "#b31942", Logo: "🇺🇸"},
	{ID: "mexico", Name: "Mexico", NameRu: "Мексика", Type: "national", Country: "Mexico", Region: "North America", Color: "#006847", Color2: "#ce1126", Logo: "🇲🇽"},
	{ID: "canada", Name: "Canada", NameRu: "Канада", Type: "national", Country: "Canada", Region: "North America", Color: "#d52b1e", Color2: "#ffffff", Logo: "🇨🇦"},
	{ID: "costa_rica", Name: "Costa Rica", NameRu: "Коста-Рика", Type: "national", Country: "Costa Rica", Region: "North America", Color: "#002b7f", Color2: "#ce1126", Logo: "🇨🇷"},
}

// clubIDs — множество валидных ID, строится один раз при инициализации пакета.
var clubIDs = func() map[string]struct{} {
	m := make(map[string]struct{}, len(Clubs))
	for _, c := range Clubs {
		m[c.ID] = struct{}{}
	}
	return m
}()

// IsValidClubID сообщает, есть ли клуб с таким ID в справочнике.
// Пустая строка считается валидной — это «клуб не выбран».
func IsValidClubID(id string) bool {
	if id == "" {
		return true
	}
	_, ok := clubIDs[id]
	return ok
}
