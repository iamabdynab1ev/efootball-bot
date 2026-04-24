package service

import (
	"context"
	"math"

	"efootball-bot/internal/repository"
)

type EloService struct {
	userRepo repository.UserRepository
}

func NewEloService(ur repository.UserRepository) *EloService {
	return &EloService{userRepo: ur}
}

// Calculate — ELO очкосини ҳисоблайди
// homeGoals, awayGoals — матч натижаси
// homeTP, awayTP — Team Power
func (s *EloService) Calculate(
	ctx context.Context,
	homeUserID, awayUserID int64,
	homeRating, awayRating int,
	homeTP, awayTP int,
	homeGoals, awayGoals int16,
) (homeNew, awayNew int) {

	// 1. Кутилган натижа (стандарт ELO формула)
	expectedHome := 1.0 / (1.0 + math.Pow(10, float64(awayRating-homeRating)/400))
	expectedAway := 1.0 - expectedHome

	// 2. Реал натижа
	var actualHome, actualAway float64
	if homeGoals > awayGoals {
		actualHome, actualAway = 1.0, 0.0
	} else if homeGoals < awayGoals {
		actualHome, actualAway = 0.0, 1.0
	} else {
		actualHome, actualAway = 0.5, 0.5
	}

	// 3. K коэффициент
	K := 32.0

	// 4. Базавий ўзгариш
	homeDelta := K * (actualHome - expectedHome)
	awayDelta := K * (actualAway - expectedAway)

	// 5. Team Power бонуси (максимум ±10)
	tpDiff := float64(awayTP-homeTP) / 500.0
	bonus := tpDiff * 5.0
	if bonus > 10 {
		bonus = 10
	}
	if bonus < -10 {
		bonus = -10
	}

	// Ғолибга бонус, мағлубга тескари
	if homeGoals > awayGoals {
		homeDelta += bonus
		awayDelta -= bonus
	} else if homeGoals < awayGoals {
		awayDelta += bonus 
		homeDelta -= bonus
	}

	// 6. Янги рейтинг (минимум 100)
	homeNew = homeRating + int(math.Round(homeDelta))
	awayNew = awayRating + int(math.Round(awayDelta))

	if homeNew < 100 {
		homeNew = 100
	}
	if awayNew < 100 {
		awayNew = 100
	}

	return homeNew, awayNew
}
