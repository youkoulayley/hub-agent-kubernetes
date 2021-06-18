package reviewer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/hub-agent/pkg/acp"
	"github.com/traefik/hub-agent/pkg/acp/admission/ingclass"
	"github.com/traefik/hub-agent/pkg/acp/admission/quota"
	"github.com/traefik/hub-agent/pkg/acp/basicauth"
	"github.com/traefik/hub-agent/pkg/acp/digestauth"
	"github.com/traefik/hub-agent/pkg/acp/jwt"
	traefikv1alpha1 "github.com/traefik/hub-agent/pkg/crd/api/traefik/v1alpha1"
	traefikkubemock "github.com/traefik/hub-agent/pkg/crd/generated/client/traefik/clientset/versioned/fake"
	admv1 "k8s.io/api/admission/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestTraefikIngress_HandleACPName(t *testing.T) {
	factory := func(policies PolicyGetter) reviewer {
		fwdAuthMdlwrs := NewFwdAuthMiddlewares("", policies, traefikkubemock.NewSimpleClientset().TraefikV1alpha1())

		return NewTraefikIngress(ingressClassesMock{}, fwdAuthMdlwrs, quota.New(999))
	}

	ingressHandleACPName(t, factory)
}

func TestTraefikIngress_CanReviewChecksKind(t *testing.T) {
	i := ingressClassesMock{
		getDefaultControllerFunc: func() (string, error) {
			return ingclass.ControllerTypeTraefik, nil
		},
	}

	tests := []struct {
		desc      string
		kind      metav1.GroupVersionKind
		canReview bool
	}{
		{
			desc: "can review networking.k8s.io v1 Ingresses",
			kind: metav1.GroupVersionKind{
				Group:   "networking.k8s.io",
				Version: "v1",
				Kind:    "Ingress",
			},
			canReview: true,
		},
		{
			desc: "can't review invalid networking.k8s.io Ingress version",
			kind: metav1.GroupVersionKind{
				Group:   "networking.k8s.io",
				Version: "invalid",
				Kind:    "Ingress",
			},
			canReview: false,
		},
		{
			desc: "can't review invalid networking.k8s.io Ingress group",
			kind: metav1.GroupVersionKind{
				Group:   "invalid",
				Version: "v1",
				Kind:    "Ingress",
			},
			canReview: false,
		},
		{
			desc: "can't review non Ingress networking.k8s.io v1 resources",
			kind: metav1.GroupVersionKind{
				Group:   "networking.k8s.io",
				Version: "v1",
				Kind:    "NetworkPolicy",
			},
			canReview: false,
		},
		{
			desc: "can review extensions v1beta1 Ingresses",
			kind: metav1.GroupVersionKind{
				Group:   "extensions",
				Version: "v1beta1",
				Kind:    "Ingress",
			},
			canReview: true,
		},
		{
			desc: "can't review invalid extensions Ingress version",
			kind: metav1.GroupVersionKind{
				Group:   "extensions",
				Version: "invalid",
				Kind:    "Ingress",
			},
			canReview: false,
		},
		{
			desc: "can't review invalid v1beta1 Ingress group",
			kind: metav1.GroupVersionKind{
				Group:   "invalid",
				Version: "v1beta1",
				Kind:    "Ingress",
			},
			canReview: false,
		},
		{
			desc: "can't review invalid extension v1beta1 resource",
			kind: metav1.GroupVersionKind{
				Group:   "extensions",
				Version: "v1beta1",
				Kind:    "Invalid",
			},
			canReview: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			policies := func(canonicalName string) *acp.Config {
				return nil
			}
			fwdAuthMdlwrs := NewFwdAuthMiddlewares("", policyGetterMock(policies), nil)
			review := NewTraefikIngress(i, fwdAuthMdlwrs, quota.New(999))

			var ing netv1.Ingress
			b, err := json.Marshal(ing)
			require.NoError(t, err)

			ar := admv1.AdmissionReview{
				Request: &admv1.AdmissionRequest{
					Kind: test.kind,
					Object: runtime.RawExtension{
						Raw: b,
					},
				},
			}

			ok, err := review.CanReview(ar)
			require.NoError(t, err)
			assert.Equal(t, test.canReview, ok)
		})
	}
}

