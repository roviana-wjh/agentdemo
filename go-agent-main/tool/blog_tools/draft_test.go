package blog_tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

func TestDraftToolsUseInternalAPIAndNeverPublish(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Header.Get("X-Internal-Token") != "secret" {
			t.Fatal("missing internal token")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, exists := body["status"]; exists {
			t.Fatal("draft tool must not accept a status field")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":9,"title":"Agent","version":1,"status":"draft"}}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	tools, err := NewDraftToolsWithClient(client)
	if err != nil {
		t.Fatal(err)
	}
	wants := []string{"create_article_draft", "update_article_draft"}
	for i, created := range tools {
		info, _ := created.Info(context.Background())
		if info.Name != wants[i] {
			t.Fatalf("tool[%d] = %q", i, info.Name)
		}
	}
	result, err := tools[0].(tool.InvokableTool).InvokableRun(context.Background(), `{"author_id":7,"title":"Agent","content":"# Body"}`)
	if err != nil || result == "" {
		t.Fatalf("create draft = %q, %v", result, err)
	}
	if _, err := tools[1].(tool.InvokableTool).InvokableRun(context.Background(), `{"author_id":7,"article_id":9,"title":"Agent","content":"# Updated"}`); err != nil {
		t.Fatalf("update draft: %v", err)
	}
	if len(paths) != 2 || paths[0] != "/internal/v1/tools/create_article_draft" || paths[1] != "/internal/v1/tools/update_article_draft" {
		t.Fatalf("paths = %v", paths)
	}
}

func TestDraftToolsRejectMissingOwnershipAndContent(t *testing.T) {
	client, _ := NewClient("http://127.0.0.1", "secret", http.DefaultClient)
	tools, err := NewDraftToolsWithClient(client)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		index int
		args  string
	}{
		{0, `{"author_id":0,"title":"x","content":"y"}`},
		{0, `{"author_id":1,"title":" ","content":"y"}`},
		{1, `{"author_id":1,"article_id":0,"title":"x","content":"y"}`},
	} {
		if _, err := tools[tc.index].(tool.InvokableTool).InvokableRun(context.Background(), tc.args); err == nil {
			t.Errorf("tool %d accepted %s", tc.index, tc.args)
		}
	}
}
