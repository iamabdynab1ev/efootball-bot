package service

import (
	"context"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"fmt"
	"math/rand"
	"sort"
)

// TournamentConfig holds auto-calculated parameters for a given league size and format.
type TournamentConfig struct {
	NumGroups     int
	GroupAdvance  int
	BestRunnersUp int
	NumDivisions  int
	NumRounds     int
}

// Calculate derives tournament parameters from the number of members and format string.
func Calculate(n int, roundsType string) TournamentConfig {
	cfg := TournamentConfig{}
	switch roundsType {
	case "groups", "groups_playoff":
		switch {
		case n <= 8:
			cfg.NumGroups, cfg.GroupAdvance, cfg.BestRunnersUp = 2, 2, 0
		case n <= 16:
			cfg.NumGroups, cfg.GroupAdvance, cfg.BestRunnersUp = 4, 2, 0
		default:
			cfg.NumGroups, cfg.GroupAdvance, cfg.BestRunnersUp = 8, 2, 0
		}
	case "swiss":
		cfg.NumRounds = NumRoundsForSwiss(n)
	case "nations_league":
		switch {
		case n <= 8:
			cfg.NumDivisions = 2
		case n <= 16:
			cfg.NumDivisions = 4
		default:
			cfg.NumDivisions = 8
		}
	}
	return cfg
}

// NumRoundsForSwiss returns the number of rounds for a Swiss-system tournament.
func NumRoundsForSwiss(n int) int {
	r := 0
	for p := 1; p < n; p *= 2 {
		r++
	}
	if r < 3 {
		return 3
	}
	return r
}

// ─── internal helpers ─────────────────────────────────────────────────────────

func approvedMembers(members []*models.LeagueMember) []*models.LeagueMember {
	out := make([]*models.LeagueMember, 0, len(members))
	for _, m := range members {
		if m.Status == models.MemberApproved {
			out = append(out, m)
		}
	}
	return out
}

// sortMembersByStanding sorts descending by points → goal-diff → goals-for.
func sortMembersByStanding(members []*models.LeagueMember) {
	sort.SliceStable(members, func(i, j int) bool {
		mi, mj := members[i], members[j]
		if mi.Points != mj.Points {
			return mi.Points > mj.Points
		}
		gdi := mi.GoalsFor - mi.GoalsAgainst
		gdj := mj.GoalsFor - mj.GoalsAgainst
		if gdi != gdj {
			return gdi > gdj
		}
		return mi.GoalsFor > mj.GoalsFor
	})
}

// groupLetters returns ["A","B","C",...] for n groups/divisions.
func groupLetters(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = string(rune('A' + i))
	}
	return out
}

func intPtr(v int) *int { return &v }

// generateGroupMatches creates a round-robin schedule for one group/division.
// Uses the standard polygon rotation: fix players[0], rotate the rest.
// Adds a -1 bye sentinel when the player count is odd.
func generateGroupMatches(leagueID int64, members []*models.LeagueMember, stage string) []*models.Match {
	players := make([]int64, len(members))
	for i, m := range members {
		players[i] = m.UserID
	}

	n := len(players)
	hasBye := n%2 != 0
	if hasBye {
		players = append(players, -1) // sentinel
		n++
	}

	p := make([]int64, n)
	copy(p, players)

	var matches []*models.Match
	for round := 1; round <= n-1; round++ {
		for i := 0; i < n/2; i++ {
			home := p[i]
			away := p[n-1-i]
			if home == -1 || away == -1 {
				continue // skip bye slot
			}
			matches = append(matches, &models.Match{
				LeagueID:   leagueID,
				HomeUserID: home,
				AwayUserID: away,
				Round:      int16(round),
				Status:     models.MatchScheduled,
				Stage:      stage,
			})
		}
		// Rotate: keep p[0] fixed, move last to position 1
		last := p[n-1]
		copy(p[2:], p[1:n-1])
		p[1] = last
	}
	return matches
}

