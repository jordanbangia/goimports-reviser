package reviser

type options struct {
	RemoveUnusedImports      bool
	UseAliasForVersionSuffix bool
}

type Option interface {
	Apply(o *options)
}

func WithRemoveUnusedImports(removeUnusedImports bool) Option {
	return withApplyFunc(func(o *options) {
		o.RemoveUnusedImports = removeUnusedImports
	})
}

func WithUseAlias(useAlias bool) Option {
	return withApplyFunc(func(o *options) {
		o.UseAliasForVersionSuffix = useAlias
	})
}

type applier struct {
	f func(o *options)
}

func (a *applier) Apply(o *options) {
	a.f(o)
}

func withApplyFunc(f func(o *options)) *applier {
	return &applier{f: f}
}

func apply(os []Option) *options {
	o := &options{}
	for _, option := range os {
		option.Apply(o)
	}

	return o
}
