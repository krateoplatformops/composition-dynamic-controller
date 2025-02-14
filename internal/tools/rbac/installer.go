package rbac

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"

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

func NewRBACInstaller(cli dynamic.Interface) *RBACInstaller {
	return &RBACInstaller{DynamicClient: cli}
}

func (i *RBACInstaller) ApplyRBAC(rbac *RBAC) error {
	if rbac.ClusterRole != nil {
		_, err := i.ApplyClusterRole(context.Background(), rbac.ClusterRole)
		if err != nil {
			return fmt.Errorf("failed to Apply ClusterRole: %w", err)
		}
	}
	if rbac.ClusterRoleBinding != nil {
		_, err := i.ApplyClusterRoleBinding(context.Background(), rbac.ClusterRoleBinding)
		if err != nil {
			return fmt.Errorf("failed to Apply ClusterRoleBinding: %w", err)
		}
	}
	for _, ns := range rbac.Namespaced {
		if ns.Role != nil {
			_, err := i.ApplyRole(context.Background(), ns.Role)
			if err != nil {
				return fmt.Errorf("failed to Apply Role: %w", err)
			}
		}
		if ns.RoleBinding != nil {
			_, err := i.ApplyRoleBinding(context.Background(), ns.RoleBinding)
			if err != nil {
				return fmt.Errorf("failed to Apply RoleBinding: %w", err)
			}
		}
	}
	return nil

}

func (i *RBACInstaller) ApplyRole(ctx context.Context, role *rbacv1.Role) (*rbacv1.Role, error) {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(role)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Role to unstructured: %w", err)
	}
	u := &unstructured.Unstructured{Object: m}

	cli := i.DynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "roles",
		},
	).Namespace(role.Namespace)

	if _, err = cli.Get(ctx, role.Name, metav1.GetOptions{}); errors.IsNotFound(err) {
		res, err := cli.Create(ctx, u, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to Create Role: %w", err)
		}

		role = &rbacv1.Role{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, role)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to Role: %w", err)
		}

		return role, nil
	}

	res, err := cli.Update(ctx, u, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to Update Role: %w", err)
	}

	role = &rbacv1.Role{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, role)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to Role: %w", err)
	}

	return role, nil
}

func (i *RBACInstaller) ApplyRoleBinding(ctx context.Context, roleBinding *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(roleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to convert RoleBinding to unstructured: %w", err)
	}
	u := &unstructured.Unstructured{Object: m}

	cli := i.DynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "rolebindings",
		},
	).Namespace(roleBinding.Namespace)

	if _, err := cli.Get(ctx, roleBinding.Name, metav1.GetOptions{}); errors.IsNotFound(err) {
		res, err := cli.Create(ctx, u, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to Create RoleBinding: %w", err)
		}

		roleBinding = &rbacv1.RoleBinding{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, roleBinding)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to RoleBinding: %w", err)
		}

		return roleBinding, nil
	}

	res, err := cli.Update(ctx, u, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to Update RoleBinding: %w", err)
	}

	roleBinding = &rbacv1.RoleBinding{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, roleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to RoleBinding: %w", err)
	}

	return roleBinding, nil
}

func (i *RBACInstaller) ApplyClusterRole(ctx context.Context, clusterRole *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(clusterRole)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ClusterRole to unstructured: %w", err)
	}
	u := &unstructured.Unstructured{Object: m}

	cli := i.DynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "clusterroles",
		},
	)

	if _, err := cli.Get(ctx, clusterRole.Name, metav1.GetOptions{}); errors.IsNotFound(err) {
		res, err := cli.Create(ctx, u, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to Create ClusterRole: %w", err)
		}

		clusterRole = &rbacv1.ClusterRole{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, clusterRole)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to ClusterRole: %w", err)
		}

		return clusterRole, nil
	}

	res, err := cli.Update(ctx, u, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to Update ClusterRole: %w", err)
	}

	clusterRole = &rbacv1.ClusterRole{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, clusterRole)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to ClusterRole: %w", err)
	}
	return clusterRole, nil
}

func (i *RBACInstaller) ApplyClusterRoleBinding(ctx context.Context, clusterRoleBinding *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(clusterRoleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ClusterRoleBinding to unstructured: %w", err)
	}
	u := &unstructured.Unstructured{Object: m}

	cli := i.DynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "clusterrolebindings",
		},
	)

	if _, err := cli.Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{}); errors.IsNotFound(err) {
		res, err := cli.Create(ctx, u, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to Create ClusterRoleBinding: %w", err)
		}

		clusterRoleBinding = &rbacv1.ClusterRoleBinding{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, clusterRoleBinding)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to ClusterRoleBinding: %w", err)
		}

		return clusterRoleBinding, nil
	}

	res, err := cli.Update(ctx, u, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to Update ClusterRoleBinding: %w", err)
	}

	clusterRoleBinding = &rbacv1.ClusterRoleBinding{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(res.Object, clusterRoleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to ClusterRoleBinding: %w", err)
	}

	return clusterRoleBinding, nil
}
