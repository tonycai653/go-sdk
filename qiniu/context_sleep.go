package qiniu

import (
	"context"
	"time"
)

// SleepWithContext 等待定时钟到期或者context被取消
func SleepWithContext(ctx context.Context, dur time.Duration) error {
	t := time.NewTimer(dur)
	defer t.Stop()

	select {
	case <-t.C:
		break
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}
