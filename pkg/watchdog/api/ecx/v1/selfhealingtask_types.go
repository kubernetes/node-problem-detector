/*


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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SelfHealingTaskSpec defines the desired state of SelfHealingTask
type SelfHealingCheckSpec struct {
	ID             int32       `json:"id,omitempty"`
	NodeCode       string      `json:"nodeCode,omitempty"`
	NodeLabels     string      `json:"nodeLabels,omitempty"`
	Mark           string      `json:"mark,omitempty"`
	LogPath        string      `json:"logPath,omitempty"`
	Pattern        string      `json:"pattern,omitempty"`
	Script         string      `json:"script,omitempty"`
	Intervals      int32       `json:"intervals,omitempty"`
	Version        string      `json:"version,omitempty"`
	ItemName       string      `json:"itemName,omitempty"`
	MonitorType    int32       `json:"monitorType,omitempty"`
	CreateUserName string      `json:"createUserName,omitempty"`
	UpdateUserName string      `json:"updateUserName,omitempty"`
	LastedVersion  string      `json:"lastedVersion,omitempty"`
	CreateTime     metav1.Time `json:"createTime,omitempty"`
	ModifyTime     metav1.Time `json:"modifyTime,omitempty"`
}

// SelfHealingTaskStatus defines the observed state of SelfHealingTask
type SelfHealingCheckStatus struct {
	State      string                      `json:"state,omitempty"`
	Conditions []SelfHealingCheckCondition `json:"conditions,omitempty"`
}

type SelfHealingCheckCondition struct {
	InstanceName       string      `json:"instanceName,omitempty"`
	Namespace          string      `json:"namespace"`
	NodeName           string      `json:"nodeName"`
	State              string      `json:"state,omitempty"`
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Message            string      `json:"message,omitempty"`
	Reason             string      `json:"reason,omitempty"`
}

type SelfHealingActionSpec struct {
	ID             int32       `json:"id,omitempty"`
	NodeCode       string      `json:"nodeCode,omitempty"`
	NodeLabels     string      `json:"nodeLabels,omitempty"`
	Mark           string      `json:"mark,omitempty"`
	LogPath        string      `json:"logPath,omitempty"`
	Pattern        string      `json:"pattern,omitempty"`
	Script         string      `json:"script,omitempty"`
	Intervals      int32       `json:"intervals,omitempty"`
	Version        string      `json:"version,omitempty"`
	ItemName       string      `json:"itemName,omitempty"`
	MonitorType    int32       `json:"monitorType,omitempty"`
	CreateUserName string      `json:"createUserName,omitempty"`
	UpdateUserName string      `json:"updateUserName,omitempty"`
	LastedVersion  string      `json:"lastedVersion,omitempty"`
	CreateTime     metav1.Time `json:"createTime,omitempty"`
	ModifyTime     metav1.Time `json:"modifyTime,omitempty"`
}

// SelfHealingTaskStatus defines the observed state of SelfHealingTask
type SelfHealingActionStatus struct {
	State      string                        `json:"state,omitempty"`
	Conditions []SelfHealingActionConditions `json:"conditions,omitempty"`
}

type SelfHealingActionConditions struct {
	InstanceName       string      `json:"InstanceName"`
	Namespace          string      `json:"namespace"`
	NodeName           string      `json:"nodeName"`
	State              string      `json:"state,omitempty"`
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Message            string      `json:"message,omitempty"`
	Reason             string      `json:"reason,omitempty"`
}

type SelfHealingTaskSpec struct {
	SelfHealingCheckSpec  SelfHealingCheckSpec  `json:"selfHealingCheckSpec"`
	SelfHealingActionSpec SelfHealingActionSpec `json:"selfHealingActionSpec"`
}

type SelfHealingTaskStatus struct {
	SelfHealingActionStatus SelfHealingActionStatus `json:"selfHealingActionStatus"`
	SelfHealingCheckStatus  SelfHealingCheckStatus  `json:"selfHealingCheckStatus"`
}

// +genclient
// +kubebuilder:object:root=true

// SelfHealingTask is the Schema for the selfhealingtasks API
type SelfHealingTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SelfHealingTaskSpec   `json:"spec,omitempty"`
	Status SelfHealingTaskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SelfHealingTaskList contains a list of SelfHealingTask
type SelfHealingTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SelfHealingTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SelfHealingTask{}, &SelfHealingTaskList{})
}
