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

package priority

import (
	"github.com/kube-queue/api/pkg/apis/scheduling/v1alpha1"
	"github.com/kube-queue/kube-queue/pkg/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"testing"
)

func TestQueueLess(t *testing.T) {
	tests := []struct {
		name    string
		quInfo1 *framework.QueueUnitInfo
		quInfo2 *framework.QueueUnitInfo
		want    bool
	}{
		{
			name: "qu1's priority greater than qu2",
			quInfo1: &framework.QueueUnitInfo{
				Unit: makeQueueUnit("qu1", 100),
			},
			quInfo2: &framework.QueueUnitInfo{
				Unit: makeQueueUnit("qu2", 50),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Priority{}
			if got := p.QueueLess(tt.quInfo1, tt.quInfo2); got != tt.want {
				t.Errorf("Less() = %v, want %v", got, tt.want)
			}
		})
	}
}

func makeQueueUnit(name string, priority int32) *v1alpha1.QueueUnit {
	return &v1alpha1.QueueUnit{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.QueueUnitSpec{
			Priority: &priority,
		},
	}
}
