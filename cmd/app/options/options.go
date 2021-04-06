package options

import (
	"flag"
)

// ServerOption is the main context object for the queue controller.
type ServerOption struct {
	ExtensionConfig string
	KubeConfig      string
	ListenTo        string
}

func NewServerOption() *ServerOption {
	s := ServerOption{}
	return &s
}

func (s *ServerOption) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&s.KubeConfig, "kubeconfig", "", "the path to the kube config")
	fs.StringVar(&s.ListenTo, "listen", "", "the address queue-controller will listen to")
	fs.StringVar(&s.ExtensionConfig, "extensions", "", "the path to the extension configuration file")
}
