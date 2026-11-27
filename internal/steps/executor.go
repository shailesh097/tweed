package steps

import (
	"context"
	"fmt"

	"github.com/shailesh097/tweed/internal/config"
	"github.com/shailesh097/tweed/internal/runner"
	"github.com/shailesh097/tweed/internal/ui"
)

// Summary counts what happened during an apply.
type Summary struct {
	Applied, Skipped, Failed int
}

// String formats the summary for the end-of-run report.
func (s Summary) String() string {
	return fmt.Sprintf("%d applied, %d already done, %d failed", s.Applied, s.Skipped, s.Failed)
}

// Options controls Execute.
type Options struct {
	KeepGoing bool // continue after a failed step instead of stopping
	NoUpgrade bool // leave out upgrade steps (apply --no-upgrade)
}

// Execute runs the steps of each named module in order.
func Execute(ctx context.Context, cfg *config.Config, modules []string, r runner.Runner, log *ui.Logger, opt Options) (Summary, error) {
	var sum Summary
	var firstErr error
	for _, name := range modules {
		log.Header("module %s", name)
		for _, st := range FromModule(cfg.Modules[name]) {
			if err := ctx.Err(); err != nil {
				return sum, err
			}
			if _, ok := st.(*upgradeStep); ok && opt.NoUpgrade {
				log.Skip("%s (skipped: --no-upgrade)", st.Describe())
				continue
			}
			done, err := st.Check(ctx, r)
			if err != nil && !r.DryRun() {
				log.Warn("%s: could not check state (%v); applying anyway", st.Describe(), err)
			}
			if done {
				sum.Skipped++
				log.Skip("%s (already done)", st.Describe())
				continue
			}
			log.Info("%s", st.Describe())
			if err := st.Apply(ctx, r); err != nil {
				sum.Failed++
				log.Error("%s: %v", st.Describe(), err)
				if firstErr == nil {
					firstErr = fmt.Errorf("module %s: %s: %w", name, st.Describe(), err)
				}
				if !opt.KeepGoing {
					return sum, firstErr
				}
				continue
			}
			sum.Applied++
			if !r.DryRun() {
				log.Success("%s", st.Describe())
			}
		}
	}
	return sum, firstErr
}
