package kibana

import (
	"github.com/log_management/logging-operator/cmd/manager/utils"
	loggingv1alpha1 "github.com/log_management/logging-operator/pkg/apis/logging/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func createEnvironmentVariables(esSpec *utils.ElasticSearchSpec) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "ELASTICSEARCH_URL",
			Value: esSpec.HTTPString + esSpec.CurrentHost + ":" + esSpec.CurrentPort,
		},
	}
}

// CreateKibanaDeployment - creates Kibana deployment
func CreateKibanaDeployment(cr *loggingv1alpha1.LogManagement, esSpec *utils.ElasticSearchSpec) *appsv1.Deployment {
	label := map[string]string{
		"app": "kibana",
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kibana",
			Labels:    label,
			Namespace: cr.ObjectMeta.Namespace,
		},

		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: label,
			},

			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: label,
				},

				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "kibana",
							Image: "docker.elastic.co/kibana/kibana:" + cr.Spec.ESKibanaVersion,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5601,
								},
							},
							Env: createEnvironmentVariables(esSpec),
						},
					},
				},
			},
		},
	}
}

// CreateKibanaService - generates kibana service
func CreateKibanaService(cr *loggingv1alpha1.LogManagement) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "kibana",
			Namespace: cr.ObjectMeta.Namespace,
		},

		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kibana",
			},
			Ports: []corev1.ServicePort{
				{
					Port: 5601,
					TargetPort: intstr.IntOrString{
						IntVal: int32(5601),
					},
				},
			},
			Type: "LoadBalancer",
		},
	}
}
