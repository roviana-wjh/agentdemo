package memory

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

// 记忆类型
const (
	MemoryTypeIdentity       = "identity"        // 身份：我是谁、公司、角色
	MemoryTypePreference     = "preference"      // 偏好：喜欢/习惯
	MemoryTypeProjectContext = "project_context" // 项目上下文：技术栈、正在做什么
)

// UserMemory 是跨会话的长期记忆，按 user_id 存储（区别于 session 粒度）。
type UserMemory struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     string    `gorm:"index:idx_user_type;size:64;not null" json:"user_id"`
	Type       string    `gorm:"index:idx_user_type;size:32;not null" json:"type"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 指定表名。
func (UserMemory) TableName() string {
	return "user_memories"
}

// UserMemoryStore 抽象长期记忆存储。
type UserMemoryStore interface {
	// Append 写入若干条记忆，按 (user_id, content) 去重：已存在则更新置信度与时间，不存在则插入。
	Append(ctx context.Context, mems []UserMemory) error
	// List 返回某用户的记忆，按 UpdatedAt 倒序，最多 limit 条（limit<=0 表示不限）。
	List(ctx context.Context, userID string, limit int) ([]UserMemory, error)
	// Delete 删除某用户下 content 完全匹配的记忆（Curator 用）。
	Delete(ctx context.Context, userID, content string) error
}

// ParseUserID 从 sessionID 解析出 userID。
// 约定格式 "{userID}:{convID}"；无冒号时（旧格式）整体作为 userID。
func ParseUserID(sessionID string) string {
	if i := strings.Index(sessionID, ":"); i > 0 {
		return sessionID[:i]
	}
	return sessionID
}

// ---------- MySQL 实现 ----------

// MySQLUserMemoryStore 基于 gorm 的实现。
type MySQLUserMemoryStore struct {
	db *gorm.DB
}

// NewMySQLUserMemoryStore 创建实现并自动建表。
func NewMySQLUserMemoryStore(db *gorm.DB) (*MySQLUserMemoryStore, error) {
	if err := db.AutoMigrate(&UserMemory{}); err != nil {
		return nil, err
	}
	return &MySQLUserMemoryStore{db: db}, nil
}

func (s *MySQLUserMemoryStore) Append(ctx context.Context, mems []UserMemory) error {
	if len(mems) == 0 {
		return nil
	}
	db := s.db.WithContext(ctx)

	for _, m := range mems {
		if m.UserID == "" || strings.TrimSpace(m.Content) == "" {
			continue
		}

		// 按 (user_id, content) 去重：先查是否存在
		var existing UserMemory
		err := db.Where("user_id = ? AND content = ?", m.UserID, m.Content).
			First(&existing).Error

		switch {
		case err == nil:
			// 已存在：更新置信度与时间（不新增）
			if updErr := db.Model(&existing).
				Updates(map[string]interface{}{
					"confidence": m.Confidence,
					"type":       m.Type,
					"updated_at": time.Now(),
				}).Error; updErr != nil {
				return updErr
			}
		case err == gorm.ErrRecordNotFound:
			// 不存在：插入
			if createErr := db.Create(&UserMemory{
				UserID:     m.UserID,
				Type:       m.Type,
				Content:    m.Content,
				Confidence: m.Confidence,
			}).Error; createErr != nil {
				return createErr
			}
		default:
			return err
		}
	}
	return nil
}

func (s *MySQLUserMemoryStore) List(ctx context.Context, userID string, limit int) ([]UserMemory, error) {
	var out []UserMemory
	q := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *MySQLUserMemoryStore) Delete(ctx context.Context, userID, content string) error {
	return s.db.WithContext(ctx).
		Where("user_id = ? AND content = ?", userID, content).
		Delete(&UserMemory{}).Error
}

// ---------- 降级实现 ----------

// NoopUserMemoryStore 在 MySQL 不可用时使用，所有操作为空，保证系统退回纯 session 模式而不崩溃。
type NoopUserMemoryStore struct{}

func (NoopUserMemoryStore) Append(ctx context.Context, mems []UserMemory) error { return nil }
func (NoopUserMemoryStore) List(ctx context.Context, userID string, limit int) ([]UserMemory, error) {
	return nil, nil
}
func (NoopUserMemoryStore) Delete(ctx context.Context, userID, content string) error { return nil }
