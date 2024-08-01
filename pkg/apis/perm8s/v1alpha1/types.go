package v1alpha1

import (
    v4 "k8s.io/api/rbac/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
// +kubeBuilder:scope=Cluster
// +kubeBuilder:resource:scope=Cluster
// +kubeBuilder:resource.scope=Cluster
// +kubebuilder:printcolumn:JSONPath=".spec.URL",name=Root URL,type=string
// +kubebuilder:field:scope=Cluster
type AuthentikSynchronisationSource struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec AuthentikSynchronisationSourceSpec `json:"spec"`
}

type AuthentikSynchronisationSourceSpec struct {
    URL        string `json:"url"`
    Scheme     string `json:"scheme"`
    SecretName string `json:"secretName"`
    // RequiredGroups is a list where a user only gets considered for this data source once they are a member of at least one of these groups
    // Leaving this array empty will autopass all users
    RequiredGroups []string `json:"requiredGroups"`
    // GroupMappings should be a map authentikGroupID => Kubernetes Group Name
    GroupMappings map[string]string `json:"groupMappings"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AuthentikSynchronisationSourceList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata"`

    Items []AuthentikSynchronisationSource `json:"items"`
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
