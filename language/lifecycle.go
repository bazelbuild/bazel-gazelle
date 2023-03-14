package language

import "context"

// LifecycleManager allows an extension to initialize and
// free up resources at different points. Extensions should
// embed BaseLifecycleManager instead of implementing this
// interface directly.
type LifecycleManager interface {
	FinishableLanguage
	Init(ctx context.Context)
	DoneResolvingDeps()
}

var _ LifecycleManager = (*BaseLifecycleManager)(nil)

type BaseLifecycleManager struct{}

func (m *BaseLifecycleManager) Init(ctx context.Context) {}

func (m *BaseLifecycleManager) DoneGeneratingRules() {}

func (m *BaseLifecycleManager) DoneResolvingDeps() {}
