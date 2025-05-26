package routes

import (
	"encoding/json"
	"fmt"
	"github.com/juice-shop/multi-juicer/balancer/pkg/bundle"
	"github.com/juice-shop/multi-juicer/balancer/pkg/teamcookie"
	"net/http"
)

type settings = map[string]interface{}

func handleSettingsGet(bundle *bundle.Bundle) http.Handler {
	return http.HandlerFunc(
		func(responseWriter http.ResponseWriter, req *http.Request) {
			_, err := teamcookie.GetTeamFromRequest(bundle, req)
			if err != nil {
				http.Error(responseWriter, "", http.StatusUnauthorized)
				return
			}
			data := req.PathValue("setting")
			var response settings
			switch data {
			case "scoreOverviewVisibleForUsers":
				value := bundle.GetScoreOverviewVisibleForUsers()
				response = settings{
					"scoreOverviewVisibleForUsers": value,
				}
			case "balancerEnabled":
				value := bundle.GetBalancerEnabled()
				response = settings{
					"balancerEnabled": value,
				}
			case "all":
				response = settings{
					"scoreOverviewVisibleForUsers": bundle.GetScoreOverviewVisibleForUsers(),
					"balancerEnabled":              bundle.GetBalancerEnabled(),
				}
			default:
				http.Error(responseWriter, "Unknown setting", http.StatusBadRequest)
				return
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				bundle.Log.Printf("Failed to marshal response: %s", err)
				http.Error(responseWriter, "", http.StatusInternalServerError)
				return
			}

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			responseWriter.Write(responseBytes)
		})
}

func handleSettingsPost(bundle *bundle.Bundle) http.Handler {
	return http.HandlerFunc(
		func(responseWriter http.ResponseWriter, req *http.Request) {
			team, err := teamcookie.GetTeamFromRequest(bundle, req)
			if err != nil || team != "admin" {
				http.Error(responseWriter, "", http.StatusUnauthorized)
				return
			}

			var data settings

			if req.Body == nil {
				http.Error(responseWriter, "invalid request body", http.StatusBadRequest)
				return
			}
			if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
				http.Error(responseWriter, "invalid request body", http.StatusBadRequest)
				return
			}
			defer req.Body.Close()

			for setting, value := range data {
				switch setting {
				case "scoreOverviewVisibleForUsers":
					fallthrough
				case "balancerEnabled":
					if _, ok := value.(bool); !ok {
						http.Error(responseWriter, fmt.Sprintf("invalid value: %s, for setting: %s", value, setting), http.StatusBadRequest)
						return
					}
				default:
					http.Error(responseWriter, fmt.Sprintf("unknown setting: %s", setting), http.StatusBadRequest)
					return
				}
			}

			for setting, value := range data {
				switch setting {
				case "scoreOverviewVisibleForUsers":
					bundle.UpdateScoreOverviewVisibleForUsers(value.(bool))
				case "balancerEnabled":
					bundle.UpdateBalancerEnabled(value.(bool))
				}
			}

			bundle.Log.Printf("settings updated: %+v", data)

			responseWriter.WriteHeader(http.StatusOK)
			responseWriter.Write([]byte{})
		})
}
