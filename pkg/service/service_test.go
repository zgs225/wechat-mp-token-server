package service

import (
	"context"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/silenceper/wechat/cache"
)

func TestBasicService_GetToken(t *testing.T) {
	s := New(cache.NewMemory(), log.NewLogfmtLogger(os.Stderr))
	tk, err := s.GetToken(context.Background(), "wx6ccc1e9fcd0395d0", "42b1cff352229d3fef919d31bda85e6d")
	if err != nil {
		t.Error(err)
	}
	t.Log("AccessToken is ", tk)
}
