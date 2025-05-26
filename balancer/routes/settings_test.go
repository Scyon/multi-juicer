package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/juice-shop/multi-juicer/balancer/pkg/bundle"
	"github.com/juice-shop/multi-juicer/balancer/pkg/testutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestHandleSettingsGet(t *testing.T) {
	tests := []struct {
		name           string
		setting        string
		cookie         string
		expectedStatus int
		expectedBody   settings
		setupBundle    func(bundle *bundle.Bundle)
	}{
		{
			name:           "Get scoreOverviewVisibleForUsers",
			setting:        "scoreOverviewVisibleForUsers",
			cookie:         fmt.Sprintf("team=%s", testutil.SignTestTeamname("admin")),
			expectedStatus: http.StatusOK,
			expectedBody: settings{
				"scoreOverviewVisibleForUsers": true,
			},
			setupBundle: func(b *bundle.Bundle) {
				b.UpdateScoreOverviewVisibleForUsers(true)
				b.UpdateBalancerEnabled(false)
			},
		},
		{
			name:           "Get all settings",
			setting:        "all",
			cookie:         fmt.Sprintf("team=%s", testutil.SignTestTeamname("admin")),
			expectedStatus: http.StatusOK,
			expectedBody: settings{
				"scoreOverviewVisibleForUsers": true,
				"balancerEnabled":              false,
			},
			setupBundle: func(b *bundle.Bundle) {
				b.UpdateScoreOverviewVisibleForUsers(true)
				b.UpdateBalancerEnabled(false)
			},
		},
		{
			name:           "Get non-existing setting",
			setting:        "this-setting-doesnt-exist",
			cookie:         fmt.Sprintf("team=%s", testutil.SignTestTeamname("admin")),
			expectedStatus: http.StatusBadRequest,
			expectedBody:   settings{},
			setupBundle: func(b *bundle.Bundle) {
				b.UpdateScoreOverviewVisibleForUsers(true)
				b.UpdateBalancerEnabled(false)
			},
		},
		{
			name:           "Not logged-in",
			setting:        "balancerEnabled",
			cookie:         "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   settings{},
			setupBundle: func(b *bundle.Bundle) {
				b.UpdateScoreOverviewVisibleForUsers(true)
				b.UpdateBalancerEnabled(false)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := testutil.NewTestBundle()
			tt.setupBundle(b)

			req, _ := http.NewRequest("GET", "/balancer/api/settings/"+tt.setting, nil)
			req.Header.Set("Cookie", tt.cookie)

			w := httptest.NewRecorder()
			server := http.NewServeMux()

			AddRoutes(server, b, nil)
			server.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}
			if !reflect.DeepEqual(tt.expectedBody, settings{}) {
				var response settings
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response body: %v", err)
				}
				if !reflect.DeepEqual(response, tt.expectedBody) {
					t.Fatalf("Expected response %+v, got %+v", tt.expectedBody, response)
				}
			}
		})
	}
}

func TestHandleSettingsPost(t *testing.T) {
	tests := []struct {
		name           string
		settings       interface{} // usually type "settings"
		cookie         string
		expectedStatus int
		responseNeedle string
	}{
		{
			name: "post settings",
			settings: settings{
				"scoreOverviewVisibleForUsers": true,
				"balancerEnabled":              true,
			},
			cookie:         fmt.Sprintf("team=%s", testutil.SignTestTeamname("admin")),
			expectedStatus: http.StatusOK,
			responseNeedle: "",
		},
		{
			name: "post settings as normal user",
			settings: settings{
				"scoreOverviewVisibleForUsers": true,
				"balancerEnabled":              true,
			},
			cookie:         fmt.Sprintf("team=%s", testutil.SignTestTeamname("test")),
			expectedStatus: http.StatusUnauthorized,
			responseNeedle: "",
		},
		{
			name: "post unknown setting",
			settings: settings{
				"nonExistingSetting": true,
			},
			cookie:         fmt.Sprintf("team=%s", testutil.SignTestTeamname("admin")),
			expectedStatus: http.StatusBadRequest,
			responseNeedle: "unknown setting",
		},
		{
			name:           "invalid body",
			settings:       "invalid-body",
			cookie:         fmt.Sprintf("team=%s", testutil.SignTestTeamname("admin")),
			expectedStatus: http.StatusBadRequest,
			responseNeedle: "invalid request body",
		},
		{
			name: "invalid value",
			settings: map[string]string{
				"balancerEnabled": "invalid-value",
			},
			cookie:         fmt.Sprintf("team=%s", testutil.SignTestTeamname("admin")),
			expectedStatus: http.StatusBadRequest,
			responseNeedle: "invalid value",
		},
		{
			name: "value as string",
			settings: map[string]string{
				"balancerEnabled": "true",
			},
			cookie:         fmt.Sprintf("team=%s", testutil.SignTestTeamname("admin")),
			expectedStatus: http.StatusBadRequest,
			responseNeedle: "invalid value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := testutil.NewTestBundle()

			reqBody, _ := json.Marshal(tt.settings)
			req, _ := http.NewRequest("POST", "/balancer/api/settings", bytes.NewReader(reqBody))
			req.Header.Set("Cookie", tt.cookie)

			w := httptest.NewRecorder()
			server := http.NewServeMux()

			AddRoutes(server, b, nil)

			server.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code != http.StatusOK {
				responseBody := w.Body.String()
				if !strings.Contains(responseBody, tt.responseNeedle) {
					t.Fatalf("Unexpected response body: %s", responseBody)
				}
			} else {
				for setting, value := range tt.settings.(settings) {
					switch setting {
					case "scoreOverviewVisibleForUsers":
						if value != b.GetScoreOverviewVisibleForUsers() {
							t.Fatalf("Value for %s not configured, expected: %t, found: %t", setting, value, b.GetScoreOverviewVisibleForUsers())
						}
					case "balancerEnabled":
						if value != b.GetBalancerEnabled() {
							t.Fatalf("Value for %s not configured, expected: %t, found: %t", setting, value, b.GetBalancerEnabled())
						}
					}
				}
			}
		})
	}
}
