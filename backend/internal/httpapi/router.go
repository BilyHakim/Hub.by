package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	db            *pgxpool.Pool
	logger        *slog.Logger
	secureCookies bool
	loginMu       sync.Mutex
	loginAttempts map[string]loginAttempt
}

func NewRouter(db *pgxpool.Pool, logger *slog.Logger, frontendOrigin string) http.Handler {
	api := &API{
		db: db, logger: logger, secureCookies: strings.HasPrefix(frontendOrigin, "https://"),
		loginAttempts: make(map[string]loginAttempt),
	}
	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/v1/dashboard", api.dashboard)
	protected.HandleFunc("GET /api/v1/transactions", api.listTransactions)
	protected.HandleFunc("GET /api/v1/transactions/recent", api.listRecentTransactions)
	protected.HandleFunc("POST /api/v1/transactions", api.createTransaction)
	protected.HandleFunc("DELETE /api/v1/transactions/{id}", api.deleteTransaction)
	protected.HandleFunc("POST /api/v1/transfers", api.createTransfer)
	protected.HandleFunc("DELETE /api/v1/transfers/{id}", api.deleteTransfer)
	protected.HandleFunc("GET /api/v1/budgets", api.getBudget)
	protected.HandleFunc("PUT /api/v1/budgets", api.replaceBudget)
	protected.HandleFunc("GET /api/v1/accounts", api.listAccounts)
	protected.HandleFunc("POST /api/v1/accounts", api.createAccount)
	protected.HandleFunc("PATCH /api/v1/accounts/{id}", api.updateAccount)
	protected.HandleFunc("DELETE /api/v1/accounts/{id}", api.deleteAccount)
	protected.HandleFunc("GET /api/v1/categories", api.listCategories)
	protected.HandleFunc("POST /api/v1/categories", api.createCategory)
	protected.HandleFunc("PATCH /api/v1/categories/{id}", api.updateCategory)
	protected.HandleFunc("DELETE /api/v1/categories/{id}", api.deleteCategory)
	protected.HandleFunc("GET /api/v1/goals", api.listGoals)
	protected.HandleFunc("POST /api/v1/goals", api.createGoal)
	protected.HandleFunc("PATCH /api/v1/goals/{id}", api.updateGoal)
	protected.HandleFunc("PUT /api/v1/goals/{id}", api.replaceGoal)
	protected.HandleFunc("DELETE /api/v1/goals/{id}", api.deleteGoal)
	protected.HandleFunc("GET /api/v1/investments", api.listInvestments)
	protected.HandleFunc("POST /api/v1/investments", api.createInvestment)
	protected.HandleFunc("PATCH /api/v1/investments/{id}", api.updateInvestment)
	protected.HandleFunc("DELETE /api/v1/investments/{id}", api.deleteInvestment)
	protected.HandleFunc("GET /api/v1/obligations", api.listObligations)
	protected.HandleFunc("POST /api/v1/obligations", api.createObligation)
	protected.HandleFunc("PUT /api/v1/obligations/{id}", api.updateObligation)
	protected.HandleFunc("DELETE /api/v1/obligations/{id}", api.deleteObligation)
	protected.HandleFunc("POST /api/v1/obligations/{id}/payments", api.createObligationPayment)
	protected.HandleFunc("DELETE /api/v1/obligation-payments/{id}", api.deleteObligationPayment)
	protected.HandleFunc("GET /api/v1/watch", api.getWatchOverview)
	protected.HandleFunc("POST /api/v1/watch/titles", api.createWatchTitle)
	protected.HandleFunc("PATCH /api/v1/watch/titles/{id}", api.updateWatchTitleStatus)
	protected.HandleFunc("DELETE /api/v1/watch/titles/{id}", api.deleteWatchTitle)
	protected.HandleFunc("POST /api/v1/watch/sessions", api.createWatchSession)
	protected.HandleFunc("DELETE /api/v1/watch/sessions/{id}", api.deleteWatchSession)
	protected.HandleFunc("GET /api/v1/me", api.getProfile)
	protected.HandleFunc("PATCH /api/v1/me", api.updateProfile)
	protected.HandleFunc("GET /api/v1/workspaces", api.listWorkspaces)
	protected.HandleFunc("POST /api/v1/workspaces", api.createWorkspace)
	protected.HandleFunc("PATCH /api/v1/me/workspace", api.selectWorkspace)
	protected.HandleFunc("GET /api/v1/modules/pyramid", api.getPyramid)
	protected.HandleFunc("PATCH /api/v1/modules/pyramid/items/{id}", api.updatePyramidItem)
	protected.HandleFunc("GET /api/v1/modules/checkup", api.getFinancialCheckup)
	protected.HandleFunc("GET /api/v1/modules/emergency-fund", api.getEmergencyFund)
	protected.HandleFunc("PATCH /api/v1/modules/emergency-fund", api.updateEmergencyFund)
	protected.HandleFunc("GET /api/v1/modules/mortgage", api.getMortgageSimulation)
	protected.HandleFunc("PUT /api/v1/modules/mortgage", api.updateMortgageSimulation)
	protected.HandleFunc("GET /api/v1/modules/rebalancing", api.getRebalancing)
	protected.HandleFunc("GET /api/v1/modules/retirement", api.getRetirementPlan)
	protected.HandleFunc("PUT /api/v1/modules/retirement", api.updateRetirementPlan)
	protected.HandleFunc("GET /api/v1/settings/finance-period", api.getFinancePeriodSetting)
	protected.HandleFunc("PATCH /api/v1/settings/finance-period", api.updateFinancePeriodSetting)
	protected.HandleFunc("POST /api/v1/auth/logout", api.logout)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("POST /api/v1/auth/login", api.login)
	mux.Handle("/api/v1/", api.requireAuth(protected))

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
			w.Header().Set("Access-Control-Allow-Credentials", "true")
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
