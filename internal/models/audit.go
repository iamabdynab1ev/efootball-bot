package models

import "time"

// AuditEntry — запись журнала действий (кто/когда/что сделал).
//
// ActorID/TargetID/EntityID/LeagueID — указатели, т.к. действие может быть
// системным (нет актора) или не относиться к лиге/цели. ActorName/TargetName
// заполняются только при выборке (JOIN users) и при живой публикации — в БД их
// нет, чтобы не дублировать имя на каждой строке.
type AuditEntry struct {
	ID         int64          `json:"id"`
	ActorID    *int64         `json:"actor_id,omitempty"`
	ActorName  string         `json:"actor_name,omitempty"`
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type,omitempty"`
	EntityID   *int64         `json:"entity_id,omitempty"`
	LeagueID   *int64         `json:"league_id,omitempty"`
	TargetID   *int64         `json:"target_id,omitempty"`
	TargetName string         `json:"target_name,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	IP         string         `json:"ip,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// AuditFilter — параметры выборки ленты аудита. Все поля опциональны; BeforeID
// реализует keyset-пагинацию (id < BeforeID), стабильную под вставками.
type AuditFilter struct {
	ActorID  *int64
	TargetID *int64
	LeagueID *int64
	Action   string
	BeforeID int64
	Limit    int
}

// Действия аудита. Константы — единый словарь, чтобы фронт и бэкенд не
// расходились в строках и можно было фильтровать/локализовать по коду.
const (
	AuditLogin            = "login"
	AuditJoinLeague       = "league.join"
	AuditSubmitResult     = "match.result"
	AuditConfirmMatch     = "match.confirm"
	AuditDisputeMatch     = "match.dispute"
	AuditAdminResolve     = "match.admin_resolve"
	AuditLeagueCreate     = "league.create"
	AuditLeagueUpdate     = "league.update"
	AuditLeagueDelete     = "league.delete"
	AuditLeagueDraw       = "league.draw"
	AuditMemberApprove    = "member.approve"
	AuditMemberReject     = "member.reject"
	AuditUserDelete       = "user.delete"
	AuditUserBan          = "user.ban"
	AuditBroadcast        = "admin.broadcast"
)
