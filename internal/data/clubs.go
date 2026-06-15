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
