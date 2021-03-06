package fluentbit

import (
	"bytes"
	"strconv"
	"text/template"

	loggingv1alpha1 "github.com/log_management/logging-operator/pkg/apis/logging/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// func validateConfigMap(cr *loggingv1alpha1.LogManagement) error {
// 	for _, in := range cr.Spec.Inputs {
// 		present := false
// 		for _, par := range cr.Spec.Parsers {
// 			if in.Parser == par.Name {
// 				present = true
// 				break
// 			}
// 		}
// 		if !present {
// 			return errors.New("Parser not found")
// 		}
// 	}
// 	return nil
// }

func generateEnvironmentVariables() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "FLUENT_ELASTICSEARCH_HOST",
			Value: "10.98.50.241",
		},
		{
			Name:  "FLUENT_ELASTICSEARCH_PORT",
			Value: "30240",
		},
	}
}

func generateVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "varlog",
			MountPath: "/var/log",
		},
		{
			Name:      "fluent-bit-config",
			MountPath: "/fluent-bit/etc/",
		},
		{
			Name:      "varlibcontainers",
			ReadOnly:  true,
			MountPath: "/var/lib/docker/containers",
		},
	}
}

func generateVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "varlog",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/log",
				},
			},
		},
		{
			Name: "fluent-bit-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "fluent-bit-config",
					},
				},
			},
		},
		{
			Name: "varlibcontainers",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/docker/containers",
				},
			},
		},
	}
}

// CreateDaemonSet generates the FluentBit DS
func CreateDaemonSet(cr *loggingv1alpha1.LogManagement, serviceAccount *corev1.ServiceAccount) *appsv1.DaemonSet {
	labels := map[string]string{
		"k8s-app": "fluent-bit-logging",
	}
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluent-bit",
			Namespace: cr.ObjectMeta.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:            "fluent-bit",
							Image:           "fluent/fluent-bit:0.14.6",
							ImagePullPolicy: "Always",
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									ContainerPort: 2020,
								},
							},
							Env:          generateEnvironmentVariables(),
							VolumeMounts: generateVolumeMounts(),
						},
					},
					Volumes:            generateVolumes(),
					ServiceAccountName: serviceAccount.Name,
					Tolerations: []corev1.Toleration{
						corev1.Toleration{
							Key:      "node-role.kubernetes.io/master",
							Operator: "Exists",
							Effect:   "NoSchedule",
						},
					},
				},
			},
		},
	}
}

// CreateConfigMap - generate config map
func CreateConfigMap(cr *loggingv1alpha1.LogManagement) *corev1.ConfigMap {

	templateInput := TemplateInput{
		FluentBitLogFile: cr.Spec.FluentBitLogFile,
		K8sMetadata:      cr.Spec.K8sMetadata,
	}
	for index, i := range cr.Spec.Watch {
		var parsers []InputParser
		var deployments []Deployment
		for _, p := range i.Parsers {
			parsers = append(parsers, InputParser{Name: p.Name})
		}
		if len(i.Deployments) == 0 {
			deployments = append(deployments, Deployment{Name: ""})
		}
		for _, d := range i.Deployments {
			deployments = append(deployments, Deployment{Name: d.Name})
		}
		cr.Spec.Watch[index].Tag = "tag_" + strconv.Itoa(index)
		templateInput.Inputs = append(templateInput.Inputs, Input{
			Namespace:   i.Namespace,
			Deployments: deployments,
			Tag:         cr.Spec.Watch[index].Tag,
			Parsers:     parsers,
		})
	}

	for _, i := range cr.Spec.Parsers {
		templateInput.Parsers = append(templateInput.Parsers, Parser{Name: i.Name, Regex: i.Regex})
	}
	configMap, _ := generateConfig(templateInput, configmapTemplate)
	parserMap, _ := generateConfig(templateInput, parsersTemplate)

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: "fluent-bit-config",
			Labels: map[string]string{
				"k8s-app": "fluent-bit",
			},
			Namespace: cr.ObjectMeta.Namespace,
		},

		Data: map[string]string{
			"fluent-bit.conf": *configMap,
			"parsers.conf":    *parserMap,
		},
	}
}

// TemplateInput defines the input template placeholder
type TemplateInput struct {
	FluentBitLogFile string
	K8sMetadata      bool
	Inputs           []Input
	Parsers          []Parser
}

// Input defines the structure of input placeholder
type Input struct {
	Namespace   string
	Deployments []Deployment
	Tag         string
	Parsers     []InputParser
}

// Parser defines structure of Parsers
type Parser struct {
	Name  string
	Regex string
}

// InputParser defines input parser structure
type InputParser struct {
	Name string
}

// Deployment spec
type Deployment struct {
	Name string
}

func generateConfig(input TemplateInput, templateFile string) (*string, error) {
	output := new(bytes.Buffer)
	tmpl, err := template.New("config").Parse(templateFile)
	if err != nil {
		return nil, err
	}
	err = tmpl.Execute(output, input)
	if err != nil {
		return nil, err
	}
	outputString := output.String()
	return &outputString, nil
}
