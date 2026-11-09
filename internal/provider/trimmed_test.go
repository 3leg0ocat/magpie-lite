package provider

import "testing"

// magpie-lite offers its own list to add; every name on it is a preset, and
// one left off still reads, for a provider saved from it.
func TestTrimmedOffersItsList(t *testing.T) {
	was := trimmed
	t.Cleanup(func() { trimmed = was })
	for id := range trimmedPresets {
		if builtinPreset(id) == nil {
			t.Errorf("%s is no preset", id)
		}
	}
	if !Offered("openrouter") {
		t.Fatal("untrimmed: every preset is offered")
	}
	trimmed = "on"
	for _, id := range []string{"anthropic", "qwen-cn", "bailian-token-plan", "ollama", RemoteMagpiePreset} {
		if !Offered(id) {
			t.Errorf("%s isn't offered", id)
		}
	}
	for _, id := range []string{"openrouter", "qwen", "stepfun", "huaweicloud", "longcat", "bedrock", VertexPreset} {
		if Offered(id) {
			t.Errorf("%s is offered", id)
		}
		if Preset(id) == nil {
			t.Errorf("%s no longer reads", id)
		}
	}
	if Partners() != nil || PartnersNow(0) != nil {
		t.Fatal("trimmed: a partner list")
	}
}
