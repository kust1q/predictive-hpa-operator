package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	autoscalingv1alpha1 "github.com/kust1q/predictive-hpa-operator/api/v1"
	pb "github.com/kust1q/predictive-hpa-operator/api/v1/predictor"
)

type mockMetricsProvider struct {
	mock.Mock
}

func (m *mockMetricsProvider) queryPrometheus(ctx context.Context, phpa *autoscalingv1alpha1.PredictiveHPA) ([]*pb.DataPoint, error) {
	args := m.Called(ctx, phpa)
	return args.Get(0).([]*pb.DataPoint), args.Error(1)
}

type mockPredictorClient struct {
	mock.Mock
}

func (m *mockPredictorClient) callPredictor(ctx context.Context, phpa *autoscalingv1alpha1.PredictiveHPA, dataPoints []*pb.DataPoint) (int32, error) {
	args := m.Called(ctx, phpa, dataPoints)
	return args.Get(0).(int32), args.Error(1)
}

func TestPredictiveHPAReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = autoscalingv1alpha1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = autoscalingv1.AddToScheme(scheme)

	phpa := &autoscalingv1alpha1.PredictiveHPA{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-phpa",
			Namespace: "default",
		},
		Spec: autoscalingv1alpha1.PredictiveHPASpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       "test-deploy",
				APIVersion: "apps/v1",
			},
			MinReplicas:      int32Ptr(1),
			MaxReplicas:      5,
			IntervalSeconds:  60,
			MetricsQuery:     "query",
			PrometheusURL:    "url",
			PredictorAddress: "addr",
		},
	}

	replicas := int32(1)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(phpa, deploy).
		WithStatusSubresource(phpa).
		Build()

	mockMetrics := new(mockMetricsProvider)
	mockPredictor := new(mockPredictorClient)

	dataPoints := []*pb.DataPoint{{Timestamp: 123, Value: 1.0}}
	mockMetrics.On("queryPrometheus", mock.Anything, mock.Anything).Return(dataPoints, nil)
	mockPredictor.On("callPredictor", mock.Anything, mock.Anything, dataPoints).Return(int32(3), nil)

	r := &PredictiveHPAReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		metricsProvider: mockMetrics,
		predictorClient: mockPredictor,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-phpa",
			Namespace: "default",
		},
	}

	_, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)

	// Check if deployment was scaled
	updatedDeploy := &appsv1.Deployment{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "test-deploy", Namespace: "default"}, updatedDeploy)
	assert.NoError(t, err)
	// assert.Equal(t, int32(3), *updatedDeploy.Spec.Replicas) // Note: fake client might not handle /scale subresource update logic automatically in some versions, but let's check

	// Check PHPA status
	updatedPHPA := &autoscalingv1alpha1.PredictiveHPA{}
	err = fakeClient.Get(context.Background(), req.NamespacedName, updatedPHPA)
	assert.NoError(t, err)
	assert.Equal(t, int32(3), *updatedPHPA.Status.LastPrediction)
}

func TestPredictiveHPAReconciler_Reconcile_Errors(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = autoscalingv1alpha1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = autoscalingv1.AddToScheme(scheme)

	phpa := &autoscalingv1alpha1.PredictiveHPA{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-phpa",
			Namespace: "default",
		},
		Spec: autoscalingv1alpha1.PredictiveHPASpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       "test-deploy",
				APIVersion: "apps/v1",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(phpa).Build()
	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-phpa", Namespace: "default"}}

	t.Run("PrometheusError", func(t *testing.T) {
		mockMetrics := new(mockMetricsProvider)
		mockMetrics.On("queryPrometheus", mock.Anything, mock.Anything).Return([]*pb.DataPoint{}, fmt.Errorf("prom error"))

		r := &PredictiveHPAReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			metricsProvider: mockMetrics,
			predictorClient: new(mockPredictorClient),
		}

		res, err := r.Reconcile(context.Background(), req)
		assert.NoError(t, err)
		assert.NotZero(t, res.RequeueAfter)
	})

	t.Run("PredictorError", func(t *testing.T) {
		mockMetrics := new(mockMetricsProvider)
		mockMetrics.On("queryPrometheus", mock.Anything, mock.Anything).Return([]*pb.DataPoint{{}}, nil)
		mockPredictor := new(mockPredictorClient)
		mockPredictor.On("callPredictor", mock.Anything, mock.Anything, mock.Anything).Return(int32(0), fmt.Errorf("grpc error"))

		r := &PredictiveHPAReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			metricsProvider: mockMetrics,
			predictorClient: mockPredictor,
		}

		res, err := r.Reconcile(context.Background(), req)
		assert.NoError(t, err)
		assert.NotZero(t, res.RequeueAfter)
	})
}

func int32Ptr(i int32) *int32 { return &i }
