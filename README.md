# Perm8s
Perm8s is a solution to allow for easier user access management for external users to a Kubernetes Cluster and its resources. <br/>
Perm8s basically provides easy-to-understand abstractions over the Kubernetes RBAC API, which are fully managed by this controller.

## Installation

## Usage
### User
The base of Perm8s is the `User` CRD. A User hereby represents a ServiceAccount in a Namespace (usually the namespace that you deployed Perm8s in) as well as a Kubeconfig secret that is generated automatically for that user.


A user can be the member of one or more groups, and these groups define what access that user will have. If a user has no groups assigned to him, he will have no access to any resources inside the cluster.

A User is defined as follows:
```yaml
kind: User
apiVersion: perm8s.tobiasgrether.com/v1alpha1
metadata:
  name: test-user
spec:
  displayName: "My Test User"
  authenticationSource: local
  groupMemberships: ["test-group", "list-namespaces"]
```

Perm8s will automatically handle the following tasks for each user:
- Create a ServiceAccount for the User
- Create a Kubernetes Authentication Secret for that User (which can be used with f.e. kubectl)
- Create the necessary RoleBindings and ClusterRoleBindings for each group that the user is a member of.

### Groups
Groups allow you to simplify permission management by specifying that a uniquely named group of users all have the same permissions.
A user can be a member of multiple groups, which will cause the permissions to be combined.

A group is defined as follows:
```yaml
kind: Group
apiVersion: perm8s.tobiasgrether.com/v1alpha1
metadata:
  name: test-group
spec:
  displayName: test-group
  description: "My little group for testing"
  clusterGroup: false
  permissions:
    - apiGroups: [""]
      resources: ["secrets"]
      verbs: ["get", "watch", "list"]
    - apiGroups: [""]
      resources: ["pods", "pods/log", "pods/eviction"]
      verbs: ["list", "watch", "get", "delete"]
  namespaces: ["test-namespace"]
```

There are two types of groups: **namespaced groups** and **cluster groups**.

**Namespaced Groups** will provide the given permissions to all members only in every namespace that is explicitly defined in the `namespaces` property. This is what most people usually use. Only give people some access to some resources in select namespaces. Example: only give the `intern` group access to the pods in the staging namespace.

**Cluster Groups** will provide the given permissions to all members across the entire cluster. This will ignore any other Namespaced Groups. A user that has permissions to list and get secrets through a Cluster Group will be able to do that in **every namespace**. So be careful with Cluster Groups.

### Synchronisation
Perm8s also allows you to sync users from an external source. This system is easily adaptable to basically anything that can provide a list of users and groups. As an example, Authentik is implemented, but it can be expanded to support other technologies like LDAP.

Perm8s will periodically fetch a list of all users that are supposed to be in the cluster according to the remote synchronisation source, and will make sure that the cluster state matches the state of the sync source. This will synchronise groups for existing users, create new users for users that did not have an account yet, and automatically delete any account that is not in the provided allowed user list anymore.

Every `AuthenticationSource` has a `type` field, which defines the kind of data backend that is behind the source.

Here is an example configuration with authentik:
```yaml
kind: SynchronisationSource
apiVersion: perm8s.tobiasgrether.com/v1alpha1
metadata:
  name: acme

spec:
  type: authentik
  authentik:
    scheme: https
    url: sso.acme.com
    secretName: authentik-token
    requiredGroups: ["8c71f59c-0855-4ba8-8fa9-afcacadd5250"]
  groupMappings:
    "8c71f59c-0855-4ba8-8fa9-afcacadd5250": "developer"
```

This will only consider users as valid that have the given group, and groupMappings will convert internal identifiers into readable group names that are mapped to the `Group` CRD that was mentioned earlier.