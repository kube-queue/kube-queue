package options

import (
	"flag"
)

// ServerOption is the main context object for the queue controller.
type ServerOption struct {
	KubeConfig string
}

func NewServerOption() *ServerOption {
	s := ServerOption{}
	return &s
}

func (s *ServerOption) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&s.KubeConfig, "kubeconfig", "", "the path to the kube config")
}
