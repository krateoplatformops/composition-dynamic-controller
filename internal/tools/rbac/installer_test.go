package rbac

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

func TestRBACInstaller_CreateRBAC(t *testing.T) {
	scheme := runtime.NewScheme()
	rbacv1.AddToScheme(scheme)

	client := fake.NewSimpleDynamicClient(scheme)
	installer := &RBACInstaller{DynamicClient: client}

	rbac := &RBAC{
		ClusterRole: &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "test-clusterrole"},
		},
		Namespaced: map[string]Namespaced{
			"default": {
				Role: &rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{Name: "test-role", Namespace: "default"},
				},
				RoleBinding: &rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{Name: "test-rolebinding", Namespace: "default"},
				},
			},
		},
		ClusterRoleBinding: &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "test-clusterrolebinding"},
		},
	}

	err := installer.CreateRBAC(rbac)
	if err != nil {
		t.Fatalf("CreateRBAC failed: %v", err)
	}

	// Verify ClusterRole
	_, err = installer.GetClusterRole(context.Background(), "test-clusterrole")
	if err != nil {
		t.Fatalf("GetClusterRole failed: %v", err)
	}

	// Verify Role
	_, err = installer.GetRole(context.Background(), "test-role", "default")
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}

	// Verify RoleBinding
	_, err = installer.GetRoleBinding(context.Background(), "test-rolebinding", "default")
	if err != nil {
		t.Fatalf("GetRoleBinding failed: %v", err)
	}

	// Verify ClusterRoleBinding
	_, err = installer.GetClusterRoleBinding(context.Background(), "test-clusterrolebinding")
	if err != nil {
		t.Fatalf("GetClusterRoleBinding failed: %v", err)
	}
}

func TestRBACInstaller_CreateRole(t *testing.T) {
	scheme := runtime.NewScheme()
	rbacv1.AddToScheme(scheme)

	client := fake.NewSimpleDynamicClient(scheme)
	installer := &RBACInstaller{DynamicClient: client}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: "test-role", Namespace: "default"},
	}

	createdRole, err := installer.CreateRole(context.Background(), role)
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	if createdRole.Name != role.Name {
		t.Errorf("expected role name %s, got %s", role.Name, createdRole.Name)
	}

	// Test creating an existing role
	_, err = installer.CreateRole(context.Background(), role)
	if err == nil {
		t.Fatalf("expected error when creating existing role, got nil")
	}
}

func TestRBACInstaller_CreateRoleBinding(t *testing.T) {
	scheme := runtime.NewScheme()
	rbacv1.AddToScheme(scheme)

	client := fake.NewSimpleDynamicClient(scheme)
	installer := &RBACInstaller{DynamicClient: client}

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "test-rolebinding", Namespace: "default"},
	}

	createdRoleBinding, err := installer.CreateRoleBinding(context.Background(), roleBinding)
	if err != nil {
		t.Fatalf("CreateRoleBinding failed: %v", err)
	}

	if createdRoleBinding.Name != roleBinding.Name {
		t.Errorf("expected rolebinding name %s, got %s", roleBinding.Name, createdRoleBinding.Name)
	}

	// Test creating an existing role binding
	_, err = installer.CreateRoleBinding(context.Background(), roleBinding)
	if err == nil {
		t.Fatalf("expected error when creating existing role binding, got nil")
	}
}

func TestRBACInstaller_CreateClusterRole(t *testing.T) {
	scheme := runtime.NewScheme()
	rbacv1.AddToScheme(scheme)

	client := fake.NewSimpleDynamicClient(scheme)
	installer := &RBACInstaller{DynamicClient: client}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: "test-clusterrole"},
	}

	createdClusterRole, err := installer.CreateClusterRole(context.Background(), clusterRole)
	if err != nil {
		t.Fatalf("CreateClusterRole failed: %v", err)
	}

	if createdClusterRole.Name != clusterRole.Name {
		t.Errorf("expected clusterrole name %s, got %s", clusterRole.Name, createdClusterRole.Name)
	}

	// Test creating an existing cluster role
	_, err = installer.CreateClusterRole(context.Background(), clusterRole)
	if err == nil {
		t.Fatalf("expected error when creating existing cluster role, got nil")
	}
}