// generateKnockoutBracket seeds userIDs into a single-elimination bracket
// (nearest power-of-2 bracket size; spare slots get byes that auto-advance).
func generateKnockoutBracket(
	ctx context.Context,
	leagueID int64,
	userIDs []int64,
	bracketRepo repository.BracketRepository,
	matchRepo repository.MatchRepository,
) error {
	n := len(userIDs)
	if n < 2 {
		return fmt.Errorf("need at least 2 players for knockout bracket")
	}

	firstStage := models.FirstStageForSize(n)

	// Round bracket size up to the next power of 2.
	bracketSize := models.PrevPowerOf2(n)
	if bracketSize < n {
		bracketSize *= 2
	}

	var slots []*models.BracketSlot
	var matches []*models.Match

	playerIdx := 0
	numSlots := bracketSize / 2

	for i := 0; i < numSlots; i++ {
		var homeID, awayID *int64
		if playerIdx < len(userIDs) {
			id := userIDs[playerIdx]
			homeID = &id
			playerIdx++
		}
		if playerIdx < len(userIDs) {
			id := userIDs[playerIdx]
			awayID = &id
			playerIdx++
		}

		slots = append(slots, &models.BracketSlot{
			LeagueID:   leagueID,
			Stage:      firstStage,
			Slot:       i,
			HomeUserID: homeID,
			AwayUserID: awayID,
		})

		// Only create a match when both seats are filled; single-seed byes
		// advance automatically (handled by the bracket progression logic).
		if homeID != nil && awayID != nil {
			matches = append(matches, &models.Match{
				LeagueID:    leagueID,
				HomeUserID:  *homeID,
				AwayUserID:  *awayID,
				Round:       1,
				Status:      models.MatchScheduled,
				Stage:       firstStage,
				BracketSlot: intPtr(i),
			})
		}
	}

	if err := bracketRepo.CreateSlots(ctx, slots); err != nil {
		return fmt.Errorf("create bracket slots: %w", err)
	}
	if len(matches) > 0 {
		if err := matchRepo.CreateBatch(ctx, matches); err != nil {
			return fmt.Errorf("create bracket matches: %w", err)
		}
	}
	return nil
}

// ─── GroupStageService ────────────────────────────────────────────────────────

// GroupStageService handles group-stage schedule generation.
type GroupStageService struct {
	matchRepo  repository.MatchRepository
	leagueRepo repository.LeagueRepository
}

func NewGroupStageService(mr repository.MatchRepository, lr repository.LeagueRepository) *GroupStageService {
	return &GroupStageService{matchRepo: mr, leagueRepo: lr}
}

// GenerateGroupStage creates a round-robin schedule within each group.
// If no members have a group assigned yet, they are randomly distributed
// into numGroups groups labelled A, B, C, …
func (s *GroupStageService) GenerateGroupStage(ctx context.Context, leagueID int64, numGroups, groupAdvance int) error {
	members, err := s.leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("get members: %w", err)
	}

	approved := approvedMembers(members)
	if len(approved) < 2 {
		return fmt.Errorf("need at least 2 approved members")
	}

	// Auto-assign groups when none are set yet.
	needsAssignment := true
	for _, m := range approved {
		if m.GroupName != "" {
			needsAssignment = false
			break
		}
	}

	if needsAssignment {
		if numGroups < 1 {
			numGroups = 2
		}
		rand.Shuffle(len(approved), func(i, j int) { approved[i], approved[j] = approved[j], approved[i] })
		letters := groupLetters(numGroups)
		for i, m := range approved {
			letter := letters[i%numGroups]
			if err := s.leagueRepo.SetMemberGroup(ctx, leagueID, m.UserID, letter); err != nil {
				return fmt.Errorf("assign group for user %d: %w", m.UserID, err)
			}
			m.GroupName = letter // update in-memory so grouping below works
		}
	}

	byGroup := make(map[string][]*models.LeagueMember)
	for _, m := range approved {
		if m.GroupName != "" {
			byGroup[m.GroupName] = append(byGroup[m.GroupName], m)
		}
	}

	var all []*models.Match
	for group, gm := range byGroup {
		all = append(all, generateGroupMatches(leagueID, gm, group)...)
	}
	if len(all) == 0 {
		return fmt.Errorf("no matches generated")
	}
	return s.matchRepo.CreateBatch(ctx, all)
}

// GeneratePlayoffFromGroups builds a knockout bracket from group-stage results.
// groupAdvance top finishers per group advance, plus bestRunnersUp additional
// best runners-up across all groups (FIFA-style).
func (s *GroupStageService) GeneratePlayoffFromGroups(ctx context.Context, leagueID int64, groupAdvance, bestRunnersUp int, bracketRepo repository.BracketRepository) error {
	groups, err := s.leagueRepo.GetLeagueGroups(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("get groups: %w", err)
	}

	var advancingIDs []int64
	for _, group := range groups {
		gm, err := s.leagueRepo.GetMembersByGroup(ctx, leagueID, group)
		if err != nil {
			return fmt.Errorf("get group %s members: %w", group, err)
		}
		sortMembersByStanding(gm)
		for i := 0; i < groupAdvance && i < len(gm); i++ {
			advancingIDs = append(advancingIDs, gm[i].UserID)
		}
	}

	if bestRunnersUp > 0 {
		ru, err := s.leagueRepo.GetGroupRunnersUp(ctx, leagueID)
		if err != nil {
			return fmt.Errorf("get runners-up: %w", err)
		}
		sortMembersByStanding(ru)
		for i := 0; i < bestRunnersUp && i < len(ru); i++ {
			advancingIDs = append(advancingIDs, ru[i].UserID)
		}
	}

	if len(advancingIDs) < 2 {
		return fmt.Errorf("not enough advancers (%d) for playoff", len(advancingIDs))
	}
	return generateKnockoutBracket(ctx, leagueID, advancingIDs, bracketRepo, s.matchRepo)
}

