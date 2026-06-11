package models

import "time"

type Achievement struct {
	ID     int    `db:"id"`
	Code   string `db:"code"`
	Icon   string `db:"icon"`
	NameUz string `db:"name_uz"`
	NameRu string `db:"name_ru"`
	NameTg string `db:"name_tg"`
}

type UserAchievement struct {
	ID            int64        `db:"id"`
	UserID        int64        `db:"user_id"`
	AchievementID int          `db:"achievement_id"`
	LeagueID      *int64       `db:"league_id"`
	EarnedAt      time.Time    `db:"earned_at"`
	Achievement   *Achievement // joined
}

type RoundDeadline struct {
	ID              int64     `db:"id"`
	LeagueID        int64     `db:"league_id"`
	Round           int16     `db:"round"`
	Deadline        time.Time `db:"deadline"`
	Reminder24hSent bool      `db:"reminder_24h_sent"`
	Reminder1hSent  bool      `db:"reminder_1h_sent"`
	CreatedAt       time.Time `db:"created_at"`
}

type SeasonAward struct {
	ID        int64     `db:"id"`
	SeasonID  int64     `db:"season_id"`
	LeagueID  *int64    `db:"league_id"`
	AwardType string    `db:"award_type"`
	UserID    int64     `db:"user_id"`
	Value     *int      `db:"value"`
	CreatedAt time.Time `db:"created_at"`
	// Joined fields
	UserDisplayName string
	LeagueName      string
	SeasonName      string
}

type GlobalStats struct {
	TotalMatches      int `db:"total_matches"`
	TotalWins         int `db:"total_wins"`
	TotalDraws        int `db:"total_draws"`
	TotalLosses       int `db:"total_losses"`
	TotalGoalsFor     int `db:"total_goals_for"`
	TotalGoalsAgainst int `db:"total_goals_against"`
}
