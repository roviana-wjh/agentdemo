package setting

import "testing"

func TestSharedEnvValuePrefersProcessEnvironment(t *testing.T) {
	t.Setenv("BLOG_INTERNAL_TOKEN_TEST", "from-process")
	if got := sharedEnvValue("BLOG_INTERNAL_TOKEN_TEST"); got != "from-process" {
		t.Fatalf("sharedEnvValue() = %q", got)
	}
}
