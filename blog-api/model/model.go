package model

import (
	"awesomeProject1/setting"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Tags is persisted as a JSON array and exposed as a string array in APIs.
type Tags []string

func (tags Tags) Value() (driver.Value, error) {
	if tags == nil {
		tags = Tags{}
	}
	return json.Marshal(tags)
}

func (tags *Tags) Scan(value any) error {
	if value == nil {
		*tags = Tags{}
		return nil
	}
	var data []byte
	switch typed := value.(type) {
	case []byte:
		data = typed
	case string:
		data = []byte(typed)
	default:
		return fmt.Errorf("scan article tags: unsupported value type %T", value)
	}
	if len(data) == 0 || string(data) == "null" {
		*tags = Tags{}
		return nil
	}
	return json.Unmarshal(data, tags)
}

func (tags Tags) MarshalJSON() ([]byte, error) {
	if tags == nil {
		return []byte("[]"), nil
	}
	type plainTags []string
	return json.Marshal(plainTags(tags))
}

type User struct {
	gorm.Model
	Name      string    `json:"name" gorm:"type:varchar(255);unique"`
	Password  string    `json:"password" gorm:"type:varchar(255)"`
	Lastlogin time.Time `json:"lastlogin"`
}

type JWts struct {
	gorm.Model
	Token    string    `json:"token" gorm:"size:255;not null"`
	ExpireAt time.Time `json:"expire_at" gorm:"not null"`
}

type Article struct {
	gorm.Model
	Title        string     `json:"title" gorm:"type:varchar(255);not null"`
	Content      string     `json:"content" gorm:"type:text"`
	Image        string     `json:"image" gorm:"type:varchar(255)"`
	Tags         Tags       `json:"tags" gorm:"type:json"`
	Author       User       `gorm:"foreignKey:AuthorID"`
	AuthorID     uint       `json:"author_id" gorm:"not null"`
	Status       string     `json:"status" gorm:"not null;type:varchar(200);default:'draft'"`
	ViewCount    int        `json:"view_count" gorm:"type:int(11);not null;default:0"`
	Version      uint       `json:"version" gorm:"not null;default:1"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
	ContentHash  string     `json:"content_hash,omitempty" gorm:"type:char(64)"`
	IndexStatus  string     `json:"index_status" gorm:"type:varchar(32);not null;default:'not_indexed'"`
	IndexedAt    *time.Time `json:"indexed_at,omitempty"`
	IndexError   string     `json:"index_error,omitempty" gorm:"type:text"`
	ReviewStatus string     `json:"review_status" gorm:"type:varchar(32);not null;default:'not_reviewed';index"`
	ReviewHash   string     `json:"review_hash,omitempty" gorm:"type:char(64)"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
}

type ArticleReview struct {
	gorm.Model
	ArticleID       uint   `json:"article_id" gorm:"not null;index"`
	ArticleVersion  uint   `json:"article_version" gorm:"not null"`
	ContentHash     string `json:"content_hash" gorm:"type:char(64);not null;index"`
	Status          string `json:"status" gorm:"type:varchar(32);not null;index"`
	RiskLevel       string `json:"risk_level" gorm:"type:varchar(16);not null"`
	Score           int    `json:"score"`
	CategoriesJSON  string `json:"-" gorm:"type:text"`
	ReasonsJSON     string `json:"-" gorm:"type:text"`
	SuggestionsJSON string `json:"-" gorm:"type:text"`
	RuleHitsJSON    string `json:"-" gorm:"type:text"`
}
type Comment struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	ArticleID uint      `json:"article_id" gorm:"not null"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	CreatedAt time.Time `json:"created_at"`
}

var dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
	setting.MySQLSetting.User,
	setting.MySQLSetting.Password,
	setting.MySQLSetting.Host,
	setting.MySQLSetting.Name)
var DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
	// 禁止创建数据表的外键约束
	DisableForeignKeyConstraintWhenMigrating: true,
})

func Setup() {
	err := DB.AutoMigrate(&User{})
	if err != nil {
		return
	}
	err1 := DB.AutoMigrate(&JWts{})
	if err1 != nil {
		return
	}
	err2 := DB.AutoMigrate(&Article{})
	if err2 != nil {
		return
	}
	if err3 := DB.AutoMigrate(&Comment{}); err3 != nil {
		return
	}
	if err4 := DB.AutoMigrate(&ArticleReview{}); err4 != nil {
		return
	}
	sqlDB, _ := DB.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
}
