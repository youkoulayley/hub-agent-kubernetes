/*
Copyright (C) 2022-2023 Traefik Labs

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

package state

import (
	traefikv1alpha1 "github.com/traefik/hub-agent-kubernetes/pkg/crd/api/traefik/v1alpha1"
	"github.com/traefik/hub-agent-kubernetes/pkg/httpclient"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Cluster describes a Cluster.
type Cluster struct {
	Ingresses             map[string]*Ingress             `json:"ingresses"`
	IngressRoutes         map[string]*IngressRoute        `json:"ingressRoutes"`
	Services              map[string]*Service             `json:"services"`
	AccessControlPolicies map[string]*AccessControlPolicy `json:"accessControlPolicies"`
	EdgeIngresses         map[string]*EdgeIngress         `json:"edgeIngresses"`
	APIs                  map[string]*API                 `json:"apis"`
	APIAccesses           map[string]*APIAccess           `json:"apiAccesses"`
	APICollections        map[string]*APICollection       `json:"apiCollections"`
	APIPortals            map[string]*APIPortal           `json:"apiPortals"`
	APIGateways           map[string]*APIGateway          `json:"apiGateways"`
	Nodes                 Nodes                           `json:"nodes"`
}

// ResourceMeta represents the metadata which identify a Kubernetes resource.
type ResourceMeta struct {
	Kind      string `json:"kind"`
	Group     string `json:"group"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// Service describes a Service.
type Service struct {
	Name          string             `json:"name"`
	Namespace     string             `json:"namespace"`
	Type          corev1.ServiceType `json:"type"`
	Annotations   map[string]string  `json:"annotations,omitempty"`
	ExternalIPs   []string           `json:"externalIPs,omitempty"`
	ExternalPorts []int              `json:"externalPorts,omitempty"`
}

// OpenAPISpecLocation describes the location of an OpenAPI specification.
type OpenAPISpecLocation struct {
	Path string `json:"path"`
	Port int    `json:"port"`
}

// IngressMeta represents the common Ingress metadata properties.
type IngressMeta struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// Ingress describes an Kubernetes Ingress.
type Ingress struct {
	ResourceMeta
	IngressMeta

	IngressClassName *string               `json:"ingressClassName,omitempty"`
	TLS              []netv1.IngressTLS    `json:"tls,omitempty"`
	Rules            []netv1.IngressRule   `json:"rules,omitempty"`
	DefaultBackend   *netv1.IngressBackend `json:"defaultBackend,omitempty"`
	Services         []string              `json:"services,omitempty"`
}

// IngressRoute describes a Traefik IngressRoute.
type IngressRoute struct {
	ResourceMeta
	IngressMeta

	TLS      *IngressRouteTLS `json:"tls,omitempty"`
	Routes   []Route          `json:"routes,omitempty"`
	Services []string         `json:"services,omitempty"`
}

// IngressRouteTLS represents a simplified Traefik IngressRoute TLS configuration.
type IngressRouteTLS struct {
	Domains    []traefikv1alpha1.Domain `json:"domains,omitempty"`
	SecretName string                   `json:"secretName,omitempty"`
	Options    *TLSOptionRef            `json:"options,omitempty"`
}

// TLSOptionRef references TLSOptions.
type TLSOptionRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// Route represents a Traefik IngressRoute route.
type Route struct {
	Match    string         `json:"match"`
	Services []RouteService `json:"services,omitempty"`
}

// RouteService represents a Kubernetes service targeted by a Traefik IngressRoute route.
type RouteService struct {
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	PortName   string `json:"portName,omitempty"`
	PortNumber int32  `json:"portNumber,omitempty"`
}

// AccessControlPolicy describes an Access Control Policy configured within a cluster.
type AccessControlPolicy struct {
	Name       string                         `json:"name"`
	Method     string                         `json:"method"`
	JWT        *AccessControlPolicyJWT        `json:"jwt,omitempty"`
	APIKey     *AccessControlPolicyAPIKey     `json:"apiKey,omitempty"`
	BasicAuth  *AccessControlPolicyBasicAuth  `json:"basicAuth,omitempty"`
	OIDC       *AccessControlPolicyOIDC       `json:"oidc,omitempty"`
	OIDCGoogle *AccessControlPolicyOIDCGoogle `json:"oidcGoogle,omitempty"`
	OAuthIntro *AccessControlPolicyOAuthIntro `json:"oAuthIntro,omitempty"`
}

// AccessControlPolicyJWT describes the settings for JWT authentication within an access control policy.
type AccessControlPolicyJWT struct {
	SigningSecret              string            `json:"signingSecret,omitempty"`
	SigningSecretBase64Encoded bool              `json:"signingSecretBase64Encoded"`
	PublicKey                  string            `json:"publicKey,omitempty"`
	JWKsFile                   string            `json:"jwksFile,omitempty"`
	JWKsURL                    string            `json:"jwksUrl,omitempty"`
	StripAuthorizationHeader   bool              `json:"stripAuthorizationHeader,omitempty"`
	ForwardHeaders             map[string]string `json:"forwardHeaders,omitempty"`
	TokenQueryKey              string            `json:"tokenQueryKey,omitempty"`
	Claims                     string            `json:"claims,omitempty"`
}

// AccessControlPolicyBasicAuth holds the HTTP basic authentication configuration.
type AccessControlPolicyBasicAuth struct {
	Users                    string `json:"users,omitempty"` // Redacted.
	Realm                    string `json:"realm,omitempty"`
	StripAuthorizationHeader bool   `json:"stripAuthorizationHeader,omitempty"`
	ForwardUsernameHeader    string `json:"forwardUsernameHeader,omitempty"`
}

// AccessControlPolicyAPIKey describes the settings for APIKey authentication within an access control policy.
type AccessControlPolicyAPIKey struct {
	KeySource      TokenSource                    `json:"keySource,omitempty"`
	Keys           []AccessControlPolicyAPIKeyKey `json:"keys,omitempty"`
	ForwardHeaders map[string]string              `json:"forwardHeaders,omitempty"`
}

// AccessControlPolicyAPIKeyKey defines an API key.
type AccessControlPolicyAPIKeyKey struct {
	ID       string            `json:"id"`
	Metadata map[string]string `json:"metadata"`
	Value    string            `json:"value"` // Redacted.
}

// AccessControlPolicyOIDC holds the OIDC configuration.
type AccessControlPolicyOIDC struct {
	Issuer   string           `json:"issuer,omitempty"`
	ClientID string           `json:"clientId,omitempty"`
	Secret   *SecretReference `json:"secret,omitempty"`

	RedirectURL string            `json:"redirectUrl,omitempty"`
	LogoutURL   string            `json:"logoutUrl,omitempty"`
	Scopes      []string          `json:"scopes,omitempty"`
	AuthParams  map[string]string `json:"authParams,omitempty"`
	StateCookie *AuthStateCookie  `json:"stateCookie,omitempty"`
	Session     *AuthSession      `json:"session,omitempty"`

	ForwardHeaders map[string]string `json:"forwardHeaders,omitempty"`
	Claims         string            `json:"claims,omitempty"`
}

// AccessControlPolicyOIDCGoogle holds the Google OIDC configuration.
type AccessControlPolicyOIDCGoogle struct {
	ClientID string           `json:"clientId,omitempty"`
	Secret   *SecretReference `json:"secret,omitempty"`

	RedirectURL string            `json:"redirectUrl,omitempty"`
	LogoutURL   string            `json:"logoutUrl,omitempty"`
	AuthParams  map[string]string `json:"authParams,omitempty"`
	StateCookie *AuthStateCookie  `json:"stateCookie,omitempty"`
	Session     *AuthSession      `json:"session,omitempty"`

	ForwardHeaders map[string]string `json:"forwardHeaders,omitempty"`
	Emails         []string          `json:"emails,omitempty"`
}

// SecretReference represents a Secret Reference.
// It has enough information to retrieve secret in any namespace.
type SecretReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// AuthStateCookie carries the state cookie configuration.
type AuthStateCookie struct {
	Path     string `json:"path,omitempty"`
	Domain   string `json:"domain,omitempty"`
	SameSite string `json:"sameSite,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
}

// AuthSession carries session and session cookie configuration.
type AuthSession struct {
	Path     string `json:"path,omitempty"`
	Domain   string `json:"domain,omitempty"`
	SameSite string `json:"sameSite,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	Refresh  *bool  `json:"refresh,omitempty"`
}

// AccessControlPolicyOAuthIntro holds the OAuth 2.0 token introspection configuration.
type AccessControlPolicyOAuthIntro struct {
	ClientConfig   ClientConfig      `json:"clientConfig,omitempty"`
	TokenSource    TokenSource       `json:"tokenSource,omitempty"`
	Claims         string            `json:"claims,omitempty"`
	ForwardHeaders map[string]string `json:"forwardHeaders,omitempty"`
}

// ClientConfig configures the HTTP client of the OAuth 2.0 Token Introspection ACP handler.
type ClientConfig struct {
	httpclient.Config

	URL           string            `json:"url,omitempty"`
	Auth          ClientConfigAuth  `json:"auth,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	TokenTypeHint string            `json:"tokenTypeHint,omitempty"`
}

// ClientConfigAuth configures authentication to the Authorization Server.
type ClientConfigAuth struct {
	Kind   string          `json:"kind"`
	Secret SecretReference `json:"secret"`
}

// TokenSource describes where to find a token in an HTTP request.
type TokenSource struct {
	Header           string `json:"header,omitempty"`
	HeaderAuthScheme string `json:"headerAuthScheme,omitempty"`
	Query            string `json:"query,omitempty"`
	Cookie           string `json:"cookie,omitempty"`
}

// EdgeIngress holds the definition of an EdgeIngress configuration.
type EdgeIngress struct {
	Name      string             `json:"name"`
	Namespace string             `json:"namespace"`
	Status    EdgeIngressStatus  `json:"status"`
	Service   EdgeIngressService `json:"service"`
	ACP       *EdgeIngressACP    `json:"acp,omitempty"`
}

// EdgeIngressStatus is the exposition status of an edge ingress.
type EdgeIngressStatus string

// Possible value of the EdgeIngressStatus.
const (
	EdgeIngressStatusUp   EdgeIngressStatus = "up"
	EdgeIngressStatusDown EdgeIngressStatus = "down"
)

// EdgeIngressService configures the service to exposed on the edge.
type EdgeIngressService struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

// EdgeIngressACP configures the ACP to use on the Ingress.
type EdgeIngressACP struct {
	Name string `json:"name"`
}

// API holds the definition of an API configuration.
type API struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels,omitempty"`

	PathPrefix string     `json:"pathPrefix"`
	Service    APIService `json:"service"`
}

// APIService configures the service to exposed on the edge.
type APIService struct {
	Name        string                `json:"name"`
	Port        APIServiceBackendPort `json:"port"`
	OpenAPISpec OpenAPISpec           `json:"openApiSpec,omitempty"`
}

// APIServiceBackendPort is the service port being referenced.
type APIServiceBackendPort struct {
	Name   string `json:"name"`
	Number int32  `json:"number"`
}

// OpenAPISpec defines the OpenAPI spec of an API.
type OpenAPISpec struct {
	URL      string                 `json:"url,omitempty"`
	Path     string                 `json:"path,omitempty"`
	Port     *APIServiceBackendPort `json:"port,omitempty"`
	Protocol string                 `json:"protocol,omitempty"`
}

// APIAccess holds the definition of an APIAccess configuration.
type APIAccess struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`

	Groups                []string              `json:"groups"`
	APISelector           *metav1.LabelSelector `json:"apiSelector"`
	APICollectionSelector *metav1.LabelSelector `json:"apiCollectionSelector"`
}

// APICollection holds the definition of an APICollection resource.
type APICollection struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`

	PathPrefix  string               `json:"pathPrefix,omitempty"`
	APISelector metav1.LabelSelector `json:"apiSelector"`
}

// APIPortal holds the definition of an APIPortal configuration.
type APIPortal struct {
	Name string `json:"name"`

	Description   string   `json:"description,omitempty"`
	APIGateway    string   `json:"apiGateway"`
	CustomDomains []string `json:"customDomains,omitempty"`
	HubDomain     string   `json:"hubDomain"`
}

// APIGateway holds the definition of an APIGateway resource.
type APIGateway struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`

	APIAccesses   []string `json:"apiAccesses,omitempty"`
	CustomDomains []string `json:"customDomains,omitempty"`
	HubDomain     string   `json:"hubDomain"`
}

// Nodes holds the number of nodes running Services exposed by APIs and the total number of nodes in the cluster.
type Nodes struct {
	RunningAPIs int `json:"runningApis"`
	Total       int `json:"total"`
}
