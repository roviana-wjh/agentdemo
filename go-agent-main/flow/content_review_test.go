package flow

import "testing"

func TestEvaluateContentRulesRejectsSensitiveWords(t *testing.T) {
	issues := evaluateContentRules(ContentReviewRequest{
		Title:   "测试文章",
		Content: "这是一篇包含赌博推广内容的长文章。这里补充足够多的文字，用于通过正文长度检查，并验证敏感词规则拥有最高优先级。",
	}, []string{"赌博"})
	status, risk := decideReviewStatus(issues, "low")
	if status != ReviewStatusRejected || risk != "high" {
		t.Fatalf("status=%s risk=%s issues=%+v", status, risk, issues)
	}
	if issues[0].Keyword == "赌博" {
		t.Fatal("sensitive keyword must be masked")
	}
}

func TestEvaluateContentRulesFindsMarkdownProblems(t *testing.T) {
	issues := evaluateContentRules(ContentReviewRequest{
		Title: "Go 并发实践",
		Content: "```go\nfunc main() {}\n" +
			"这里补充一段足够长的正文，介绍 goroutine、channel 和 context 的使用方式，以及服务端并发控制中的常见注意事项。",
	}, nil)
	status, risk := decideReviewStatus(issues, "low")
	if status != ReviewStatusNeedsRevision || risk != "medium" {
		t.Fatalf("status=%s risk=%s issues=%+v", status, risk, issues)
	}
}

func TestSemanticHighRiskCannotBeOverriddenByRules(t *testing.T) {
	status, risk := decideReviewStatus(nil, "high")
	if status != ReviewStatusRejected || risk != "high" {
		t.Fatalf("status=%s risk=%s", status, risk)
	}
}

func TestHighRiskRuleHasPriorityWhenSemanticReviewIsUnavailable(t *testing.T) {
	issues := []ReviewIssue{{Type: "sensitive_word", Message: "hit", Severity: "high"}}
	status, risk := decideReviewStatus(issues, "low")
	if status != ReviewStatusRejected || risk != "high" {
		t.Fatalf("status=%s risk=%s", status, risk)
	}
}

func TestExtractJSONObjectSupportsFencedOutput(t *testing.T) {
	got := extractJSONObject("```json\n{\"risk_level\":\"low\",\"score\":95}\n```")
	if got != `{"risk_level":"low","score":95}` {
		t.Fatalf("got %q", got)
	}
}
