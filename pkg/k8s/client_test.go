package k8s_test

import (
	"context"
	"sort"
	"testing"

	k8spkg "github.com/srosignoli/faultline/pkg/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/fake"
)

func assertEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", msg, got, want)
	}
}

const testNamespace = "faultline"

func newClient(t *testing.T) (*k8spkg.Client, *fake.Clientset) {
	t.Helper()
	cs := fake.NewSimpleClientset()
	return k8spkg.NewWithClientset(cs, testNamespace), cs
}

func TestCreateSimulator_ConfigMap(t *testing.T) {
	t.Parallel()
	client, cs := newClient(t)
	ctx := context.Background()

	err := client.CreateSimulator(ctx, "test-sim", "dump data", "rules data")
	if err != nil {
		t.Fatalf("CreateSimulator: %v", err)
	}

	cm, err := cs.CoreV1().ConfigMaps(testNamespace).Get(ctx, "test-sim-config", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get configmap: %v", err)
	}

	assertEqual(t, cm.Name, "test-sim-config", "configmap name")
	assertEqual(t, cm.Namespace, testNamespace, "configmap namespace")
	assertEqual(t, cm.Labels["app"], "faultline-worker", "label app")
	assertEqual(t, cm.Labels["instance"], "test-sim", "label instance")
	assertEqual(t, cm.Data["dump.txt"], "dump data", "dump.txt data")
	assertEqual(t, cm.Data["rules.yaml"], "rules data", "rules.yaml data")
}

func TestCreateSimulator_Deployment(t *testing.T) {
	t.Parallel()
	client, cs := newClient(t)
	ctx := context.Background()

	err := client.CreateSimulator(ctx, "test-sim", "dump", "rules")
	if err != nil {
		t.Fatalf("CreateSimulator: %v", err)
	}

	dep, err := cs.AppsV1().Deployments(testNamespace).Get(ctx, "test-sim-deployment", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}

	assertEqual(t, dep.Name, "test-sim-deployment", "deployment name")
	assertEqual(t, dep.Namespace, testNamespace, "deployment namespace")
	assertEqual(t, dep.Labels["app"], "faultline-worker", "deployment label app")
	assertEqual(t, dep.Labels["instance"], "test-sim", "deployment label instance")

	// Pod template labels
	assertEqual(t, dep.Spec.Template.Labels["app"], "faultline-worker", "pod template label app")
	assertEqual(t, dep.Spec.Template.Labels["instance"], "test-sim", "pod template label instance")

	if len(dep.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("no containers in deployment")
	}
	container := dep.Spec.Template.Spec.Containers[0]
	assertEqual(t, container.Image, "faultline-worker:latest", "container image")

	if len(container.Ports) == 0 {
		t.Fatal("no ports in container")
	}
	assertEqual(t, container.Ports[0].ContainerPort, int32(8080), "container port")

	if len(container.VolumeMounts) == 0 {
		t.Fatal("no volume mounts")
	}
	assertEqual(t, container.VolumeMounts[0].MountPath, "/etc/faultline/", "volume mount path")

	if len(dep.Spec.Template.Spec.Volumes) == 0 {
		t.Fatal("no volumes")
	}
	vol := dep.Spec.Template.Spec.Volumes[0]
	if vol.ConfigMap == nil {
		t.Fatal("volume has no ConfigMap source")
	}
	assertEqual(t, vol.ConfigMap.Name, "test-sim-config", "volume configmap ref")
}

func TestCreateSimulator_Service(t *testing.T) {
	t.Parallel()
	client, cs := newClient(t)
	ctx := context.Background()

	err := client.CreateSimulator(ctx, "test-sim", "dump", "rules")
	if err != nil {
		t.Fatalf("CreateSimulator: %v", err)
	}

	svc, err := cs.CoreV1().Services(testNamespace).Get(ctx, "test-sim-svc", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get service: %v", err)
	}

	assertEqual(t, svc.Name, "test-sim-svc", "service name")
	if len(svc.Spec.Ports) == 0 {
		t.Fatal("no ports in service")
	}
	assertEqual(t, svc.Spec.Ports[0].Port, int32(80), "service port")
	assertEqual(t, svc.Spec.Ports[0].TargetPort.IntVal, int32(8080), "service target port")
	assertEqual(t, svc.Spec.Selector["app"], "faultline-worker", "service selector app")
	assertEqual(t, svc.Spec.Selector["instance"], "test-sim", "service selector instance")
}

func TestListSimulators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		instances []string // simulator names to create
		rulesYAML string   // rules payload passed to CreateSimulator
		want      []string // expected instance names
	}{
		{name: "empty", instances: nil, want: []string{}},
		{name: "one", instances: []string{"alpha"}, rulesYAML: "rules-alpha", want: []string{"alpha"}},
		{name: "two", instances: []string{"alpha", "beta"}, rulesYAML: "rules-multi", want: []string{"alpha", "beta"}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, _ := newClient(t)
			ctx := context.Background()

			for _, inst := range tc.instances {
				if err := client.CreateSimulator(ctx, inst, "dump", tc.rulesYAML); err != nil {
					t.Fatalf("CreateSimulator(%s): %v", inst, err)
				}
			}

			got, err := client.ListSimulators(ctx)
			if err != nil {
				t.Fatalf("ListSimulators: %v", err)
			}

			sort.Slice(got, func(i, j int) bool { return got[i].Name < got[j].Name })
			wantNames := make([]string, len(tc.want))
			copy(wantNames, tc.want)
			sort.Strings(wantNames)

			if len(got) != len(wantNames) {
				t.Fatalf("got %v, want %v", got, wantNames)
			}
			for i := range got {
				assertEqual(t, got[i].Name, wantNames[i], "instance name")
				if tc.rulesYAML != "" {
					assertEqual(t, got[i].RulesYAML, tc.rulesYAML, "rules yaml for "+got[i].Name)
				}
			}
		})
	}
}

func TestDeleteSimulator(t *testing.T) {
	t.Parallel()
	client, cs := newClient(t)
	ctx := context.Background()

	if err := client.CreateSimulator(ctx, "doomed", "dump", "rules"); err != nil {
		t.Fatalf("CreateSimulator: %v", err)
	}

	if err := client.DeleteSimulator(ctx, "doomed"); err != nil {
		t.Fatalf("DeleteSimulator: %v", err)
	}

	_, err := cs.CoreV1().ConfigMaps(testNamespace).Get(ctx, "doomed-config", metav1.GetOptions{})
	if !k8serrors.IsNotFound(err) {
		t.Errorf("expected configmap to be deleted, got: %v", err)
	}

	_, err = cs.AppsV1().Deployments(testNamespace).Get(ctx, "doomed-deployment", metav1.GetOptions{})
	if !k8serrors.IsNotFound(err) {
		t.Errorf("expected deployment to be deleted, got: %v", err)
	}

	_, err = cs.CoreV1().Services(testNamespace).Get(ctx, "doomed-svc", metav1.GetOptions{})
	if !k8serrors.IsNotFound(err) {
		t.Errorf("expected service to be deleted, got: %v", err)
	}
}

func TestDeleteSimulator_idempotent(t *testing.T) {
	t.Parallel()
	client, _ := newClient(t)
	ctx := context.Background()

	if err := client.DeleteSimulator(ctx, "never-existed"); err != nil {
		t.Errorf("expected nil for non-existent simulator, got: %v", err)
	}
}
