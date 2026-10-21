package skill

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// MySQLSkillStore implements SkillStore using MySQL/GORM.
type MySQLSkillStore struct {
	db *gorm.DB
}

// NewMySQLSkillStore creates a MySQL-backed skill store and auto-migrates the schema.
func NewMySQLSkillStore(db *gorm.DB) (*MySQLSkillStore, error) {
	if db == nil {
		return nil, fmt.Errorf("db connection is nil")
	}
	if err := db.AutoMigrate(&Skill{}); err != nil {
		return nil, fmt.Errorf("auto migrate skills table: %w", err)
	}
	return &MySQLSkillStore{db: db}, nil
}

// Save creates or updates a skill. If a skill with the same name exists, it replaces it.
func (s *MySQLSkillStore) Save(ctx context.Context, skill *Skill) error {
	if skill == nil || skill.Name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	db := s.db.WithContext(ctx)

	// Check if skill with the same name exists
	var existing Skill
	err := db.Where("name = ?", skill.Name).First(&existing).Error

	if err == nil {
		// Skill exists, update it
		skill.ID = existing.ID
		skill.CreatedAt = existing.CreatedAt
		skill.UsageCount = existing.UsageCount
		skill.LastUsedAt = existing.LastUsedAt
		skill.UpdatedAt = time.Now()
		return db.Save(skill).Error
	}

	if err == gorm.ErrRecordNotFound {
		// Skill doesn't exist, create new
		skill.CreatedAt = time.Now()
		skill.UpdatedAt = time.Now()
		return db.Create(skill).Error
	}

	return fmt.Errorf("check existing skill: %w", err)
}

// List retrieves skill summaries (name+description only) for prompt injection.
func (s *MySQLSkillStore) List(ctx context.Context, limit int) ([]*SkillSummary, error) {
	if limit <= 0 {
		limit = 50
	}

	var skills []Skill
	err := s.db.WithContext(ctx).
		Select("name, description").
		Order("updated_at DESC").
		Limit(limit).
		Find(&skills).Error

	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}

	summaries := make([]*SkillSummary, 0, len(skills))
	for _, skill := range skills {
		summaries = append(summaries, &SkillSummary{
			Name:        skill.Name,
			Description: skill.Description,
		})
	}
	return summaries, nil
}

// Get retrieves the complete skill details by name.
func (s *MySQLSkillStore) Get(ctx context.Context, name string) (*Skill, error) {
	if name == "" {
		return nil, fmt.Errorf("skill name cannot be empty")
	}

	var skill Skill
	err := s.db.WithContext(ctx).Where("name = ?", name).First(&skill).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("skill not found: %s", name)
		}
		return nil, fmt.Errorf("get skill: %w", err)
	}
	return &skill, nil
}

// IncrUsage increments usage count and updates last used timestamp.
func (s *MySQLSkillStore) IncrUsage(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	db := s.db.WithContext(ctx)
	now := time.Now()
	updates := map[string]interface{}{
		"usage_count": gorm.Expr("usage_count + ?", 1),
		"last_used_at": now,
		"updated_at":   now,
	}

	result := db.Model(&Skill{}).Where("name = ?", name).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("increment usage: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("skill not found: %s", name)
	}
	return nil
}

// DeleteByName removes a skill by name.
func (s *MySQLSkillStore) DeleteByName(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	result := s.db.WithContext(ctx).Where("name = ?", name).Delete(&Skill{})
	if result.Error != nil {
		return fmt.Errorf("delete skill: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("skill not found: %s", name)
	}
	return nil
}

// ListAll retrieves all skills with full details, used by Curator.
func (s *MySQLSkillStore) ListAll(ctx context.Context) ([]*Skill, error) {
	var skills []*Skill
	err := s.db.WithContext(ctx).Find(&skills).Error
	if err != nil {
		return nil, fmt.Errorf("list all skills: %w", err)
	}
	return skills, nil
}
