/*
Copyright The CloudNativePG Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// When you add a new label or annotation, please make sure that you also update the
// publicly visible documentation, namely the `docs/src/labels_annotations.md` file
const (
	// ClusterLabelName is the name of cluster which the backup CR belongs to
	ClusterLabelName = "cnpg.io/cluster"

	// JobRoleLabelName is the name of the label containing the purpose of the executed job
	JobRoleLabelName = "cnpg.io/jobRole"

	// PvcRoleLabelName is the name of the label containing the purpose of the pvc
	PvcRoleLabelName = "cnpg.io/pvcRole"

	// PodRoleLabelName is the name of the label containing the podRole value
	PodRoleLabelName = "cnpg.io/podRole"

	// InstanceNameLabelName is the name of the label containing the instance name
	InstanceNameLabelName = "cnpg.io/instanceName"

	// BackupNameLabelName is the name of the label containing the backup id
	BackupNameLabelName = "cnpg.io/backupName"

	// OperatorVersionAnnotationName is the name of the annotation containing
	// the version of the operator that generated a certain object
	OperatorVersionAnnotationName = "cnpg.io/operatorVersion"

	// AppArmorAnnotationPrefix will be the name of the AppArmor profile to apply
	// This is required for Azure but can be set in other environments
	AppArmorAnnotationPrefix = "container.apparmor.security.beta.kubernetes.io"

	// ReconciliationLoopAnnotationName is the name of the annotation controlling
	// the status of the reconciliation loop for the cluster
	ReconciliationLoopAnnotationName = "cnpg.io/reconciliationLoop"

	// HibernateClusterManifestAnnotationName contains the hibernated cluster manifest
	// Deprecated. Replaced by: ClusterManifestAnnotationName. This annotation is
	// kept for backward compatibility
	HibernateClusterManifestAnnotationName = "cnpg.io/hibernateClusterManifest"

	// HibernatePgControlDataAnnotationName contains the pg_controldata output of the hibernated cluster
	// Deprecated. Replaced by: PgControldataAnnotationName. This annotation is
	// kept for backward compatibility
	HibernatePgControlDataAnnotationName = "cnpg.io/hibernatePgControlData"

	// PodEnvHashAnnotationName is the name of the annotation containing the podEnvHash value
	// Deprecated: the PodSpec annotation covers the environment drift. This annotation is
	// kept for backward compatibility
	PodEnvHashAnnotationName = "cnpg.io/podEnvHash"

	// PodSpecAnnotationName is the name of the annotation with the PodSpec derived from the cluster
	PodSpecAnnotationName = "cnpg.io/podSpec"

	// ClusterManifestAnnotationName contains the cluster manifest
	ClusterManifestAnnotationName = "cnpg.io/clusterManifest"

	// PgControldataAnnotationName contains the pg_controldata output of the cluster
	PgControldataAnnotationName = "cnpg.io/pgControldata"

	// skipEmptyWalArchiveCheck turns off the checks that ensure that the WAL archive is empty before writing data
	skipEmptyWalArchiveCheck = "cnpg.io/skipEmptyWalArchiveCheck"
)

type annotationStatus string

const (
	annotationStatusDisabled annotationStatus = "disabled"
	annotationStatusEnabled  annotationStatus = "enabled"
)

// PodRole describes the Role of a given pod
type PodRole string

const (
	// PodRoleInstance the label value indicating an instance
	PodRoleInstance PodRole = "instance"
)

// PVCRole describes the role of a PVC
type PVCRole string

const (
	// PVCRolePgData is a PVC used for storing PG_DATA
	PVCRolePgData PVCRole = "PG_DATA"
	// PVCRolePgWal is a PVC used for storing PG_WAL
	PVCRolePgWal PVCRole = "PG_WAL"
)

// LabelClusterName labels the object with the cluster name
func LabelClusterName(object *metav1.ObjectMeta, name string) {
	if object.Labels == nil {
		object.Labels = make(map[string]string)
	}

	object.Labels[ClusterLabelName] = name
}

// SetOperatorVersion set inside a certain object metadata the annotation
// containing the version of the operator that generated the object
func SetOperatorVersion(object *metav1.ObjectMeta, version string) {
	if object.Annotations == nil {
		object.Annotations = make(map[string]string)
	}

	object.Annotations[OperatorVersionAnnotationName] = version
}

// InheritanceController controls if a label or an annotation should be
// inherited
type InheritanceController interface {
	// IsAnnotationInherited checks if a certain annotation should be
	// inherited
	IsAnnotationInherited(name string) bool

	// IsLabelInherited checks if a certain label should be
	// inherited
	IsLabelInherited(name string) bool
}

// InheritAnnotations puts into the object metadata the passed annotations if
// the annotations are supposed to be inherited. The passed configuration is
// used to determine whenever a certain annotation is inherited or not
func InheritAnnotations(
	object *metav1.ObjectMeta,
	annotations map[string]string,
	fixedAnnotations map[string]string,
	controller InheritanceController,
) {
	if object.Annotations == nil {
		object.Annotations = make(map[string]string)
	}

	for key, value := range fixedAnnotations {
		object.Annotations[key] = value
	}

	for key, value := range annotations {
		if controller.IsAnnotationInherited(key) {
			object.Annotations[key] = value
		}
	}
}

// InheritLabels puts into the object metadata the passed labels if
// the labels are supposed to be inherited. The passed configuration is
// used to determine whenever a certain label is inherited or not
func InheritLabels(
	object *metav1.ObjectMeta,
	labels map[string]string,
	fixedLabels map[string]string,
	controller InheritanceController,
) {
	if object.Labels == nil {
		object.Labels = make(map[string]string)
	}

	for key, value := range fixedLabels {
		object.Labels[key] = value
	}

	for key, value := range labels {
		if controller.IsLabelInherited(key) {
			object.Labels[key] = value
		}
	}
}

func getAnnotationAppArmor(spec *corev1.PodSpec, annotations map[string]string) map[string]string {
	containsContainerWithName := func(name string, containers ...corev1.Container) bool {
		for _, container := range containers {
			if container.Name == name {
				return true
			}
		}

		return false
	}

	appArmorAnnotations := make(map[string]string)
	for annotation, value := range annotations {
		if strings.HasPrefix(annotation, AppArmorAnnotationPrefix) {
			appArmorSplit := strings.SplitN(annotation, "/", 2)
			if len(appArmorSplit) < 2 {
				continue
			}

			containerName := appArmorSplit[1]
			if containsContainerWithName(containerName, append(spec.Containers, spec.InitContainers...)...) {
				appArmorAnnotations[annotation] = value
			}
		}
	}
	return appArmorAnnotations
}

// IsAnnotationAppArmorPresent checks if one of the annotations is an AppArmor annotation
func IsAnnotationAppArmorPresent(spec *corev1.PodSpec, annotations map[string]string) bool {
	annotation := getAnnotationAppArmor(spec, annotations)
	return len(annotation) != 0
}

// IsAnnotationAppArmorPresentInObject checks if the AppArmor annotations are present or not in the given Object
func IsAnnotationAppArmorPresentInObject(
	object *metav1.ObjectMeta,
	spec *corev1.PodSpec,
	annotations map[string]string,
) bool {
	objAnnotations := getAnnotationAppArmor(spec, object.Annotations)
	appArmorAnnotations := getAnnotationAppArmor(spec, annotations)
	return reflect.DeepEqual(objAnnotations, appArmorAnnotations)
}

// AnnotateAppArmor adds an annotation to the pod
func AnnotateAppArmor(object *metav1.ObjectMeta, spec *corev1.PodSpec, annotations map[string]string) {
	if object.Annotations == nil {
		object.Annotations = make(map[string]string)
	}
	appArmorAnnotations := getAnnotationAppArmor(spec, annotations)
	for annotation, value := range appArmorAnnotations {
		object.Annotations[annotation] = value
	}
}

// IsReconciliationDisabled checks if the reconciliation loop is disabled on the given resource
func IsReconciliationDisabled(object *metav1.ObjectMeta) bool {
	return object.Annotations[ReconciliationLoopAnnotationName] == string(annotationStatusDisabled)
}

// IsEmptyWalArchiveCheckEnabled returns a boolean indicating if we should run the logic that checks if the WAL archive
// storage is empty
func IsEmptyWalArchiveCheckEnabled(object *metav1.ObjectMeta) bool {
	return object.Annotations[skipEmptyWalArchiveCheck] != string(annotationStatusEnabled)
}

// MergeMap transfers the content of a giver map to a receiver
func MergeMap(receiver, giver map[string]string) {
	for key, value := range giver {
		receiver[key] = value
	}
}
