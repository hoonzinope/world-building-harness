package harness

import (
	"context"

	"github.com/hoonzi/world-harness/internal/harness/story"
)

type mockGMProvider struct{}

func (mockGMProvider) Generate(ctx context.Context, req gmRequest) (gmOutput, string, string, string, error) {
	return story.NewGMProvider("mock").Generate(ctx, req)
}

func containsString(in []string, want string) bool {
	for _, got := range in {
		if got == want {
			return true
		}
	}
	return false
}

