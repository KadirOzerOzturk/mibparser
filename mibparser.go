package mibparser

type Option func(*Opts)

type Opts struct {
	Path string
}

type MIBParser struct {
	opts Opts
}

func NewPath(path string) Option {
	return func(opts *Opts) {
		opts.Path = path
	}
}
