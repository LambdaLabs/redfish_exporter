package collector

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/LambdaLabs/redfish_exporter/internal/ctxlog"
	"golang.org/x/sync/errgroup"
)

type recoverGroup struct {
	ctx context.Context
	eg  errgroup.Group
}

func newRecoverGroup(ctx context.Context) *recoverGroup {
	return &recoverGroup{ctx: ctx}
}

func (seg *recoverGroup) Go(f func() error) {
	seg.eg.Go(func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger := ctxlog.GetContextLogger(seg.ctx)
				logger.Error("panic recovered",
					slog.Any("panic", r),
				)
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		return f()
	})
}

func (seg *recoverGroup) Wait() error {
	return seg.eg.Wait()
}
