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
type SelfHealingCheckInstanceSpec struct {
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
type SelfHealingCheckInstanceStatus struct {
	NodeName   string                             `json:"nodeName"`
	State      string                             `json:"state,omitempty"`
	Conditions SelfHealingCheckInstanceConditions `json:"conditions,omitempty"`
}

type SelfHealingCheckInstanceConditions struct {
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Message            string      `json:"message,omitempty"`
	Reason             string      `json:"reason,omitempty"`
}

type SelfHealingActionInstanceSpec struct {
	ID             int32       `json:"id,omitempty"`
	NodeCode       string      `json:"nodeCode,omitempty"`
	NodeName       string      `json:"nodeName,omitempty"`
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
type SelfHealingActionInstanceStatus struct {
	State      string                                `json:"state,omitempty"`
	Conditions []SelfHealingActionInstanceConditions `json:"conditions,omitempty"`
}

type SelfHealingActionInstanceConditions struct {
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Message            string      `json:"message,omitempty"`
	Reason             string      `json:"reason,omitempty"`
}

type SelfHealingTaskInstanceSpec struct {
	SelfHealingCheckSpec  SelfHealingCheckSpec  `json:"selfHealingCheckSpec"`
	SelfHealingActionSpec SelfHealingActionSpec `json:"selfHealingActionSpec"`
}

type SelfHealingTaskInstanceStatus struct {
	SelfHealingActionStatus SelfHealingActionInstanceStatus `json:"selfHealingActionStatus"`
	SelfHealingCheckStatus  SelfHealingCheckInstanceStatus  `json:"selfHealingCheckStatus"`
}

// +genclient
// +kubebuilder:object:root=true

// SelfHealingTask is the Schema for the selfhealingtasks API
type SelfHealingTaskInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SelfHealingTaskInstanceSpec   `json:"spec,omitempty"`
	Status SelfHealingTaskInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SelfHealingTaskList contains a list of SelfHealingTask
type SelfHealingTaskInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SelfHealingTaskInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SelfHealingTaskInstance{}, &SelfHealingTaskInstanceList{})
}
