package models

import "time"

// Friendly — товарищеский матч: вызов без турнира. Счёт вносит любой из
// участников, подтверждает второй; подтверждённый результат влияет на ELO.
type Friendly struct {
	ID              int64      `json:"id"`
	ChallengerID    int64      `json:"challenger_id"`
	OpponentID      int64      `json:"opponent_id"`
	Status          string     `json:"status"` // pending|accepted|score_claimed|confirmed|declined|cancelled
	ChallengerGoals *int16     `json:"challenger_goals,omitempty"`
	OpponentGoals   *int16     `json:"opponent_goals,omitempty"`
	ClaimedBy       *int64     `json:"claimed_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// Джойны для отображения.
	ChallengerName string `json:"challenger_name"`
	ChallengerClub string `json:"challenger_club,omitempty"`
	OpponentName   string `json:"opponent_name"`
	OpponentClub   string `json:"opponent_club,omitempty"`
}
