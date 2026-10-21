package skill

import (
	"context"
	"time"
)

// Skill represents a learned workflow pattern extracted from successful tool call sequences.
// Skills are globally shared (not user-specific) as they capture reusable procedural knowledge.
type Skill struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:128;not null" json:"name"`
	Description string    `gorm:"size:512;not null" json:"description"`
	TriggerTags string    `gorm:"type:text" json:"trigger_tags"`
	Procedure   string    `gorm:"type:text;not null" json:"procedure"`
	ToolsUsed   string    `gorm:"type:text" json:"tools_used"`
	UsageCount  int       `gorm:"default:0" json:"usage_count"`
	LastUsedAt  time.Time `gorm:"index" json:"last_used_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Skill) TableName() string {
	return "skills"
}

// SkillSummary is a lightweight representation containing only name and description,
// used for injecting skill index into prompts without bloating context.
type SkillSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SkillStore defines the interface for persisting and retrieving skills.
type SkillStore interface {
	// Save creates or updates a skill. If a skill with the same name exists, it replaces it.
	Save(ctx context.Context, skill *Skill) error

	// List retrieves skill summaries (name+description only) for injecting into prompts.
	List(ctx context.Context, limit int) ([]*SkillSummary, error)

	// Get retrieves the complete skill details by name, used by skill_view tool.
	Get(ctx context.Context, name string) (*Skill, error)

	// IncrUsage increments usage count and updates last used timestamp.
	IncrUsage(ctx context.Context, name string) error

	// DeleteByName removes a skill by name, used by Curator for cleanup.
	DeleteByName(ctx context.Context, name string) error

	// ListAll retrieves all skills with full details, used by Curator for quality checks.
	ListAll(ctx context.Context) ([]*Skill, error)
}

// NoopSkillStore is a no-op implementation used when MySQL is unavailable.
// It allows the system to continue operating without skill learning functionality.
type NoopSkillStore struct{}

func (NoopSkillStore) Save(ctx context.Context, skill *Skill) error {
	return nil
}

func (NoopSkillStore) List(ctx context.Context, limit int) ([]*SkillSummary, error) {
	return nil, nil
}

func (NoopSkillStore) Get(ctx context.Context, name string) (*Skill, error) {
	return nil, nil
}

func (NoopSkillStore) IncrUsage(ctx context.Context, name string) error {
	return nil
}

func (NoopSkillStore) DeleteByName(ctx context.Context, name string) error {
	return nil
}

func (NoopSkillStore) ListAll(ctx context.Context) ([]*Skill, error) {
	return nil, nil
}
