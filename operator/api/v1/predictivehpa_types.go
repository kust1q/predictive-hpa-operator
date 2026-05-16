package v1

import (
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PredictiveHPASpec defines the desired state of PredictiveHPA.
type PredictiveHPASpec struct {
	// ScaleTargetRef points to the target resource to scale.
	ScaleTargetRef autoscalingv1.CrossVersionObjectReference `json:"scaleTargetRef"`

	// MinReplicas is the lower limit for the number of replicas.
	// +kubebuilder:validation:Minimum=1
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// MaxReplicas is the upper limit for the number of replicas.
	// +kubebuilder:validation:Minimum=1
	MaxReplicas int32 `json:"maxReplicas"`

	// MetricsQuery is the Prometheus query used to fetch historical metrics.
	// +kubebuilder:validation:MinLength=1
	MetricsQuery string `json:"metricsQuery"`

	// PrometheusURL is the URL of the Prometheus server.
	// +kubebuilder:validation:Pattern=`^https?://.*`
	PrometheusURL string `json:"prometheusURL"`

	// PredictorAddress is the gRPC address of the Python ML service.
	// +kubebuilder:validation:MinLength=1
	PredictorAddress string `json:"predictorAddress"`

	// IntervalSeconds is the interval in seconds between prediction cycles.
	// +kubebuilder:validation:Minimum=10
	// +optional
	// +kubebuilder:default=60
	IntervalSeconds int32 `json:"intervalSeconds,omitempty"`
}

// PredictiveHPAStatus defines the observed state of PredictiveHPA.
type PredictiveHPAStatus struct {
	// CurrentReplicas is the current number of replicas.
	// +optional
	CurrentReplicas int32 `json:"currentReplicas,omitempty"`

	// DesiredReplicas is the desired number of replicas.
	// +optional
	DesiredReplicas int32 `json:"desiredReplicas,omitempty"`

	// LastPrediction is the value of the last prediction received.
	// +optional
	LastPrediction *int32 `json:"lastPrediction,omitempty"`

	// LastScaleTime is the last time the PredictiveHPA scaled the target.
	// +optional
	LastScaleTime *metav1.Time `json:"lastScaleTime,omitempty"`

	// Conditions represent the current state of the PredictiveHPA resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PredictiveHPA is the Schema for the predictivehpas API.
type PredictiveHPA struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec PredictiveHPASpec `json:"spec"`

	// +optional
	Status PredictiveHPAStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PredictiveHPAList contains a list of PredictiveHPA.
type PredictiveHPAList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PredictiveHPA `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PredictiveHPA{}, &PredictiveHPAList{})
}
