package utils

import (
	"app-operator/api/v1"
	"bytes"
	"html/template"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func parseTemplate(templateName string, app *v1.App) []byte {
	tmpl, err := template.ParseFiles("controllers/template/" + templateName + ".yml")
	if err != nil {
		panic(err)
	}
	b := new(bytes.Buffer)
	err = tmpl.Execute(b, app)
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}

func getLabels(mObj ...map[string]string) map[string]string {
	newObj := map[string]string{}
	for _, m := range mObj {
		for k, v := range m {
			newObj[k] = v
		}
	}
	return newObj
}

func NewDeployment(app *v1.App) *appsv1.Deployment {
	d := &appsv1.Deployment{}
	err := yaml.Unmarshal(parseTemplate("deployment", app), d)
	if err != nil {
		panic(err)
	}

	labels := d.Labels
	appLabels := app.Labels
	d.Labels = getLabels(labels, appLabels)
	d.Spec.Template.Labels = getLabels(labels, appLabels)

	d.Spec.Template.Spec.Affinity = app.Spec.DeployConfig.Affinity
	d.Spec.Template.Spec.Volumes = app.Spec.DeployConfig.Volumes
	d.Spec.Template.Spec.Containers[0].Env = app.Spec.DeployConfig.Envs
	d.Spec.Template.Spec.Containers[0].LivenessProbe = app.Spec.DeployConfig.LivenessProbe
	d.Spec.Template.Spec.Containers[0].ReadinessProbe = app.Spec.DeployConfig.ReadinessProbe
	d.Spec.Template.Spec.Containers[0].Resources = app.Spec.DeployConfig.Resources
	d.Spec.Template.Spec.Containers[0].VolumeMounts = app.Spec.DeployConfig.VolumeMounts

	return d
}

func NewIngress(app *v1.App) *netv1.Ingress {
	i := &netv1.Ingress{}
	err := yaml.Unmarshal(parseTemplate("ingress", app), i)
	if err != nil {
		panic(err)
	}

	labels := i.Labels
	appLabels := app.Labels
	i.Labels = getLabels(labels, appLabels)

	if app.Spec.IngressConfig.IngressHost != "" {
		i.Spec.Rules[0].Host = app.Spec.IngressConfig.IngressHost
	}
	if app.Spec.IngressConfig.IngressClass != "" {
		i.Spec.IngressClassName = &app.Spec.IngressConfig.IngressClass
	}
	return i
}

func NewService(app *v1.App) *corev1.Service {
	s := &corev1.Service{}
	err := yaml.Unmarshal(parseTemplate("service", app), s)
	if err != nil {
		panic(err)
	}

	labels := s.Labels
	appLabels := app.Labels
	s.Labels = getLabels(labels, appLabels)

	if app.Spec.ServiceConfig.ServiceType != "" {
		s.Spec.Type = app.Spec.ServiceConfig.ServiceType
	}
	s.Spec.Ports = app.Spec.ServiceConfig.ServicePorts

	return s
}
