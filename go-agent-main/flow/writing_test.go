package flow

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type recordingDraftTool struct {
	name  string
	calls int
	args  string
}

func (t *recordingDraftTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: t.name}, nil
}

func (t *recordingDraftTool) InvokableRun(_ context.Context, arguments string, _ ...tool.Option) (string, error) {
	t.calls++
	t.args = arguments
	return `{"id":9,"title":"Draft","version":1,"status":"draft"}`, nil
}

func TestSaveDraftRequiresConfirmationAndSelectsCreateOrUpdate(t *testing.T) {
	create := &recordingDraftTool{name: "create_article_draft"}
	update := &recordingDraftTool{name: "update_article_draft"}
	flow := &WritingFlow{createDraft: create, updateDraft: update}

	if _, err := flow.SaveDraft(context.Background(), SaveDraftRequest{
		AuthorID: 7, Title: "Draft", Content: "Body",
	}); err == nil {
		t.Fatal("SaveDraft() accepted an unconfirmed write")
	}
	if create.calls != 0 || update.calls != 0 {
		t.Fatal("a draft tool ran before confirmation")
	}

	result, err := flow.SaveDraft(context.Background(), SaveDraftRequest{
		AuthorID: 7, Title: "Draft", Content: "Body", Confirmed: true,
	})
	if err != nil || result.Status != "draft" || create.calls != 1 {
		t.Fatalf("create result = %+v, err = %v, calls = %d", result, err, create.calls)
	}
	if _, err := flow.SaveDraft(context.Background(), SaveDraftRequest{
		AuthorID: 7, ArticleID: 9, Title: "Draft", Content: "Updated", Confirmed: true,
	}); err != nil || update.calls != 1 {
		t.Fatalf("update err = %v, calls = %d", err, update.calls)
	}
}