func TestTraefikIngress_CanReviewChecksIngressClass(t *testing.T) {
	tests := []struct {
		desc                   string
		annotation             string
		spec                   string
		wrongDefaultController bool
		canReview              bool
	}{
		{
			desc:      "can review a valid resource",
			canReview: true,
		},
		{
			desc:                   "can't review if the default controller is not of the correct type",
			wrongDefaultController: true,
			canReview:              false,
		},
		{
			desc:       "can review if annotation is correct",
			annotation: "traefik",
			canReview:  true,
		},
		{
			desc:       "can review if using a custom ingress class (annotation)",
			annotation: "custom-traefik-ingress-class",
			canReview:  true,
		},
		{
			desc:       "can't review if using another annotation",
			annotation: "nginx",
			canReview:  false,
		},
		{
			desc:      "can review if using a custom ingress class (spec)",
			spec:      "custom-traefik-ingress-class",
			canReview: true,
		},
		{
			desc:      "can't review if using another controller",
			spec:      "nginx",
			canReview: false,
		},
		{
			desc:       "spec takes priority over annotation#1",
			annotation: "nginx",
			spec:       "custom-traefik-ingress-class",
			canReview:  true,
		},
		{
			desc:       "spec takes priority over annotation#2",
			annotation: "traefik",
			spec:       "nginx",
			canReview:  false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			i := ingressClassesMock{
				getControllerFunc: func(name string) string {
					if name == "custom-traefik-ingress-class" {
						return ingclass.ControllerTypeTraefik
					}
					return "nope"
				},
				getDefaultControllerFunc: func() (string, error) {
					if test.wrongDefaultController {
						return "nope", nil
					}
					return ingclass.ControllerTypeTraefik, nil
				},
			}

			policies := func(canonicalName string) *acp.Config {
				return nil
			}
			fwdAuthMdlwrs := NewFwdAuthMiddlewares("", policyGetterMock(policies), nil)
			review := NewTraefikIngress(i, fwdAuthMdlwrs, quota.New(999))

			ing := netv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": test.annotation,
					},
				},
				Spec: netv1.IngressSpec{
					IngressClassName: &test.spec,
				},
			}

			b, err := json.Marshal(ing)
			require.NoError(t, err)

			ar := admv1.AdmissionReview{
				Request: &admv1.AdmissionRequest{
					Kind: metav1.GroupVersionKind{
						Group:   "networking.k8s.io",
						Version: "v1",
						Kind:    "Ingress",
					},
					Object: runtime.RawExtension{
						Raw: b,
					},
				},
			}

			ok, err := review.CanReview(ar)
			require.NoError(t, err)
			assert.Equal(t, test.canReview, ok)
		})
	}
}

