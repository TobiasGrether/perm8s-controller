package v1alpha1

import (
    v4 "k8s.io/api/rbac/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AuthenticationSource string

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:noStatus
// +kubeBuilder:scope=Cluster
// +kubeBuilder:resource:scope=Cluster
// +kubeBuilder:resource.scope=Cluster
// +kubebuilder:printcolumn:JSONPath=".spec.description",name=Description,type=string
// +kubebuilder:field:scope=Cluster
type Group struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec GroupSpec `json:"spec"`
}

type GroupSpec struct {
    DisplayName  string          `json:"displayName"`
    Description  string          `json:"description"`
    Permissions  []v4.PolicyRule `json:"permissions"ƒ`
    Namespaces   []string        `json:"namespaces"`
    ClusterGroup bool            `json:"clusterGroup"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:noStatus
// +kubebuilder:printcolumn:JSONPath=".spec.type",name=Source Type,type=string
type SynchronisationSource struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec SynchronisationSourceSpec `json:"spec"`
}

type SynchronisationSourceSpec struct {
    // +kubebuilder:validation:Enum=authentik;ldap
    Type      string                             `json:"type"`
    // +kubebuilder:validation:Optional
    Authentik *AuthentikSynchronisationSourceSpec `json:"authentik"`
    // GroupMappings should be a map internal group identifier => Kubernetes Group Name
    // This is useful when your IdP or SyncSource returns some kind of UUID for the groups,
    // but you want human-readable named groups in the cluster
    GroupMappings map[string]string `json:"groupMappings"`
}

type AuthentikSynchronisationSourceSpec struct {
    URL        string `json:"url"`
    Scheme     string `json:"scheme"`
    SecretName string `json:"secretName"`
    // RequiredGroups is a list where a user only gets considered for this data source once they are a member of at least one of these groups
    // Leaving this array empty will autopass all users
    RequiredGroups []string `json:"requiredGroups"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SynchronisationSourceList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata"`

    Items []SynchronisationSource `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:noStatus
// +kubebuilder:printcolumn:JSONPath=".spec.authenticationSource",name=Authentication Source,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.displayName",name=Display Name,type=string
type User struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec UserSpec `json:"spec"`
}

type UserSpec struct {
    DisplayName          string   `json:"displayName"`
    AuthenticationSource string   `json:"authenticationSource"`
    GroupMemberships     []string `json:"groupMemberships"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type GroupList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata"`

    Items []Group `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type UserList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata"`

    Items []User `json:"items"`
}
