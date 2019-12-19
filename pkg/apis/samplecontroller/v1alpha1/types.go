/*
Copyright 2017 The Kubernetes Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VM is a specification for a VM resource
type VM struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VMSpec   `json:"spec"`
	Status VMStatus `json:"status"`
}

// VMSpec is the spec for a VM resource
type VMSpec struct {
	VMName	string `json:"vmname"`
}

// VMStatus is the status for a VM resource
type VMStatus struct {
	VMId			string `json:"vmId"`
	CPUUtilization	string `json:"cpuUtilization"`
	OpStatus 		string `json:"opStatus"`
	Msg             string `json:"msg"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VMList is a list of VM resources
type VMList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VM `json:"items"`
}

const (
	RunningState string = "RunningState"
	NewState string = ""
	FailedState string = "FailedState"
	UnknownState string = "UnknownState"
)