package options

import (
	"flag"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
)

const DefaultResyncPeriod = 1 * time.Minute

const (
	TFJob = "tf-job"
)

// ServerOption is the main context object for the queue controller.
type ServerOption struct {
	Kubeconfig   string
	Namespace    string
	ResyncPeriod time.Duration

	// Supported job type
	ListenTFJob bool
	EnabledJobs []string
}

func NewServerOption() *ServerOption {
	s := ServerOption{
		EnabledJobs: []string{},
	}
	return &s
}

func (s *ServerOption) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&s.Namespace, "namespace", v1.NamespaceAll,
		`The namespace to monitor tfjobs. If unset, it monitors all namespaces cluster-wide.
                If set, it only monitors tfjobs in the given namespace.`)

	fs.DurationVar(&s.ResyncPeriod, "resyc-period", DefaultResyncPeriod,
		"Resync interval of the tf-operator")

	fs.BoolVar(&s.ListenTFJob, TFJob, true, fmt.Sprintf("listen to %s\n", TFJob))
	if s.ListenTFJob {
		s.EnabledJobs = append(s.EnabledJobs, TFJob)
	}
}
