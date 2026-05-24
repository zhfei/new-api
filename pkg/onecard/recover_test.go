package onecard

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/types"
)

func TestShouldAutoDisable429(t *testing.T) {
	err := &types.NewAPIError{StatusCode: http.StatusTooManyRequests, Err: errors.New("rate limit")}
	if !ShouldAutoDisable429(&RequestContext{}, ChannelInfo{Type: constant.ChannelTypeCodex}, err) {
		t.Fatal("expected 429 to be auto-disabled")
	}

	err.StatusCode = http.StatusUnauthorized
	if ShouldAutoDisable429(&RequestContext{}, ChannelInfo{Type: constant.ChannelTypeCodex}, err) {
		t.Fatal("expected non-429 to skip onecard auto-disable")
	}
}

func TestCodex429RecoverAt(t *testing.T) {
	now := int64(1716547200)
	policy := &Codex429RecoverPolicy{}

	tests := []struct {
		name string
		err  *types.NewAPIError
		want int64
	}{
		{
			name: "five hours",
			err:  &types.NewAPIError{StatusCode: http.StatusTooManyRequests, Err: errors.New("try again in 5 hours")},
			want: time.Unix(now, 0).Add(5 * time.Hour).Unix(),
		},
		{
			name: "weekly limit",
			err:  &types.NewAPIError{StatusCode: http.StatusTooManyRequests, Err: errors.New("weekly limit reached, try again in 7 days")},
			want: time.Unix(now, 0).Add(7 * 24 * time.Hour).Unix(),
		},
		{
			name: "fallback",
			err:  &types.NewAPIError{StatusCode: http.StatusTooManyRequests, Err: errors.New("rate limit reached")},
			want: time.Unix(now, 0).Add(5 * time.Hour).Unix(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policy.RecoverAt(&RequestContext{}, tt.err, now); got != tt.want {
				t.Fatalf("RecoverAt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAutoRecoverInfoHelpers(t *testing.T) {
	info := map[string]interface{}{
		OtherInfoAutoRecoverReasonKey: AutoRecoverReason429,
		OtherInfoAutoRecoverAtKey:     float64(1716565200),
	}
	if !Is429AutoRecoverInfo(info) {
		t.Fatal("expected 429 recover info")
	}
	recoverAt, ok := ParseAutoRecoverAt(info)
	if !ok || recoverAt != 1716565200 {
		t.Fatalf("ParseAutoRecoverAt() = %d, %v", recoverAt, ok)
	}

	cleaned := Clean429AutoRecoverInfo(info)
	if _, exists := cleaned[OtherInfoAutoRecoverReasonKey]; exists {
		t.Fatal("expected recover reason to be cleaned")
	}
}

func TestAutoRecoverKeyInfoHelpers(t *testing.T) {
	now := int64(1716547200)
	info := map[string]interface{}{}
	dueInfo := map[string]interface{}{
		OtherInfoAutoRecoverReasonKey: AutoRecoverReason429,
		OtherInfoAutoRecoverAtKey:     now - 1,
	}
	futureInfo := map[string]interface{}{
		OtherInfoAutoRecoverReasonKey: AutoRecoverReason429,
		OtherInfoAutoRecoverAtKey:     now + 3600,
	}

	info = Set429AutoRecoverKeyInfo(info, 0, dueInfo)
	info = Set429AutoRecoverKeyInfo(info, 1, futureInfo)

	dueIndexes := Due429AutoRecoverKeyIndexes(info, now)
	if len(dueIndexes) != 1 || dueIndexes[0] != 0 {
		t.Fatalf("Due429AutoRecoverKeyIndexes() = %v, want [0]", dueIndexes)
	}
	if !Has429AutoRecoverKeys(info) {
		t.Fatal("expected key recover info")
	}

	info = Clean429AutoRecoverKeyInfo(info, 0)
	dueIndexes = Due429AutoRecoverKeyIndexes(info, now+7200)
	if len(dueIndexes) != 1 || dueIndexes[0] != 1 {
		t.Fatalf("Due429AutoRecoverKeyIndexes() after clean = %v, want [1]", dueIndexes)
	}

	info = Clean429AutoRecoverKeyInfo(info, 1)
	if Has429AutoRecoverKeys(info) {
		t.Fatal("expected all key recover info to be cleaned")
	}
}
