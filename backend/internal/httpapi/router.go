package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewRouter(db *pgxpool.Pool, logger *slog.Logger, frontendOrigin string) http.Handler {
	api := &API{db: db, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("GET /api/v1/dashboard", api.dashboard)
	mux.HandleFunc("GET /api/v1/transactions", api.listTransactions)
	mux.HandleFunc("POST /api/v1/transactions", api.createTransaction)
	mux.HandleFunc("DELETE /api/v1/transactions/{id}", api.deleteTransaction)
	mux.HandleFunc("GET /api/v1/accounts", api.listAccounts)
	mux.HandleFunc("POST /api/v1/accounts", api.createAccount)
	mux.HandleFunc("PATCH /api/v1/accounts/{id}", api.updateAccount)
	mux.HandleFunc("DELETE /api/v1/accounts/{id}", api.deleteAccount)
	mux.HandleFunc("GET /api/v1/categories", api.listCategories)
	mux.HandleFunc("POST /api/v1/categories", api.createCategory)
	mux.HandleFunc("PATCH /api/v1/categories/{id}", api.updateCategory)
	mux.HandleFunc("DELETE /api/v1/categories/{id}", api.deleteCategory)
	mux.HandleFunc("GET /api/v1/goals", api.listGoals)
	mux.HandleFunc("POST /api/v1/goals", api.createGoal)
	mux.HandleFunc("PATCH /api/v1/goals/{id}", api.updateGoal)
	mux.HandleFunc("PUT /api/v1/goals/{id}", api.replaceGoal)
	mux.HandleFunc("DELETE /api/v1/goals/{id}", api.deleteGoal)
	mux.HandleFunc("GET /api/v1/investments", api.listInvestments)
	mux.HandleFunc("POST /api/v1/investments", api.createInvestment)
	mux.HandleFunc("PATCH /api/v1/investments/{id}", api.updateInvestment)
	mux.HandleFunc("DELETE /api/v1/investments/{id}", api.deleteInvestment)
	mux.HandleFunc("GET /api/v1/me", api.getProfile)
	mux.HandleFunc("PATCH /api/v1/me", api.updateProfile)
	mux.HandleFunc("GET /api/v1/workspaces", api.listWorkspaces)
	mux.HandleFunc("POST /api/v1/workspaces", api.createWorkspace)
	mux.HandleFunc("PATCH /api/v1/me/workspace", api.selectWorkspace)
	mux.HandleFunc("GET /api/v1/modules/pyramid", api.getPyramid)
	mux.HandleFunc("PATCH /api/v1/modules/pyramid/items/{id}", api.updatePyramidItem)
	mux.HandleFunc("GET /api/v1/modules/checkup", api.getFinancialCheckup)
	mux.HandleFunc("GET /api/v1/modules/emergency-fund", api.getEmergencyFund)
	mux.HandleFunc("PATCH /api/v1/modules/emergency-fund", api.updateEmergencyFund)
	mux.HandleFunc("GET /api/v1/modules/mortgage", api.getMortgageSimulation)
	mux.HandleFunc("PUT /api/v1/modules/mortgage", api.updateMortgageSimulation)
	mux.HandleFunc("GET /api/v1/modules/rebalancing", api.getRebalancing)
	mux.HandleFunc("GET /api/v1/modules/retirement", api.getRetirementPlan)
	mux.HandleFunc("PUT /api/v1/modules/retirement", api.updateRetirementPlan)
	mux.HandleFunc("GET /api/v1/settings/finance-period", api.getFinancePeriodSetting)
	mux.HandleFunc("PATCH /api/v1/settings/finance-period", api.updateFinancePeriodSetting)

	return recoverMiddleware(logger, loggingMiddleware(logger, corsMiddleware(frontendOrigin, mux)))
}

func (api *API) health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := api.db.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"status": "ok", "service": "hubby-finance-api"})
}

func corsMiddleware(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestOrigin := r.Header.Get("Origin")
		if requestOrigin == origin || (strings.HasPrefix(origin, "http://localhost") && strings.HasPrefix(requestOrigin, "http://localhost")) {
			w.Header().Set("Access-Control-Allow-Origin", requestOrigin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func recoverMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				logger.Error("panic recovered", "value", value)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