// ─── CupService ───────────────────────────────────────────────────────────────

// CupService handles cup (knockout) bracket generation.
type CupService struct {
	matchRepo   repository.MatchRepository
	leagueRepo  repository.LeagueRepository
	bracketRepo repository.BracketRepository
}

func NewCupService(mr repository.MatchRepository, lr repository.LeagueRepository, br repository.BracketRepository) *CupService {
	return &CupService{matchRepo: mr, leagueRepo: lr, bracketRepo: br}
}

// GenerateCup randomly seeds all approved members into a single-elimination bracket.
func (s *CupService) GenerateCup(ctx context.Context, leagueID int64) error {
	members, err := s.leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("get members: %w", err)
	}

	approved := approvedMembers(members)
	if len(approved) < 2 {
		return fmt.Errorf("need at least 2 approved members")
	}

	rand.Shuffle(len(approved), func(i, j int) { approved[i], approved[j] = approved[j], approved[i] })

	userIDs := make([]int64, len(approved))
	for i, m := range approved {
		userIDs[i] = m.UserID
	}
	return generateKnockoutBracket(ctx, leagueID, userIDs, s.bracketRepo, s.matchRepo)
}

// ─── SwissService ─────────────────────────────────────────────────────────────

// SwissService handles Swiss-system round generation.
type SwissService struct {
	matchRepo   repository.MatchRepository
	leagueRepo  repository.LeagueRepository
	bracketRepo repository.BracketRepository
}

func NewSwissService(mr repository.MatchRepository, lr repository.LeagueRepository, br repository.BracketRepository) *SwissService {
	return &SwissService{matchRepo: mr, leagueRepo: lr, bracketRepo: br}
}

// GenerateFirstRound creates round 1 with random pairings.
// If the player count is odd the last player in the shuffled list receives a bye.
func (s *SwissService) GenerateFirstRound(ctx context.Context, leagueID int64) error {
	members, err := s.leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("get members: %w", err)
	}

	approved := approvedMembers(members)
	if len(approved) < 2 {
		return fmt.Errorf("need at least 2 approved members")
	}

	rand.Shuffle(len(approved), func(i, j int) { approved[i], approved[j] = approved[j], approved[i] })

	n := len(approved)
	if n%2 != 0 {
		n-- // last player gets a bye
	}

	matches := make([]*models.Match, 0, n/2)
	for i := 0; i+1 < n; i += 2 {
		matches = append(matches, &models.Match{
			LeagueID:   leagueID,
			HomeUserID: approved[i].UserID,
			AwayUserID: approved[i+1].UserID,
			Round:      1,
			Status:     models.MatchScheduled,
			Stage:      "swiss_r1",
		})
	}

	if err := s.matchRepo.CreateBatch(ctx, matches); err != nil {
		return fmt.Errorf("create round-1 matches: %w", err)
	}
	return s.leagueRepo.SetCurrentRound(ctx, leagueID, 1)
}

// GenerateNextRound pairs players with similar standings, preferring opponents
// they have not yet faced.  Falls back to a rematch if no fresh opponent exists.
func (s *SwissService) GenerateNextRound(ctx context.Context, leagueID int64, maxRounds int) error {
	league, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("get league: %w", err)
	}

	nextRound := league.CurrentRound + 1
	if int(nextRound) > maxRounds {
		return fmt.Errorf("tournament complete: max %d rounds reached", maxRounds)
	}

	members, err := s.leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("get members: %w", err)
	}

	approved := approvedMembers(members)
	sortMembersByStanding(approved)

	// Build played-pair set to avoid rematches when possible.
	allMatches, err := s.matchRepo.GetAllForLeague(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("get existing matches: %w", err)
	}
	played := make(map[[2]int64]bool, len(allMatches)*2)
	for _, m := range allMatches {
		played[[2]int64{m.HomeUserID, m.AwayUserID}] = true
		played[[2]int64{m.AwayUserID, m.HomeUserID}] = true
	}

	paired := make([]bool, len(approved))
	var matches []*models.Match

	for i := 0; i < len(approved); i++ {
		if paired[i] {
			continue
		}
		// First preference: nearest unpaired opponent not yet faced.
		foundJ := -1
		for j := i + 1; j < len(approved); j++ {
			if !paired[j] && !played[[2]int64{approved[i].UserID, approved[j].UserID}] {
				foundJ = j
				break
			}
		}
		// Fallback: nearest unpaired (rematch allowed).
		if foundJ == -1 {
			for j := i + 1; j < len(approved); j++ {
				if !paired[j] {
					foundJ = j
					break
				}
			}
		}
		if foundJ == -1 {
			continue // odd player out — bye this round
		}

		paired[i] = true
		paired[foundJ] = true
		matches = append(matches, &models.Match{
			LeagueID:   leagueID,
			HomeUserID: approved[i].UserID,
			AwayUserID: approved[foundJ].UserID,
			Round:      nextRound,
			Status:     models.MatchScheduled,
			Stage:      fmt.Sprintf("swiss_r%d", nextRound),
		})
	}

	if len(matches) == 0 {
		return fmt.Errorf("no matches generated for round %d", nextRound)
	}
	if err := s.matchRepo.CreateBatch(ctx, matches); err != nil {
		return fmt.Errorf("create round-%d matches: %w", nextRound, err)
	}
	return s.leagueRepo.SetCurrentRound(ctx, leagueID, nextRound)
}

