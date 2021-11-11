/*
 Copyright 2021 The Kube-Queue Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package options

import (
	"flag"
)

// ServerOption is the main context object for the queue controller.
type ServerOption struct {
	KubeConfig string
	// QPS indicates the maximum QPS to the master from this client.
	// If it's zero, the created RESTClient will use DefaultQPS: 5
	QPS int
	// Maximum burst for throttle.
	// If it's zero, the created RESTClient will use DefaultBurst: 10.
	Burst int
}

func NewServerOption() *ServerOption {
	s := ServerOption{}
	return &s
}

func (s *ServerOption) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&s.KubeConfig, "kubeconfig", "", "the path to the kube config")
	fs.IntVar(&s.QPS, "qps", 5, "QPS indicates the maximum QPS to the master from this client.")
	fs.IntVar(&s.Burst, "brust", 10, "Maximum burst for throttle.")
}