func TestTraefikIngress_ReviewAddsAuthentication(t *testing.T) {
	tests := []struct {
		desc                    string
		config                  *acp.Config
		oldIngAnno              map[string]string
		ingAnno                 map[string]string
		wantPatch               map[string]string
		wantAuthResponseHeaders []string
	}{
		{
			desc: "add JWT authentication",
			config: &acp.Config{JWT: &jwt.Config{
				ForwardHeaders: map[string]string{
					"fwdHeader": "claim",
				},
			}},
			oldIngAnno: map[string]string{
				AnnotationHubAuth:   "my-old-policy@test",
				"custom-annotation": "foobar",
				"traefik.ingress.kubernetes.io/router.middlewares": "test-zz-my-old-policy-test@kubernetescrd",
			},
			ingAnno: map[string]string{
				AnnotationHubAuth:   "my-policy@test",
				"custom-annotation": "foobar",
				"traefik.ingress.kubernetes.io/router.middlewares": "custom-middleware@kubernetescrd",
			},
			wantPatch: map[string]string{
				AnnotationHubAuth:   "my-policy@test",
				"custom-annotation": "foobar",
				"traefik.ingress.kubernetes.io/router.middlewares": "custom-middleware@kubernetescrd,test-zz-my-policy-test@kubernetescrd",
			},
			wantAuthResponseHeaders: []string{"fwdHeader"},
		},
		{
			desc: "add Basic authentication",
			config: &acp.Config{BasicAuth: &basicauth.Config{
				StripAuthorizationHeader: true,
				ForwardUsernameHeader:    "User",
			}},
			oldIngAnno: map[string]string{},
			ingAnno: map[string]string{
				AnnotationHubAuth:   "my-policy@test",
				"custom-annotation": "foobar",
				"traefik.ingress.kubernetes.io/router.middlewares": "custom-middleware@kubernetescrd",
			},
			wantPatch: map[string]string{
				AnnotationHubAuth:   "my-policy@test",
				"custom-annotation": "foobar",
				"traefik.ingress.kubernetes.io/router.middlewares": "custom-middleware@kubernetescrd,test-zz-my-policy-test@kubernetescrd",
			},
			wantAuthResponseHeaders: []string{"User", "Authorization"},
		},
		{
			desc: "add Digest authentication",
			config: &acp.Config{DigestAuth: &digestauth.Config{
				StripAuthorizationHeader: true,
				ForwardUsernameHeader:    "User",
			}},
			oldIngAnno: map[string]string{},
			ingAnno: map[string]string{
				AnnotationHubAuth:   "my-policy@test",
				"custom-annotation": "foobar",
				"traefik.ingress.kubernetes.io/router.middlewares": "custom-middleware@kubernetescrd",
			},
			wantPatch: map[string]string{
				AnnotationHubAuth:   "my-policy@test",
				"custom-annotation": "foobar",
				"traefik.ingress.kubernetes.io/router.middlewares": "custom-middleware@kubernetescrd,test-zz-my-policy-test@kubernetescrd",
			},
			wantAuthResponseHeaders: []string{"User", "Authorization"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			traefikClientSet := traefikkubemock.NewSimpleClientset()
			policies := func(canonicalName string) *acp.Config {
				return test.config
			}
			fwdAuthMdlwrs := NewFwdAuthMiddlewares("", policyGetterMock(policies), traefikClientSet.TraefikV1alpha1())
			rev := NewTraefikIngress(ingressClassesMock{}, fwdAuthMdlwrs, quota.New(999))

			oldIng := struct {
				Metadata metav1.ObjectMeta `json:"metadata"`
			}{
				Metadata: metav1.ObjectMeta{
					Name:        "name",
					Namespace:   "test",
					Annotations: test.oldIngAnno,
				},
			}
			oldB, err := json.Marshal(oldIng)
			require.NoError(t, err)

			ing := struct {
				Metadata metav1.ObjectMeta `json:"metadata"`
			}{
				Metadata: metav1.ObjectMeta{
					Name:        "name",
					Namespace:   "test",
					Annotations: test.ingAnno,
				},
			}
			b, err := json.Marshal(ing)
			require.NoError(t, err)

			ar := admv1.AdmissionReview{
				Request: &admv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: b,
					},
					OldObject: runtime.RawExtension{
						Raw: oldB,
					},
				},
			}

			p, err := rev.Review(context.Background(), ar)
			assert.NoError(t, err)
			assert.NotNil(t, p)

			var patches []map[string]interface{}
			err = json.Unmarshal(p, &patches)
			require.NoError(t, err)

			assert.Equal(t, 1, len(patches))
			assert.Equal(t, "replace", patches[0]["op"])
			assert.Equal(t, "/metadata/annotations", patches[0]["path"])
			assert.Equal(t, len(test.wantPatch), len(patches[0]["value"].(map[string]interface{})))
			for k := range test.wantPatch {
				assert.Equal(t, test.wantPatch[k], patches[0]["value"].(map[string]interface{})[k])
			}

			m, err := traefikClientSet.TraefikV1alpha1().Middlewares("test").Get(context.Background(), "zz-my-policy-test", metav1.GetOptions{})
			assert.NoError(t, err)
			assert.NotNil(t, m)

			assert.Equal(t, test.wantAuthResponseHeaders, m.Spec.ForwardAuth.AuthResponseHeaders)
		})
	}
}

