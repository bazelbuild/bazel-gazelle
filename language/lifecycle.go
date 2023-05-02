package language

import "context"

// LifecycleManager allows an extension to initialize and
// free up resources at different points. Extensions should
// embed BaseLifecycleManager instead of implementing this
// interface directly.
type LifecycleManager interface {
	FinishableLanguage
	Before(ctx context.Context)
	AfterResolvingDeps(ctx context.Context)
}

var _ LifecycleManager = (*BaseLifecycleManager)(nil)

type BaseLifecycleManager struct{}

func (m *BaseLifecycleManager) Before(ctx context.Context) {}

func (m *BaseLifecycleManager) DoneGeneratingRules() {}

func (m *BaseLifecycleManager) AfterResolvingDeps(ctx context.Context) {}
