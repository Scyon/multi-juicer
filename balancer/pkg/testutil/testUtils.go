package testutil

import (
	"log"
	"os"

	"github.com/juice-shop/multi-juicer/balancer/pkg/bundle"
	"github.com/juice-shop/multi-juicer/balancer/pkg/signutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func NewTestBundle() *bundle.Bundle {
	clientset := fake.NewSimpleClientset()
	return NewTestBundleWithCustomFakeClient(clientset)
}

var testSigningKey = "test-signing-key"

func NewTestBundleWithCustomFakeClient(clientset kubernetes.Interface) *bundle.Bundle {
	return &bundle.Bundle{
		ClientSet:             clientset,
		StaticAssetsDirectory: "../ui/build/",
		RuntimeEnvironment: bundle.RuntimeEnvironment{
			Namespace: "test-namespace",
		},
		GeneratePasscode: func() string {
			return "12345678"
		},
		GetJuiceShopUrlForTeam: func(team string, bundle *bundle.Bundle) string {
			return "http://localhost:8080"
		},
		JuiceShopChallenges: []bundle.JuiceShopChallenge{
			{
				Key:        "scoreBoardChallenge",
				Name:       "Score Board",
				Difficulty: 1,
			},
			{
				Key:        "nullByteChallenge",
				Name:       "Poison Null Byte",
				Difficulty: 4,
			},
		},
		BcryptRounds: 2,
		Log:          log.New(os.Stdout, "", log.LstdFlags),
		Config: &bundle.Config{
			MaxInstances: 100,
			Settings: bundle.Settings{
				ScoreOverviewVisibleForUsers: true,
				BalancerEnabled:              true,
			},
			JuiceShopConfig: bundle.JuiceShopConfig{
				ImagePullPolicy: "IfNotPresent",
				Image:           "bkimminich/juice-shop",
				Tag:             "latest",
				NodeEnv:         "multi-juicer",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
			},
			CookieConfig: bundle.CookieConfig{
				SigningKey: testSigningKey,
				Name:       "team",
				Secure:     false,
			},
			AdminConfig: &bundle.AdminConfig{
				Password: "mock-admin-password",
			},
		},
	}
}

func SignTestTeamname(team string) string {
	signed, err := signutil.Sign(team, testSigningKey)
	if err != nil {
		panic(err)
	}
	return signed
}
