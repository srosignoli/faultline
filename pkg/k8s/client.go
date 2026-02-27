package k8s

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const Namespace = "faultline"

// SimulatorInfo is returned by ListSimulators and carries the raw rules YAML
// alongside the simulator name so callers can parse mutation rules without an
// extra round-trip.
type SimulatorInfo struct {
	Name      string
	RulesYAML string
}

// Client wraps a Kubernetes clientset scoped to a single namespace.
type Client struct {
	cs        kubernetes.Interface
	namespace string
}

// New builds a Client. If kubeconfig is empty it tries in-cluster config,
// otherwise it loads the given kubeconfig path.
func New(kubeconfig string) (*Client, error) {
	var cfg *rest.Config
	var err error
	if kubeconfig == "" {
		cfg, err = rest.InClusterConfig()
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return nil, fmt.Errorf("k8s config: %w", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("k8s clientset: %w", err)
	}
	return &Client{cs: cs, namespace: Namespace}, nil
}

// NewWithClientset builds a Client with an injected clientset (for testing).
func NewWithClientset(cs kubernetes.Interface, namespace string) *Client {
	return &Client{cs: cs, namespace: namespace}
}

func commonLabels(name string) map[string]string {
	return map[string]string{
		"app":      "faultline-worker",
		"instance": name,
	}
}

// CreateSimulator creates the ConfigMap, Deployment, and Service for a simulator.
func (c *Client) CreateSimulator(ctx context.Context, name, dumpPayload, rulesPayload string) error {
	if err := c.createConfigMap(ctx, name, dumpPayload, rulesPayload); err != nil {
		return err
	}
	if err := c.createDeployment(ctx, name); err != nil {
		return err
	}
	return c.createService(ctx, name)
}

func (c *Client) createConfigMap(ctx context.Context, name, dumpPayload, rulesPayload string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-config",
			Namespace: c.namespace,
			Labels:    commonLabels(name),
		},
		Data: map[string]string{
			"dump.txt":   dumpPayload,
			"rules.yaml": rulesPayload,
		},
	}
	_, err := c.cs.CoreV1().ConfigMaps(c.namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create configmap %s: %w", cm.Name, err)
	}
	return nil
}

func (c *Client) createDeployment(ctx context.Context, name string) error {
	replicas := int32(1)
	labels := commonLabels(name)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-deployment",
			Namespace: c.namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "worker",
							Image: "faultline-worker:latest",
							Ports: []corev1.ContainerPort{
								{ContainerPort: 8080},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config-volume",
									MountPath: "/etc/faultline/",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: name + "-config",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := c.cs.AppsV1().Deployments(c.namespace).Create(ctx, dep, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create deployment %s: %w", dep.Name, err)
	}
	return nil
}

func (c *Client) createService(ctx context.Context, name string) error {
	labels := commonLabels(name)
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-svc",
			Namespace: c.namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "web",
					Port:       80,
					TargetPort: intstr.FromInt32(8080),
				},
			},
		},
	}
	_, err := c.cs.CoreV1().Services(c.namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create service %s: %w", svc.Name, err)
	}
	return nil
}

// ListSimulators returns info about all running simulators, including their
// raw rules YAML fetched from the associated ConfigMap.
func (c *Client) ListSimulators(ctx context.Context) ([]SimulatorInfo, error) {
	list, err := c.cs.AppsV1().Deployments(c.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=faultline-worker",
	})
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	infos := make([]SimulatorInfo, 0, len(list.Items))
	for _, dep := range list.Items {
		inst, ok := dep.Labels["instance"]
		if !ok {
			continue
		}
		rulesYAML := ""
		cm, err := c.cs.CoreV1().ConfigMaps(c.namespace).Get(ctx, inst+"-config", metav1.GetOptions{})
		if err == nil {
			rulesYAML = cm.Data["rules.yaml"]
		}
		infos = append(infos, SimulatorInfo{Name: inst, RulesYAML: rulesYAML})
	}
	return infos, nil
}

// DeleteSimulator removes the ConfigMap, Deployment, and Service for a simulator.
// NotFound errors are silently ignored (idempotent).
func (c *Client) DeleteSimulator(ctx context.Context, name string) error {
	if err := c.cs.CoreV1().ConfigMaps(c.namespace).Delete(ctx, name+"-config", metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("delete configmap: %w", err)
	}
	if err := c.cs.AppsV1().Deployments(c.namespace).Delete(ctx, name+"-deployment", metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("delete deployment: %w", err)
	}
	if err := c.cs.CoreV1().Services(c.namespace).Delete(ctx, name+"-svc", metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("delete service: %w", err)
	}
	return nil
}