func TestRBACInstaller_CreateClusterRoleBinding(t *testing.T) {
	scheme := runtime.NewScheme()
	rbacv1.AddToScheme(scheme)

	client := fake.NewSimpleDynamicClient(scheme)
	installer := &RBACInstaller{DynamicClient: client}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "test-clusterrolebinding"},
	}

	createdClusterRoleBinding, err := installer.CreateClusterRoleBinding(context.Background(), clusterRoleBinding)
	if err != nil {
		t.Fatalf("CreateClusterRoleBinding failed: %v", err)
	}

	if createdClusterRoleBinding.Name != clusterRoleBinding.Name {
		t.Errorf("expected clusterrolebinding name %s, got %s", clusterRoleBinding.Name, createdClusterRoleBinding.Name)
	}

	// Test creating an existing cluster role binding
	_, err = installer.CreateClusterRoleBinding(context.Background(), clusterRoleBinding)
	if err == nil {
		t.Fatalf("expected error when creating existing cluster role binding, got nil")
	}
}

func TestRBACInstaller_GetRole(t *testing.T) {
	scheme := runtime.NewScheme()
	rbacv1.AddToScheme(scheme)

	client := fake.NewSimpleDynamicClient(scheme)
	installer := &RBACInstaller{DynamicClient: client}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: "test-role", Namespace: "default"},
	}

	_, err := installer.CreateRole(context.Background(), role)
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	retrievedRole, err := installer.GetRole(context.Background(), "test-role", "default")
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}

	if retrievedRole.Name != role.Name {
		t.Errorf("expected role name %s, got %s", role.Name, retrievedRole.Name)
	}
}

func TestRBACInstaller_GetRoleBinding(t *testing.T) {
	scheme := runtime.NewScheme()
	rbacv1.AddToScheme(scheme)

	client := fake.NewSimpleDynamicClient(scheme)
	installer := &RBACInstaller{DynamicClient: client}

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "test-rolebinding", Namespace: "default"},
	}

	_, err := installer.CreateRoleBinding(context.Background(), roleBinding)
	if err != nil {
		t.Fatalf("CreateRoleBinding failed: %v", err)
	}

	retrievedRoleBinding, err := installer.GetRoleBinding(context.Background(), "test-rolebinding", "default")
	if err != nil {
		t.Fatalf("GetRoleBinding failed: %v", err)
	}

	if retrievedRoleBinding.Name != roleBinding.Name {
		t.Errorf("expected rolebinding name %s, got %s", roleBinding.Name, retrievedRoleBinding.Name)
	}
}

func TestRBACInstaller_GetClusterRole(t *testing.T) {
	scheme := runtime.NewScheme()
	rbacv1.AddToScheme(scheme)

	client := fake.NewSimpleDynamicClient(scheme)
	installer := &RBACInstaller{DynamicClient: client}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: "test-clusterrole"},
	}

	_, err := installer.CreateClusterRole(context.Background(), clusterRole)
	if err != nil {
		t.Fatalf("CreateClusterRole failed: %v", err)
	}

	retrievedClusterRole, err := installer.GetClusterRole(context.Background(), "test-clusterrole")
	if err != nil {
		t.Fatalf("GetClusterRole failed: %v", err)
	}

	if retrievedClusterRole.Name != clusterRole.Name {
		t.Errorf("expected clusterrole name %s, got %s", clusterRole.Name, retrievedClusterRole.Name)
	}
}

func TestRBACInstaller_GetClusterRoleBinding(t *testing.T) {
	scheme := runtime.NewScheme()
	rbacv1.AddToScheme(scheme)

	client := fake.NewSimpleDynamicClient(scheme)
	installer := &RBACInstaller{DynamicClient: client}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "test-clusterrolebinding"},
	}

	_, err := installer.CreateClusterRoleBinding(context.Background(), clusterRoleBinding)
	if err != nil {
		t.Fatalf("CreateClusterRoleBinding failed: %v", err)
	}

	retrievedClusterRoleBinding, err := installer.GetClusterRoleBinding(context.Background(), "test-clusterrolebinding")
	if err != nil {
		t.Fatalf("GetClusterRoleBinding failed: %v", err)
	}

	if retrievedClusterRoleBinding.Name != clusterRoleBinding.Name {
		t.Errorf("expected clusterrolebinding name %s, got %s", clusterRoleBinding.Name, retrievedClusterRoleBinding.Name)
	}
}
