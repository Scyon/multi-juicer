package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/juice-shop/multi-juicer/balancer/pkg/bundle"
	"github.com/juice-shop/multi-juicer/balancer/pkg/signutil"
	"golang.org/x/crypto/bcrypt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ResetPasscodeResponse struct {
	Message  string `json:"message"`
	Passcode string `json:"passcode"`
}

func handleResetPasscode(bundle *bundle.Bundle) http.Handler {
	return http.HandlerFunc(
		func(responseWriter http.ResponseWriter, req *http.Request) {

			teamSigned, err := req.Cookie("balancer")
			if err != nil {
				http.Error(responseWriter, "requires auth", http.StatusUnauthorized)
				return
			}
			team, err := signutil.Unsign(teamSigned.Value, bundle.Config.CookieConfig.SigningKey)
			if err != nil {
				http.Error(responseWriter, "requires auth", http.StatusUnauthorized)
				return
			}

			newPasscode := bundle.GeneratePasscode()

			deployment, err := bundle.ClientSet.AppsV1().Deployments(bundle.RuntimeEnvironment.Namespace).Get(req.Context(), fmt.Sprintf("juiceshop-%s", team), v1.GetOptions{})
			if err != nil {
				http.NotFound(responseWriter, req)
				return
			}

			passcodeHashBytes, err := bcrypt.GenerateFromPassword([]byte(newPasscode), bundle.BcryptRounds)
			if err != nil {
				bundle.Log.Printf("Failed to hash passcode!: %s", err)
				http.Error(responseWriter, "", http.StatusInternalServerError)
				return
			}
			passcodeHash := string(passcodeHashBytes)

			patch, err := json.Marshal(map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"multi-juicer.owasp-juice.shop/passcode": passcodeHash,
					},
				},
			})

			if err != nil {
				bundle.Log.Printf("Failed to convert passcode update patch to json: %v", err)
				http.Error(responseWriter, "Failed to update passcode", http.StatusInternalServerError)
				return
			}

			bundle.ClientSet.AppsV1().Deployments(bundle.RuntimeEnvironment.Namespace).Patch(
				context.Background(),
				deployment.Name, types.StrategicMergePatchType,
				patch,
				v1.PatchOptions{},
			)

			responseBody := ResetPasscodeResponse{
				Message:  "Passcode reset successfully",
				Passcode: newPasscode,
			}
			responseBodyEncoded, err := json.Marshal(responseBody)
			if err != nil {
				bundle.Log.Printf("Failed to encode passcode reset response: %v", err)
				http.Error(responseWriter, "Failed to reset passcode", http.StatusInternalServerError)
				return
			}

			responseWriter.WriteHeader(http.StatusOK)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.Write(responseBodyEncoded)
		},
	)
}
