package routes

import (
	"net/http"

	"github.com/juice-shop/multi-juicer/balancer/pkg/bundle"
	"github.com/juice-shop/multi-juicer/balancer/pkg/scoring"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var httpRequestsCount = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_count",
		Help: "Count of HTTP requests",
	},
	[]string{"method", "code"},
)

func init() {
	prometheus.MustRegister(httpRequestsCount)
}

func AddRoutes(
	router *http.ServeMux,
	bundle *bundle.Bundle,
	scoringService *scoring.ScoringService,
) {
	router.Handle("/", trackRequestMetrics(handleProxy(bundle)))
	router.Handle("GET /balancer", redirectLoggedInTeamsToStatus(bundle, handleStaticFiles(bundle)))
	router.Handle("GET /balancer/", handleStaticFiles(bundle))
	router.Handle("POST /balancer/api/teams/{team}/join", handleTeamJoin(bundle))
	router.Handle("POST /balancer/api/teams/logout", handleLogout(bundle))
	router.Handle("POST /balancer/api/teams/reset-passcode", handleResetPasscode(bundle))
	router.Handle("GET /balancer/api/score-board/top", handleScoreBoard(bundle, scoringService))
	router.Handle("GET /balancer/api/score-board/teams/{team}/score", handleIndividualScore(bundle, scoringService))
	router.Handle("GET /balancer/api/teams/status", handleTeamStatus(bundle, scoringService))

	router.Handle("GET /balancer/api/admin/all", handleAdminListInstances(bundle))
	router.Handle("DELETE /balancer/api/admin/teams/{team}/delete", handleAdminDeleteInstance(bundle))
	router.Handle("POST /balancer/api/admin/teams/{team}/restart", handleAdminRestartInstance(bundle))
	router.Handle("GET /balancer/api/admin/settings/{setting}", handleAdminSettingsGet(bundle))
	router.Handle("POST /balancer/api/admin/settings", handleAdminSettingsPost(bundle))

	router.HandleFunc("GET /balancer/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("GET /balancer/api/readiness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

func trackRequestMetrics(next http.Handler) http.Handler {
	return promhttp.InstrumentHandlerCounter(httpRequestsCount, next)
}
