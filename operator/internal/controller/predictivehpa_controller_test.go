package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	autoscalingv1alpha1 "github.com/kust1q/predictive-hpa-operator/api/v1"
)

var _ = Describe("PredictiveHPA Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		predictivehpa := &autoscalingv1alpha1.PredictiveHPA{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind PredictiveHPA")
			err := k8sClient.Get(ctx, typeNamespacedName, predictivehpa)
			if err != nil && errors.IsNotFound(err) {
				resource := &autoscalingv1alpha1.PredictiveHPA{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: autoscalingv1alpha1.PredictiveHPASpec{
						MaxReplicas:      5,
						MetricsQuery:     "mock_query",
						PrometheusURL:    "http://mock-prometheus",
						PredictorAddress: "mock-address",
						IntervalSeconds:  60,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &autoscalingv1alpha1.PredictiveHPA{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance PredictiveHPA")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &PredictiveHPAReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				metricsProvider: &defaultMetricsProvider{},
				predictorClient: &defaultPredictorClient{},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
