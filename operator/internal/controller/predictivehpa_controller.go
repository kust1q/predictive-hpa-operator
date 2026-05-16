package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	autoscalingv1alpha1 "github.com/kust1q/predictive-hpa-operator/api/v1"
	pb "github.com/kust1q/predictive-hpa-operator/api/v1/predictor"
)

const (
	timeoutPrometheusQuery = 10 * time.Second
	timeoutPredictorCall   = 10 * time.Second
	requeueAfterError      = 30 * time.Second
)

type metricsProvider interface {
	queryPrometheus(ctx context.Context, phpa *autoscalingv1alpha1.PredictiveHPA) ([]*pb.DataPoint, error)
}

type predictorClient interface {
	callPredictor(ctx context.Context, phpa *autoscalingv1alpha1.PredictiveHPA, dataPoints []*pb.DataPoint) (int32, error)
}

// PredictiveHPAReconciler reconciles a PredictiveHPA object.
type PredictiveHPAReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	metricsProvider metricsProvider
	predictorClient predictorClient
}

type defaultMetricsProvider struct{}

func (d *defaultMetricsProvider) queryPrometheus(ctx context.Context, phpa *autoscalingv1alpha1.PredictiveHPA) ([]*pb.DataPoint, error) {
	apiClient, err := api.NewClient(api.Config{
		Address: phpa.Spec.PrometheusURL,
	})
	if err != nil {
		return nil, err
	}

	v1api := v1.NewAPI(apiClient)
	ctx, cancel := context.WithTimeout(ctx, timeoutPrometheusQuery)
	defer cancel()

	end := time.Now()
	start := end.Add(-1 * time.Hour)

	val, _, err := v1api.QueryRange(ctx, phpa.Spec.MetricsQuery, v1.Range{
		Start: start,
		End:   end,
		Step:  time.Minute,
	})
	if err != nil {
		return nil, err
	}

	matrix, ok := val.(model.Matrix)
	if !ok || len(matrix) == 0 {
		return nil, fmt.Errorf("no metrics found for query: %s", phpa.Spec.MetricsQuery)
	}

	var dataPoints []*pb.DataPoint
	for _, sample := range matrix[0].Values {
		dataPoints = append(dataPoints, &pb.DataPoint{
			Timestamp: sample.Timestamp.Unix(),
			Value:     float64(sample.Value),
		})
	}

	return dataPoints, nil
}

type defaultPredictorClient struct{}

func (d *defaultPredictorClient) callPredictor(ctx context.Context, phpa *autoscalingv1alpha1.PredictiveHPA, dataPoints []*pb.DataPoint) (int32, error) {
	conn, err := grpc.NewClient(phpa.Spec.PredictorAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return 0, err
	}
	defer func() { _ = conn.Close() }()

	grpcClient := pb.NewPredictorClient(conn)
	ctx, cancel := context.WithTimeout(ctx, timeoutPredictorCall)
	defer cancel()

	resp, err := grpcClient.Predict(ctx, &pb.PredictionRequest{
		DataPoints:             dataPoints,
		ForecastHorizonSeconds: phpa.Spec.IntervalSeconds,
	})
	if err != nil {
		return 0, err
	}

	return resp.PredictedReplicas, nil
}

// +kubebuilder:rbac:groups=autoscaling.predictive-hpa.io,resources=predictivehpas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling.predictive-hpa.io,resources=predictivehpas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=autoscaling.predictive-hpa.io,resources=predictivehpas/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments/scale,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop.
func (r *PredictiveHPAReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	phpa := &autoscalingv1alpha1.PredictiveHPA{}
	err := r.Get(ctx, req.NamespacedName, phpa)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("Reconciling PredictiveHPA", "name", phpa.Name)

	dataPoints, err := r.metricsProvider.queryPrometheus(ctx, phpa)
	if err != nil {
		log.Error(err, "Failed to query Prometheus")
		return ctrl.Result{RequeueAfter: requeueAfterError}, nil
	}

	predictedReplicas, err := r.predictorClient.callPredictor(ctx, phpa, dataPoints)
	if err != nil {
		log.Error(err, "Failed to call predictor service")
		return ctrl.Result{RequeueAfter: requeueAfterError}, nil
	}

	desiredReplicas := predictedReplicas
	if phpa.Spec.MinReplicas != nil && desiredReplicas < *phpa.Spec.MinReplicas {
		desiredReplicas = *phpa.Spec.MinReplicas
	}
	if desiredReplicas > phpa.Spec.MaxReplicas {
		desiredReplicas = phpa.Spec.MaxReplicas
	}

	targetName := phpa.Spec.ScaleTargetRef.Name
	deployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Namespace: phpa.Namespace, Name: targetName}, deployment)
	if err != nil {
		log.Error(err, "Failed to get target Deployment", "name", targetName)
		return ctrl.Result{RequeueAfter: requeueAfterError}, nil
	}

	currentReplicas := *deployment.Spec.Replicas
	if currentReplicas != desiredReplicas {
		log.Info("Scaling deployment", "name", targetName, "from", currentReplicas, "to", desiredReplicas)

		scale := &autoscalingv1.Scale{}
		err = r.SubResource("scale").Get(ctx, deployment, scale)
		if err != nil {
			return ctrl.Result{}, err
		}

		scale.Spec.Replicas = desiredReplicas
		err = r.SubResource("scale").Update(ctx, deployment, client.WithSubResourceBody(scale))
		if err != nil {
			log.Error(err, "Failed to update scale subresource")
			return ctrl.Result{}, err
		}

		phpa.Status.LastScaleTime = &metav1.Time{Time: time.Now()}
	}

	phpa.Status.CurrentReplicas = currentReplicas
	phpa.Status.DesiredReplicas = desiredReplicas
	phpa.Status.LastPrediction = &predictedReplicas

	if err := r.Status().Update(ctx, phpa); err != nil {
		log.Error(err, "Failed to update PredictiveHPA status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Duration(phpa.Spec.IntervalSeconds) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PredictiveHPAReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.metricsProvider == nil {
		r.metricsProvider = &defaultMetricsProvider{}
	}
	if r.predictorClient == nil {
		r.predictorClient = &defaultPredictorClient{}
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingv1alpha1.PredictiveHPA{}).
		Named("predictivehpa").
		Complete(r)
}