// ─── NationsLeagueService ────────────────────────────────────────────────────

// NationsLeagueService handles Nations League format generation.
type NationsLeagueService struct {
	matchRepo   repository.MatchRepository
	leagueRepo  repository.LeagueRepository
	bracketRepo repository.BracketRepository
}

func NewNationsLeagueService(mr repository.MatchRepository, lr repository.LeagueRepository, br repository.BracketRepository) *NationsLeagueService {
	return &NationsLeagueService{matchRepo: mr, leagueRepo: lr, bracketRepo: br}
}

// GenerateNationsLeague distributes approved members across numDivisions
// (labelled A, B, C, …) and generates a round-robin schedule within each.
func (s *NationsLeagueService) GenerateNationsLeague(ctx context.Context, leagueID int64, numDivisions int) error {
	members, err := s.leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("get members: %w", err)
	}

	approved := approvedMembers(members)
	if len(approved) < 2 {
		return fmt.Errorf("need at least 2 approved members")
	}
	if numDivisions < 1 {
		numDivisions = 2
	}

	rand.Shuffle(len(approved), func(i, j int) { approved[i], approved[j] = approved[j], approved[i] })

	letters := groupLetters(numDivisions)
	byDiv := make(map[string][]*models.LeagueMember)

	for i, m := range approved {
		div := letters[i%numDivisions]
		if err := s.leagueRepo.SetMemberDivision(ctx, leagueID, m.UserID, div); err != nil {
			return fmt.Errorf("assign division for user %d: %w", m.UserID, err)
		}
		byDiv[div] = append(byDiv[div], m)
	}

	var all []*models.Match
	for div, dm := range byDiv {
		stage := "div_" + div
		all = append(all, generateGroupMatches(leagueID, dm, stage)...)
	}
	if len(all) == 0 {
		return fmt.Errorf("no matches generated")
	}
	return s.matchRepo.CreateBatch(ctx, all)
}

// GenerateFinalFour creates semi-final bracket slots and matches for the
// top finisher of each division, plus an empty final slot.
func (s *NationsLeagueService) GenerateFinalFour(ctx context.Context, leagueID int64) error {
	divs, err := s.leagueRepo.GetLeagueDivisions(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("get divisions: %w", err)
	}

	var leaders []*models.LeagueMember
	for _, div := range divs {
		dm, err := s.leagueRepo.GetMembersByDivision(ctx, leagueID, div)
		if err != nil {
			return fmt.Errorf("get division %s members: %w", div, err)
		}
		sortMembersByStanding(dm)
		if len(dm) > 0 {
			leaders = append(leaders, dm[0])
		}
	}

	if len(leaders) < 2 {
		return fmt.Errorf("not enough division leaders (%d) for Final Four", len(leaders))
	}

	var slots []*models.BracketSlot
	var matches []*models.Match

	numSFs := len(leaders) / 2
	for i := 0; i < numSFs; i++ {
		home := leaders[i*2].UserID
		away := leaders[i*2+1].UserID
		slots = append(slots, &models.BracketSlot{
			LeagueID:   leagueID,
			Stage:      models.StageSF,
			Slot:       i,
			HomeUserID: &home,
			AwayUserID: &away,
		})
		matches = append(matches, &models.Match{
			LeagueID:    leagueID,
			HomeUserID:  home,
			AwayUserID:  away,
			Round:       int16(i + 1),
			Status:      models.MatchScheduled,
			Stage:       models.StageSF,
			BracketSlot: intPtr(i),
		})
	}

	// Placeholder final slot — winner_user_id filled later by bracket progression.
	slots = append(slots, &models.BracketSlot{
		LeagueID: leagueID,
		Stage:    models.StageFinal,
		Slot:     0,
	})

	if err := s.bracketRepo.CreateSlots(ctx, slots); err != nil {
		return fmt.Errorf("create Final Four slots: %w", err)
	}
	return s.matchRepo.CreateBatch(ctx, matches)
}
