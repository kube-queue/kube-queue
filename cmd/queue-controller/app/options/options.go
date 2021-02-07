package options

import (
	"flag"
)

type ExtensionConfig struct {
	TypeAddr map[string]string `json:"type_addr"`
}

// ServerOption is the main context object for the queue controller.
type ServerOption struct {
	ExtensionConfig       string
	KubeConfig            string
	PermissionCounterAddr string
	ListenTo              string
}

func NewServerOption() *ServerOption {
	s := ServerOption{}
	return &s
}

func (s *ServerOption) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&s.KubeConfig, "kubeconfig", "", "the path to the kube config")
	fs.StringVar(&s.ListenTo, "listen", "", "the address queue-controller will listen to")
	fs.StringVar(&s.PermissionCounterAddr, "permission-counter", "",
		"the address of permission counter")
	fs.StringVar(&s.ExtensionConfig, "extensions", "", "the path to the extension configuration file")
}
