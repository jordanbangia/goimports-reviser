package reviser

type Options struct {
	RemoveUnusedImports   bool
	AliasForVersionSuffix bool

	ExtraImportGroups []string
}

type Option interface {
	apply(*Options)
}

func WithRemoveUnusedImports(removeUnusedImports bool) Option {
	return newFuncCallOption(func(o *Options) {
		o.RemoveUnusedImports = removeUnusedImports
	})
}

func WithAliasForVersionSuffix(aliasForVersionSuffix bool) Option {
	return newFuncCallOption(func(o *Options) {
		o.AliasForVersionSuffix = aliasForVersionSuffix
	})
}

func WithExtraImportGroups(groups []string) Option {
	return newFuncCallOption(func(o *Options) {
		o.ExtraImportGroups = groups
	})
}

func applyOptions(optionList []Option) *Options {
	options := &Options{}
	for _, option := range optionList {
		option.apply(options)
	}
	return options
}

type funcCallOption struct {
	f func(*Options)
}

func (fdo *funcCallOption) apply(do *Options) {
	fdo.f(do)
}

func newFuncCallOption(f func(*Options)) *funcCallOption {
	return &funcCallOption{
		f: f,
	}
}
