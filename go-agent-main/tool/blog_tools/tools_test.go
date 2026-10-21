package blog_tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

func TestNewToolsExposesTask9ContractAndNormalizesLimits(t *testing.T) {
	var rawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	tools, err := NewToolsWithClient(client)
	if err != nil {
		t.Fatalf("NewToolsWithClient() error = %v", err)
	}
	wants := []string{"search_article", "get_article_detail", "get_hot_articles", "get_recent_comments", "get_hot_tags", "get_blog_statistics"}
	if len(tools) != len(wants) {
		t.Fatalf("tool count = %d", len(tools))
	}
	for i, baseTool := range tools {
		info, err := baseTool.Info(context.Background())
		if err != nil {
			t.Fatalf("Info() error = %v", err)
		}
		if info.Name != wants[i] {
			t.Errorf("tool[%d] name = %q, want %q", i, info.Name, wants[i])
		}
	}

	if _, err := tools[2].(tool.InvokableTool).InvokableRun(context.Background(), `{"limit":100}`); err != nil {
		t.Fatalf("get_hot_articles error = %v", err)
	}
	if rawQuery != "limit=20" {
		t.Fatalf("query = %q, want limit=20", rawQuery)
	}
}

func TestNewToolsRequiresInternalToken(t *testing.T) {
	if _, err := NewTools("http://127.0.0.1:8080", "", 0); err == nil {
		t.Fatal("NewTools() accepted an empty internal token")
	}
}

func TestToolsRejectInvalidRequiredValues(t *testing.T) {
	client, err := NewClient("http://127.0.0.1", "secret", http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	tools, err := NewToolsWithClient(client)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		index int
		args  string
	}{
		{index: 0, args: `{"keyword":"  "}`},
		{index: 1, args: `{"article_id":0}`},
	} {
		if _, err := tools[tc.index].(tool.InvokableTool).InvokableRun(context.Background(), tc.args); err == nil {
			t.Errorf("tool %d accepted invalid args", tc.index)
		}
	}
}