func TestTraefikIngress_ReviewUpdatesExistingMiddleware(t *testing.T) {
	tests := []struct {
		desc                    string
		config                  *acp.Config
		wantAuthResponseHeaders []string
	}{
		{
			desc: "Update middleware with JWT configuration",
			config: &acp.Config{
				JWT: &jwt.Config{
					StripAuthorizationHeader: true,
				},
			},
			wantAuthResponseHeaders: []string{"Authorization"},
		},
		{
			desc: "Update middleware with basic configuration",
			config: &acp.Config{
				BasicAuth: &basicauth.Config{
					StripAuthorizationHeader: true,
				},
			},
			wantAuthResponseHeaders: []string{"Authorization"},
		},
		{
			desc: "Update middleware with digest configuration",
			config: &acp.Config{
				DigestAuth: &digestauth.Config{
					StripAuthorizationHeader: true,
				},
			},
			wantAuthResponseHeaders: []string{"Authorization"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			middleware := traefikv1alpha1.Middleware{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "zz-my-policy-test",
					Namespace: "test",
				},
				Spec: traefikv1alpha1.MiddlewareSpec{
					ForwardAuth: &traefikv1alpha1.ForwardAuth{
						AuthResponseHeaders: []string{"fwdHeader"},
					},
				},
			}
			traefikClientSet := traefikkubemock.NewSimpleClientset(&middleware)
			policies := func(canonicalName string) *acp.Config {
				return test.config
			}
			fwdAuthMdlwrs := NewFwdAuthMiddlewares("", policyGetterMock(policies), traefikClientSet.TraefikV1alpha1())
			rev := NewTraefikIngress(ingressClassesMock{}, fwdAuthMdlwrs, quota.New(999))

			ing := struct {
				Metadata metav1.ObjectMeta `json:"metadata"`
			}{
				Metadata: metav1.ObjectMeta{
					Name:        "name",
					Namespace:   "test",
					Annotations: map[string]string{AnnotationHubAuth: "my-policy@test"},
				},
			}
			b, err := json.Marshal(ing)
			require.NoError(t, err)

			ar := admv1.AdmissionReview{
				Request: &admv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: b,
					},
				},
			}

			m, err := traefikClientSet.TraefikV1alpha1().Middlewares("test").Get(context.Background(), "zz-my-policy-test", metav1.GetOptions{})
			assert.NoError(t, err)
			assert.NotNil(t, m)
			assert.Equal(t, []string{"fwdHeader"}, m.Spec.ForwardAuth.AuthResponseHeaders)

			p, err := rev.Review(context.Background(), ar)
			assert.NoError(t, err)
			assert.NotNil(t, p)

			m, err = traefikClientSet.TraefikV1alpha1().Middlewares("test").Get(context.Background(), "zz-my-policy-test", metav1.GetOptions{})
			assert.NoError(t, err)
			assert.NotNil(t, m)

			assert.Equal(t, test.wantAuthResponseHeaders, m.Spec.ForwardAuth.AuthResponseHeaders)
		})
	}
}

func TestTraefikIngress_ReviewRespectsQuotas(t *testing.T) {
	factory := func(quotas QuotaTransaction) reviewer {
		policies := policyGetterMock(func(string) *acp.Config {
			return &acp.Config{JWT: &jwt.Config{}}
		})
		fwdAuthMdlwrs := NewFwdAuthMiddlewares(
			"",
			policies,
			traefikkubemock.NewSimpleClientset().TraefikV1alpha1(),
		)

		return NewTraefikIngress(ingressClassesMock{}, fwdAuthMdlwrs, quotas)
	}

	reviewRespectsQuotas(t, factory)
}

func TestTraefikIngress_ReviewReleasesQuotasOnDelete(t *testing.T) {
	factory := func(quotas QuotaTransaction) reviewer {
		policies := policyGetterMock(func(string) *acp.Config {
			return &acp.Config{JWT: &jwt.Config{}}
		})
		fwdAuthMdlwrs := NewFwdAuthMiddlewares(
			"",
			policies,
			traefikkubemock.NewSimpleClientset().TraefikV1alpha1(),
		)

		return NewTraefikIngress(ingressClassesMock{}, fwdAuthMdlwrs, quotas)
	}

	reviewReleasesQuotasOnDelete(t, factory)
}

func TestTraefikIngress_reviewReleasesQuotasOnAnnotationRemove(t *testing.T) {
	factory := func(quotas QuotaTransaction) reviewer {
		policies := policyGetterMock(func(string) *acp.Config {
			return &acp.Config{JWT: &jwt.Config{}}
		})
		fwdAuthMdlwrs := NewFwdAuthMiddlewares(
			"",
			policies,
			traefikkubemock.NewSimpleClientset().TraefikV1alpha1(),
		)

		return NewTraefikIngress(ingressClassesMock{}, fwdAuthMdlwrs, quotas)
	}

	reviewReleasesQuotasOnAnnotationRemove(t, factory)
}

type policyGetterMock func(canonicalName string) *acp.Config

func (m policyGetterMock) GetConfig(canonicalName string) (*acp.Config, error) {
	return m(canonicalName), nil
}

type ingressClassesMock struct {
	getControllerFunc        func(name string) string
	getDefaultControllerFunc func() (string, error)
}

func (m ingressClassesMock) GetController(name string) string {
	return m.getControllerFunc(name)
}

func (m ingressClassesMock) GetDefaultController() (string, error) {
	return m.getDefaultControllerFunc()
}

type quotaMock struct {
	txFunc func(resourceID string, amount int) (*quota.Tx, error)
}

func (q quotaMock) Tx(resourceID string, amount int) (*quota.Tx, error) {
	return q.txFunc(resourceID, amount)
}
