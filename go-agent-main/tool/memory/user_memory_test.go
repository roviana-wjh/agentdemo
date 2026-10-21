package memory

import (
	"context"
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestParseUserID(t *testing.T) {
	cases := []struct {
		name      string
		sessionID string
		want      string
	}{
		{"标准格式", "u123:conv456", "u123"},
		{"多段冒号只取第一段", "u123:conv456:extra", "u123"},
		{"旧格式无冒号", "default_user", "default_user"},
		{"空串", "", ""},
		{"冒号开头视为无 userID", ":conv456", ":conv456"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ParseUserID(c.sessionID); got != c.want {
				t.Errorf("ParseUserID(%q) = %q, want %q", c.sessionID, got, c.want)
			}
		})
	}
}

// 集成测试：仅在设置 TEST_MYSQL_DSN 时运行，验证 Append/List/去重。
// 例如：TEST_MYSQL_DSN="user:pass@tcp(127.0.0.1:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
func TestMySQLUserMemoryStore(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN 未设置，跳过 MySQL 集成测试")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接测试库失败: %v", err)
	}

	store, err := NewMySQLUserMemoryStore(db)
	if err != nil {
		t.Fatalf("建表失败: %v", err)
	}

	ctx := context.Background()
	const uid = "test_user_ut"

	// 清理，保证可重复运行
	db.Where("user_id = ?", uid).Delete(&UserMemory{})
	t.Cleanup(func() {
		db.Where("user_id = ?", uid).Delete(&UserMemory{})
	})

	// Append 3 条
	err = store.Append(ctx, []UserMemory{
		{UserID: uid, Type: MemoryTypeIdentity, Content: "用户是 Go 后端", Confidence: 0.9},
		{UserID: uid, Type: MemoryTypePreference, Content: "喜欢简洁回答", Confidence: 0.8},
		{UserID: uid, Type: MemoryTypeProjectContext, Content: "项目用 MySQL+Milvus", Confidence: 0.85},
	})
	if err != nil {
		t.Fatalf("Append 失败: %v", err)
	}

	got, err := store.List(ctx, uid, 0)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("期望 3 条，实际 %d 条", len(got))
	}

	// 重复 Append 同 content，应去重（仍 3 条），且置信度被更新
	err = store.Append(ctx, []UserMemory{
		{UserID: uid, Type: MemoryTypeIdentity, Content: "用户是 Go 后端", Confidence: 0.99},
	})
	if err != nil {
		t.Fatalf("重复 Append 失败: %v", err)
	}

	got, err = store.List(ctx, uid, 0)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("去重后期望 3 条，实际 %d 条", len(got))
	}

	// 空 UserID / 空 Content 应被跳过
	if err := store.Append(ctx, []UserMemory{
		{UserID: "", Content: "无主", Confidence: 0.9},
		{UserID: uid, Content: "   ", Confidence: 0.9},
	}); err != nil {
		t.Fatalf("Append 边界项失败: %v", err)
	}
	got, _ = store.List(ctx, uid, 0)
	if len(got) != 3 {
		t.Fatalf("边界项应被跳过，仍应为 3 条，实际 %d 条", len(got))
	}

	// Delete 一条
	if err := store.Delete(ctx, uid, "喜欢简洁回答"); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	got, _ = store.List(ctx, uid, 0)
	if len(got) != 2 {
		t.Fatalf("删除后期望 2 条，实际 %d 条", len(got))
	}
}
