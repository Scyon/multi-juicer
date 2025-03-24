package routes

import (
	"encoding/json"
	"github.com/juice-shop/multi-juicer/balancer/pkg/bundle"
	"github.com/juice-shop/multi-juicer/balancer/pkg/teamcookie"
	"net/http"
)

type setting struct {
	Name  string `json:"setting"`
	Value string `json:"value"`
}

func handleSettingsGet(bundle *bundle.Bundle) http.Handler {
	return http.HandlerFunc(
		func(responseWriter http.ResponseWriter, req *http.Request) {
			_, err := teamcookie.GetTeamFromRequest(bundle, req)
			if err != nil {
				http.Error(responseWriter, "", http.StatusUnauthorized)
				return
			}
			data := req.PathValue("setting")
			var response setting
			switch data {
			case "score-visibility":
				value := bundle.GetScoreOverviewVisibility()
				response = setting{
					Name:  "score-visibility",
					Value: value,
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

			var data setting
			if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
				http.Error(responseWriter, "Invalid request body", http.StatusBadRequest)
				return
			}
			defer req.Body.Close()

			switch data.Name {
			case "score-visibility":
				if err := bundle.UpdateScoreOverviewVisibility(data.Value); err != nil {
					http.Error(responseWriter, err.Error(), http.StatusBadRequest)
					return
				}
			default:
				http.Error(responseWriter, "Unknown setting", http.StatusBadRequest)
				return
			}

			bundle.Log.Printf("Setting updated: %+v", data)

			responseWriter.WriteHeader(http.StatusOK)
			responseWriter.Write([]byte{})
		})
}
