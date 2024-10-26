package cdn

type Option interface {
	Apply(opt interface{})
}

type OptionFunc func(interface{})

func (f OptionFunc) Apply(opt interface{}) {
	f(opt)
}
