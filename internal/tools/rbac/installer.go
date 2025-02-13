package rbac

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type RBACInstaller struct {
	DynamicClient dynamic.Interface
}

func (i *RBACInstaller) CreateRBAC(rbac *RBAC) error {
	// Install ClusterRole
	if rbac.ClusterRole != nil {
		// Check if ClusterRole already exists
		_, err := i.GetClusterRole(context.Background(), rbac.ClusterRole.Name)
		if err != nil {
			// Create ClusterRole
			_, err = i.CreateClusterRole(context.Background(), rbac.ClusterRole)
			if err != nil {
				return err
			}
		}
	}

	// Install Role
	if rbac.Namespaced != nil {
		for _, ns := range rbac.Namespaced {
			// Install Role
			if ns.Role != nil {
				// Check if Role already exists
				_, err := i.GetRole(context.Background(), ns.Role.Name, ns.Role.Namespace)
				if err != nil {
					// Create Role
					_, err = i.CreateRole(context.Background(), ns.Role)
					if err != nil {
						return err
					}
				}
			}

			// Install RoleBinding
			if ns.RoleBinding != nil {
				// Check if RoleBinding already exists
				_, err := i.GetRoleBinding(context.Background(), ns.RoleBinding.Name, ns.RoleBinding.Namespace)
				if err != nil {
					// Create RoleBinding
					_, err = i.CreateRoleBinding(context.Background(), ns.RoleBinding)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	// Install ClusterRoleBinding
	if rbac.ClusterRoleBinding != nil {
		// Check if ClusterRoleBinding already exists
		_, err := i.GetClusterRoleBinding(context.Background(), rbac.ClusterRoleBinding.Name)
		if err != nil {
			// Create ClusterRoleBinding
			_, err = i.CreateClusterRoleBinding(context.Background(), rbac.ClusterRoleBinding)
			if err != nil {
				return err
			}
		}
	}

	return nil

}

func (i *RBACInstaller) GetRole(ctx context.Context, name, namespace string) (*rbacv1.Role, error) {
	res, err := i.DynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "roles",
		},
	).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Role: %w", err)
	}

	role := &rbacv1.Role{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, role)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to Role: %w", err)
	}

	return role, nil
}

func (i *RBACInstaller) CreateRole(ctx context.Context, role *rbacv1.Role) (*rbacv1.Role, error) {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(role)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Role to unstructured: %w", err)
	}

	res, err := i.DynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "roles",
		},
	).Namespace(role.Namespace).Create(ctx, &unstructured.Unstructured{Object: m}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create Role: %w", err)
	}

	role = &rbacv1.Role{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, role)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to Role: %w", err)
	}

	return role, nil
}

func (i *RBACInstaller) GetRoleBinding(ctx context.Context, name, namespace string) (*rbacv1.RoleBinding, error) {
	res, err := i.DynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "rolebindings",
		},
	).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get RoleBinding: %w", err)
	}

	roleBinding := &rbacv1.RoleBinding{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, roleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to RoleBinding: %w", err)
	}

	return roleBinding, nil
}

func (i *RBACInstaller) CreateRoleBinding(ctx context.Context, roleBinding *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(roleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to convert RoleBinding to unstructured: %w", err)
	}

	res, err := i.DynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "rolebindings",
		},
	).Namespace(roleBinding.Namespace).Create(ctx, &unstructured.Unstructured{Object: m}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create RoleBinding: %w", err)
	}

	roleBinding = &rbacv1.RoleBinding{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, roleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to RoleBinding: %w", err)
	}

	return roleBinding, nil
}

func (i *RBACInstaller) GetClusterRole(ctx context.Context, name string) (*rbacv1.ClusterRole, error) {
	res, err := i.DynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "clusterroles",
		},
	).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ClusterRole: %w", err)
	}

	clusterRole := &rbacv1.ClusterRole{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, clusterRole)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to ClusterRole: %w", err)
	}

	return clusterRole, nil
}

func (i *RBACInstaller) CreateClusterRole(ctx context.Context, clusterRole *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(clusterRole)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ClusterRole to unstructured: %w", err)
	}

	res, err := i.DynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "clusterroles",
		},
	).Create(ctx, &unstructured.Unstructured{Object: m}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create ClusterRole: %w", err)
	}

	clusterRole = &rbacv1.ClusterRole{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, clusterRole)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to ClusterRole: %w", err)
	}

	return clusterRole, nil
}

func (i *RBACInstaller) GetClusterRoleBinding(ctx context.Context, name string) (*rbacv1.ClusterRoleBinding, error) {
	res, err := i.DynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "clusterrolebindings",
		},
	).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ClusterRoleBinding: %w", err)
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, clusterRoleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to ClusterRoleBinding: %w", err)
	}

	return clusterRoleBinding, nil
}

func (i *RBACInstaller) CreateClusterRoleBinding(ctx context.Context, clusterRoleBinding *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(clusterRoleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ClusterRoleBinding to unstructured: %w", err)
	}

	res, err := i.DynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "clusterrolebindings",
		},
	).Create(ctx, &unstructured.Unstructured{Object: m}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create ClusterRoleBinding: %w", err)
	}

	clusterRoleBinding = &rbacv1.ClusterRoleBinding{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, clusterRoleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to ClusterRoleBinding: %w", err)
	}

	return clusterRoleBinding, nil
}
