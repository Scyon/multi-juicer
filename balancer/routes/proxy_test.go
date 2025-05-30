package routes

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/juice-shop/multi-juicer/balancer/pkg/bundle"
	"github.com/juice-shop/multi-juicer/balancer/pkg/signutil"
	"github.com/juice-shop/multi-juicer/balancer/pkg/testutil"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestProxyHandler(t *testing.T) {
	teamFoo := "foobar"

	readyDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("juiceshop-%s", teamFoo),
			Namespace: "test-namespace",
			Annotations: map[string]string{
				"multi-juicer.owasp-juice.shop/challenges":          "[]",
				"multi-juicer.owasp-juice.shop/challengesSolved":    "0",
				"multi-juicer.owasp-juice.shop/lastRequest":         "1729259667397",
				"multi-juicer.owasp-juice.shop/lastRequestReadable": "2024-10-18 13:55:18.08198884+0000 UTC m=+11.556786174",
			},
			Labels: map[string]string{
				"app.kubernetes.io/name":    "juice-shop",
				"app.kubernetes.io/part-of": "multi-juicer",
			},
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 1,
		},
	}

	unreadyDeployment := readyDeployment.DeepCopy()
	unreadyDeployment.Status.ReadyReplicas = 0

	t.Run("redirects to /balancer when the balancer cookie is missing", func(t *testing.T) {
		defer clearInstanceUpCache()
		req, _ := http.NewRequest("POST", "/hello-world", nil)
		rr := httptest.NewRecorder()

		server := http.NewServeMux()

		bundle := testutil.NewTestBundle()
		AddRoutes(server, bundle, nil)

		server.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusFound, rr.Result().StatusCode)
		assert.Equal(t, "/balancer", rr.Header().Get("Location"))
		assert.Empty(t, rr.Body.String())
	})

	t.Run("redirects to /balancer when the balancer cookie is signed with another secret", func(t *testing.T) {
		defer clearInstanceUpCache()
		req, _ := http.NewRequest("POST", "/hello-world", nil)
		invalidlySignedTeam, err := signutil.Sign("invalid-team", "this-isn't-the-right-secret")
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("team=%s", invalidlySignedTeam))

		rr := httptest.NewRecorder()

		server := http.NewServeMux()

		bundle := testutil.NewTestBundle()
		AddRoutes(server, bundle, nil)

		server.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusFound, rr.Result().StatusCode)
		assert.Equal(t, "/balancer", rr.Header().Get("Location"))
		assert.Empty(t, rr.Body.String())
	})

	t.Run("routes the request to backend url generated by the JuiceShopUrlForTeam function", func(t *testing.T) {
		defer clearInstanceUpCache()
		req, _ := http.NewRequest("POST", "/hello-world", nil)
		req.Header.Set("Cookie", fmt.Sprintf("team=%s", testutil.SignTestTeamname(teamFoo)))
		rr := httptest.NewRecorder()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Hello, Test from "+r.URL.Path)
		}))
		defer ts.Close()

		server := http.NewServeMux()

		clientset := fake.NewSimpleClientset(readyDeployment)
		bu := testutil.NewTestBundleWithCustomFakeClient(clientset)

		bu.GetJuiceShopUrlForTeam = func(team string, _bundle *bundle.Bundle) string {
			return fmt.Sprintf("%s/%s/", ts.URL, team)
		}
		AddRoutes(server, bu, nil)

		server.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "Hello, Test from /foobar/hello-world\n", rr.Body.String())
	})

	t.Run("updates the deployment lastRequests annotation after a successful instance check", func(t *testing.T) {
		defer clearInstanceUpCache()
		req, _ := http.NewRequest("POST", "/hello-world", nil)
		req.Header.Set("Cookie", fmt.Sprintf("team=%s", testutil.SignTestTeamname(teamFoo)))
		rr := httptest.NewRecorder()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Hello, Test from "+r.URL.Path)
		}))
		defer ts.Close()

		server := http.NewServeMux()

		clientset := fake.NewSimpleClientset(readyDeployment)
		bu := testutil.NewTestBundleWithCustomFakeClient(clientset)

		bu.GetJuiceShopUrlForTeam = func(team string, _bundle *bundle.Bundle) string {
			return fmt.Sprintf("%s/%s/", ts.URL, team)
		}
		AddRoutes(server, bu, nil)

		server.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		updatedDeployment, err := clientset.AppsV1().Deployments(bu.RuntimeEnvironment.Namespace).Get(context.Background(), readyDeployment.Name, metav1.GetOptions{})
		assert.Nil(t, err)

		assert.NotEqual(t,
			readyDeployment.ObjectMeta.Annotations["multi-juicer.owasp-juice.shop/lastRequest"],
			updatedDeployment.ObjectMeta.Annotations["multi-juicer.owasp-juice.shop/lastRequest"],
		)
		assert.NotEqual(t,
			readyDeployment.ObjectMeta.Annotations["multi-juicer.owasp-juice.shop/lastRequestReadable"],
			updatedDeployment.ObjectMeta.Annotations["multi-juicer.owasp-juice.shop/lastRequestReadable"],
		)
	})

	t.Run("redirects to /balancer?msg=instance-restarting when the instance isn't ready", func(t *testing.T) {
		defer clearInstanceUpCache()
		req, _ := http.NewRequest("POST", "/hello-world", nil)
		req.Header.Set("Cookie", fmt.Sprintf("team=%s", testutil.SignTestTeamname(teamFoo)))
		rr := httptest.NewRecorder()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Hello, Test from "+r.URL.Path)
		}))
		defer ts.Close()

		server := http.NewServeMux()

		clientset := fake.NewSimpleClientset(unreadyDeployment)
		bu := testutil.NewTestBundleWithCustomFakeClient(clientset)

		bu.GetJuiceShopUrlForTeam = func(team string, _bundle *bundle.Bundle) string {
			return fmt.Sprintf("%s/%s/", ts.URL, team)
		}
		AddRoutes(server, bu, nil)

		server.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusFound, rr.Code)
		assert.Equal(t, fmt.Sprintf("/balancer/?msg=instance-restarting&team=%s", teamFoo), rr.Header().Get("Location"))
		assert.Empty(t, rr.Body.String())
	})
	t.Run("redirects to /balancer?msg=instance-not-found when the deployment doesn't exist", func(t *testing.T) {
		defer clearInstanceUpCache()
		req, _ := http.NewRequest("POST", "/hello-world", nil)
		req.Header.Set("Cookie", fmt.Sprintf("team=%s", testutil.SignTestTeamname(teamFoo)))
		rr := httptest.NewRecorder()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Hello, Test from "+r.URL.Path)
		}))
		defer ts.Close()

		server := http.NewServeMux()

		clientset := fake.NewSimpleClientset()
		bu := testutil.NewTestBundleWithCustomFakeClient(clientset)

		bu.GetJuiceShopUrlForTeam = func(team string, _bundle *bundle.Bundle) string {
			return fmt.Sprintf("%s/%s/", ts.URL, team)
		}
		AddRoutes(server, bu, nil)

		server.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusFound, rr.Code)
		assert.Equal(t, fmt.Sprintf("/balancer/?msg=instance-not-found&team=%s", teamFoo), rr.Header().Get("Location"))
		assert.Empty(t, rr.Body.String())
	})
	t.Run("redirects to /balancer?msg=balancer-disabled when the balancer is not enabled", func(t *testing.T) {
		defer clearInstanceUpCache()
		req, _ := http.NewRequest("POST", "/hello-world", nil)
		req.Header.Set("Cookie", fmt.Sprintf("team=%s", testutil.SignTestTeamname(teamFoo)))
		rr := httptest.NewRecorder()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Hello, Test from "+r.URL.Path)
		}))
		defer ts.Close()

		server := http.NewServeMux()

		clientset := fake.NewClientset(readyDeployment)
		bu := testutil.NewTestBundleWithCustomFakeClient(clientset)

		// disable balancer
		bu.UpdateBalancerEnabled(false)

		bu.GetJuiceShopUrlForTeam = func(team string, _bundle *bundle.Bundle) string {
			return fmt.Sprintf("%s/%s/", ts.URL, team)
		}
		AddRoutes(server, bu, nil)
		server.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusFound, rr.Code)
		assert.Equal(t, fmt.Sprintf("/balancer/?msg=balancer-disabled&team=%s", teamFoo), rr.Header().Get("Location"))
		assert.Empty(t, rr.Body.String())
	})

}
