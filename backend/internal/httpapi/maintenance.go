package httpapi

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type maintenanceItem struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	Category       string   `json:"category"`
	Brand          string   `json:"brand"`
	Model          string   `json:"model"`
	PurchaseDate   string   `json:"purchaseDate"`
	StartUsageDate string   `json:"startUsageDate"`
	PurchasePrice  *float64 `json:"purchasePrice"`
	SerialNumber   string   `json:"serialNumber"`
	WarrantyExpiry string   `json:"warrantyExpiry"`
	Location       string   `json:"location"`
	ImageURL       string   `json:"imageUrl"`
	Notes          string   `json:"notes"`
	CurrentUsage   *float64 `json:"currentUsage"`
	UsageUnit      string   `json:"usageUnit"`
}

type maintenanceRule struct {
	ID              int64    `json:"id"`
	ItemID          int64    `json:"itemId"`
	Name            string   `json:"name"`
	Kind            string   `json:"kind"`
	IntervalValue   *int     `json:"intervalValue"`
	IntervalUnit    string   `json:"intervalUnit"`
	UsageInterval   *float64 `json:"usageInterval"`
	StartDate       string   `json:"startDate"`
	StartUsage      float64  `json:"startUsage"`
	ReminderDays    int      `json:"reminderDays"`
	ReminderUsage   float64  `json:"reminderUsage"`
	Active          bool     `json:"active"`
	LastMaintenance string   `json:"lastMaintenance"`
	LastUsage       *float64 `json:"lastUsage"`
	NextDueDate     string   `json:"nextDueDate"`
	NextDueUsage    *float64 `json:"nextDueUsage"`
	Status          string   `json:"status"`
}

type maintenanceCompletion struct {
	CompletedAt string   `json:"completedAt"`
	Cost        *float64 `json:"cost"`
	Vendor      string   `json:"vendor"`
	UsageValue  *float64 `json:"usageValue"`
	Notes       string   `json:"notes"`
}

func maintenanceToday() time.Time {
	return time.Now().UTC().Add(7 * time.Hour).Truncate(24 * time.Hour)
}

func maintenanceDate(value string) (time.Time, bool) {
	date, err := time.Parse("2006-01-02", value)
	return date, err == nil && date.Year() >= 1900 && date.Year() <= 2200
}

func validMaintenanceNumber(value *float64) bool {
	return value == nil || (!math.IsNaN(*value) && !math.IsInf(*value, 0) && *value >= 0 && *value <= 1e12)
}

func validMaintenanceItem(item *maintenanceItem) bool {
	item.Name = strings.TrimSpace(item.Name)
	item.Category = strings.TrimSpace(item.Category)
	if item.UsageUnit == "" {
		item.UsageUnit = "km"
	}
	if item.Name == "" || len(item.Name) > 200 || len(item.Category) > 100 || len(item.Brand) > 200 || len(item.Model) > 200 || len(item.SerialNumber) > 200 || len(item.Location) > 200 || len(item.Notes) > 4000 || len(item.ImageURL) > 1000 {
		return false
	}
	for _, date := range []string{item.PurchaseDate, item.StartUsageDate, item.WarrantyExpiry} {
		if date != "" {
			if _, ok := maintenanceDate(date); !ok {
				return false
			}
		}
	}
	if item.ImageURL != "" {
		u, err := url.Parse(item.ImageURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return false
		}
	}
	return validMaintenanceNumber(item.PurchasePrice) && validMaintenanceNumber(item.CurrentUsage) && (item.UsageUnit == "km" || item.UsageUnit == "hours" || item.UsageUnit == "cycles")
}

func validMaintenanceRule(rule *maintenanceRule) bool {
	rule.Name = strings.TrimSpace(rule.Name)
	if rule.Name == "" || len(rule.Name) > 200 || (rule.Kind != "maintenance" && rule.Kind != "replacement") {
		return false
	}
	if _, ok := maintenanceDate(rule.StartDate); !ok {
		return false
	}
	if rule.IntervalValue == nil {
		if rule.IntervalUnit != "" {
			return false
		}
	} else {
		if *rule.IntervalValue < 1 || *rule.IntervalValue > 10000 {
			return false
		}
		if rule.IntervalUnit == "years" && *rule.IntervalValue > 1000 {
			return false
		}
		switch rule.IntervalUnit {
		case "days", "weeks", "months", "years":
		default:
			return false
		}
	}
	if rule.IntervalValue == nil && rule.UsageInterval == nil {
		return false
	}
	return validMaintenanceNumber(rule.UsageInterval) && (rule.UsageInterval == nil || *rule.UsageInterval > 0) && validMaintenanceNumber(&rule.StartUsage) && validMaintenanceNumber(&rule.ReminderUsage) && rule.ReminderDays >= 0 && rule.ReminderDays <= 365
}

func maintenanceNextDate(base time.Time, value int, unit string) time.Time {
	switch unit {
	case "days":
		return base.AddDate(0, 0, value)
	case "weeks":
		return base.AddDate(0, 0, value*7)
	}
	months := value
	if unit == "years" {
		months *= 12
	}
	first := time.Date(base.Year(), base.Month()+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	lastDay := first.AddDate(0, 1, -1).Day()
	day := base.Day()
	if day > lastDay {
		day = lastDay
	}
	return first.AddDate(0, 0, day-1)
}

func calculateMaintenanceRule(rule *maintenanceRule, currentUsage *float64, today time.Time) {
	rule.Status = "good"
	if !rule.Active {
		rule.Status = "inactive"
		return
	}
	if rule.IntervalValue != nil {
		base := rule.StartDate
		if rule.LastMaintenance > base {
			base = rule.LastMaintenance
		}
		date, _ := maintenanceDate(base)
		next := maintenanceNextDate(date, *rule.IntervalValue, rule.IntervalUnit)
		rule.NextDueDate = next.Format("2006-01-02")
		switch {
		case next.Before(today):
			rule.Status = "overdue"
		case next.Equal(today):
			rule.Status = "due_today"
		case !next.After(today.AddDate(0, 0, rule.ReminderDays)):
			rule.Status = "due_soon"
		}
	}
	if rule.UsageInterval != nil {
		base := rule.StartUsage
		if rule.LastUsage != nil && *rule.LastUsage > base {
			base = *rule.LastUsage
		}
		next := base + *rule.UsageInterval
		rule.NextDueUsage = &next
		if currentUsage == nil && rule.Status == "good" {
			rule.Status = "usage_unknown"
		}
		if currentUsage != nil {
			switch {
			case *currentUsage > next:
				rule.Status = "overdue"
			case *currentUsage == next && rule.Status != "overdue":
				rule.Status = "due_today"
			case *currentUsage >= next-rule.ReminderUsage && rule.Status == "good":
				rule.Status = "due_soon"
			}
		}
	}
}

func (api *API) maintenanceScope(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, 500, "failed to resolve active workspace")
		return 0, false
	}
	return id, true
}

func maintenanceID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, 400, "invalid maintenance id")
		return 0, false
	}
	return id, true
}

func maintenanceDBError(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "data maintenance tidak ditemukan di workspace aktif")
	} else {
		writeError(w, 500, "gagal menyimpan atau memuat maintenance")
	}
}

func (api *API) getMaintenance(w http.ResponseWriter, r *http.Request) {
	workspace, ok := api.maintenanceScope(w, r)
	if !ok {
		return
	}
	rows, err := api.db.Query(r.Context(), `SELECT i.id,i.name,COALESCE(c.name,''),i.brand,i.model,COALESCE(i.purchase_date::text,''),COALESCE(i.start_usage_date::text,''),i.purchase_price::float8,i.serial_number,COALESCE(i.warranty_expiry::text,''),i.location,i.image_url,i.notes,i.current_usage::float8,i.usage_unit FROM maintenance_items i LEFT JOIN maintenance_categories c ON c.id=i.category_id AND c.workspace_id=i.workspace_id WHERE i.workspace_id=$1 ORDER BY i.updated_at DESC,i.id DESC`, workspace)
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	items := make([]maintenanceItem, 0)
	for rows.Next() {
		var i maintenanceItem
		if err = rows.Scan(&i.ID, &i.Name, &i.Category, &i.Brand, &i.Model, &i.PurchaseDate, &i.StartUsageDate, &i.PurchasePrice, &i.SerialNumber, &i.WarrantyExpiry, &i.Location, &i.ImageURL, &i.Notes, &i.CurrentUsage, &i.UsageUnit); err != nil {
			rows.Close()
			maintenanceDBError(w, err)
			return
		}
		items = append(items, i)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	rules, err := api.loadMaintenanceRules(r.Context(), workspace, 0)
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	byItem := map[int64]*float64{}
	for _, i := range items {
		byItem[i.ID] = i.CurrentUsage
	}
	for index := range rules {
		calculateMaintenanceRule(&rules[index], byItem[rules[index].ItemID], maintenanceToday())
	}
	histories, err := api.loadMaintenanceHistories(r.Context(), workspace)
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	categories := make([]string, 0)
	categoryRows, err := api.db.Query(r.Context(), `SELECT name FROM maintenance_categories WHERE workspace_id=$1 ORDER BY name`, workspace)
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	defer categoryRows.Close()
	for categoryRows.Next() {
		var name string
		if err = categoryRows.Scan(&name); err != nil {
			maintenanceDBError(w, err)
			return
		}
		categories = append(categories, name)
	}
	if err = categoryRows.Err(); err != nil {
		maintenanceDBError(w, err)
		return
	}
	writeJSON(w, 200, envelope{"data": envelope{"items": items, "rules": rules, "histories": histories, "categories": categories, "today": maintenanceToday().Format("2006-01-02")}})
}

func (api *API) loadMaintenanceRules(ctx context.Context, workspace, itemID int64) ([]maintenanceRule, error) {
	rows, err := api.db.Query(ctx, `SELECT r.id,r.item_id,r.name,r.kind,r.interval_value,COALESCE(r.interval_unit,''),r.usage_interval::float8,r.start_date::text,r.start_usage::float8,r.reminder_days,r.reminder_usage::float8,r.active,COALESCE(h.last_date::text,''),h.last_usage::float8 FROM maintenance_rules r LEFT JOIN LATERAL (SELECT MAX(completed_at) last_date,MAX(usage_value) last_usage FROM maintenance_histories WHERE rule_id=r.id AND workspace_id=r.workspace_id) h ON TRUE WHERE r.workspace_id=$1 AND ($2::bigint=0 OR r.item_id=$2) ORDER BY r.id`, workspace, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]maintenanceRule, 0)
	for rows.Next() {
		var rule maintenanceRule
		if err = rows.Scan(&rule.ID, &rule.ItemID, &rule.Name, &rule.Kind, &rule.IntervalValue, &rule.IntervalUnit, &rule.UsageInterval, &rule.StartDate, &rule.StartUsage, &rule.ReminderDays, &rule.ReminderUsage, &rule.Active, &rule.LastMaintenance, &rule.LastUsage); err != nil {
			return nil, err
		}
		result = append(result, rule)
	}
	return result, rows.Err()
}

func (api *API) loadMaintenanceHistories(ctx context.Context, workspace int64) ([]envelope, error) {
	rows, err := api.db.Query(ctx, `SELECT h.id,h.item_id,h.rule_id,i.name,h.rule_name,h.completed_at::text,h.cost::float8,h.vendor,h.usage_value::float8,h.notes FROM maintenance_histories h JOIN maintenance_items i ON i.id=h.item_id AND i.workspace_id=h.workspace_id WHERE h.workspace_id=$1 ORDER BY h.completed_at DESC,h.id DESC`, workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]envelope, 0)
	for rows.Next() {
		var id, itemID, ruleID int64
		var itemName, ruleName, date, vendor, notes string
		var cost, usage *float64
		if err = rows.Scan(&id, &itemID, &ruleID, &itemName, &ruleName, &date, &cost, &vendor, &usage, &notes); err != nil {
			return nil, err
		}
		result = append(result, envelope{"id": id, "itemId": itemID, "ruleId": ruleID, "itemName": itemName, "ruleName": ruleName, "completedAt": date, "cost": cost, "vendor": vendor, "usageValue": usage, "notes": notes})
	}
	return result, rows.Err()
}

func (api *API) saveMaintenanceItem(w http.ResponseWriter, r *http.Request) {
	workspace, ok := api.maintenanceScope(w, r)
	if !ok {
		return
	}
	var input maintenanceItem
	if decodeJSON(r, &input) != nil {
		writeError(w, 400, "invalid item request")
		return
	}
	if !validMaintenanceItem(&input) {
		writeError(w, 422, "periksa nama, tanggal, URL gambar, dan nilai barang")
		return
	}
	var id int64
	if r.Method == http.MethodPut {
		id, ok = maintenanceID(w, r)
		if !ok {
			return
		}
	}
	tx, err := api.db.Begin(r.Context())
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if id != 0 {
		var unit string
		err = tx.QueryRow(r.Context(), `SELECT usage_unit FROM maintenance_items WHERE id=$1 AND workspace_id=$2 FOR UPDATE`, id, workspace).Scan(&unit)
		if err != nil {
			maintenanceDBError(w, err)
			return
		}
		var usageRules bool
		var maxUsage *float64
		err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM maintenance_rules WHERE item_id=$1 AND workspace_id=$2 AND usage_interval IS NOT NULL),(SELECT MAX(usage_value)::float8 FROM maintenance_histories WHERE item_id=$1 AND workspace_id=$2)`, id, workspace).Scan(&usageRules, &maxUsage)
		if err != nil {
			maintenanceDBError(w, err)
			return
		}
		if ((usageRules || maxUsage != nil) && unit != input.UsageUnit) || (maxUsage != nil && (input.CurrentUsage == nil || *input.CurrentUsage < *maxUsage)) {
			writeError(w, 422, "satuan penggunaan tidak dapat diubah setelah memiliki aturan; nilai penggunaan tidak boleh kurang dari riwayat")
			return
		}
	}
	var categoryID *int64
	if input.Category != "" {
		var category int64
		err = tx.QueryRow(r.Context(), `INSERT INTO maintenance_categories(workspace_id,name) VALUES($1,$2) ON CONFLICT(workspace_id,name) DO UPDATE SET name=EXCLUDED.name RETURNING id`, workspace, input.Category).Scan(&category)
		if err != nil {
			maintenanceDBError(w, err)
			return
		}
		categoryID = &category
	}
	args := []any{workspace, categoryID, input.Name, input.Brand, input.Model, input.PurchaseDate, input.StartUsageDate, input.PurchasePrice, input.SerialNumber, input.WarrantyExpiry, input.Location, input.ImageURL, input.Notes, input.CurrentUsage, input.UsageUnit}
	if id == 0 {
		err = tx.QueryRow(r.Context(), `INSERT INTO maintenance_items(workspace_id,category_id,name,brand,model,purchase_date,start_usage_date,purchase_price,serial_number,warranty_expiry,location,image_url,notes,current_usage,usage_unit) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::date,NULLIF($7,'')::date,$8,$9,NULLIF($10,'')::date,$11,$12,$13,$14,$15) RETURNING id`, args...).Scan(&id)
	} else {
		args = append(args, id)
		_, err = tx.Exec(r.Context(), `UPDATE maintenance_items SET category_id=$2,name=$3,brand=$4,model=$5,purchase_date=NULLIF($6,'')::date,start_usage_date=NULLIF($7,'')::date,purchase_price=$8,serial_number=$9,warranty_expiry=NULLIF($10,'')::date,location=$11,image_url=$12,notes=$13,current_usage=$14,usage_unit=$15,updated_at=now() WHERE workspace_id=$1 AND id=$16`, args...)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	status := 200
	if r.Method == http.MethodPost {
		status = 201
	}
	writeJSON(w, status, envelope{"data": envelope{"id": id}})
}

func (api *API) deleteMaintenanceItem(w http.ResponseWriter, r *http.Request) {
	workspace, ok := api.maintenanceScope(w, r)
	if !ok {
		return
	}
	id, ok := maintenanceID(w, r)
	if !ok {
		return
	}
	result, err := api.db.Exec(r.Context(), `DELETE FROM maintenance_items WHERE id=$1 AND workspace_id=$2`, id, workspace)
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	if result.RowsAffected() == 0 {
		maintenanceDBError(w, pgx.ErrNoRows)
		return
	}
	w.WriteHeader(204)
}

func (api *API) saveMaintenanceRule(w http.ResponseWriter, r *http.Request) {
	workspace, ok := api.maintenanceScope(w, r)
	if !ok {
		return
	}
	id, ok := maintenanceID(w, r)
	if !ok {
		return
	}
	var input maintenanceRule
	if decodeJSON(r, &input) != nil {
		writeError(w, 400, "invalid rule request")
		return
	}
	if !validMaintenanceRule(&input) {
		writeError(w, 422, "isi nama, tanggal acuan, dan interval yang valid")
		return
	}
	tx, err := api.db.Begin(r.Context())
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	itemID := id
	if r.Method == http.MethodPut {
		err = tx.QueryRow(r.Context(), `SELECT item_id FROM maintenance_rules WHERE id=$1 AND workspace_id=$2`, id, workspace).Scan(&itemID)
		if err != nil {
			maintenanceDBError(w, err)
			return
		}
	}
	var exists int64
	err = tx.QueryRow(r.Context(), `SELECT id FROM maintenance_items WHERE id=$1 AND workspace_id=$2 FOR UPDATE`, itemID, workspace).Scan(&exists)
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	args := []any{workspace, itemID, input.Name, input.Kind, input.IntervalValue, input.IntervalUnit, input.UsageInterval, input.StartDate, input.StartUsage, input.ReminderDays, input.ReminderUsage, input.Active}
	if r.Method == http.MethodPost {
		err = tx.QueryRow(r.Context(), `INSERT INTO maintenance_rules(workspace_id,item_id,name,kind,interval_value,interval_unit,usage_interval,start_date,start_usage,reminder_days,reminder_usage,active) VALUES($1,$2,$3,$4,$5,NULLIF($6,''),$7,$8::date,$9,$10,$11,$12) RETURNING id`, args...).Scan(&id)
	} else {
		args = append(args, id)
		_, err = tx.Exec(r.Context(), `UPDATE maintenance_rules SET name=$3,kind=$4,interval_value=$5,interval_unit=NULLIF($6,''),usage_interval=$7,start_date=$8::date,start_usage=$9,reminder_days=$10,reminder_usage=$11,active=$12,updated_at=now() WHERE workspace_id=$1 AND item_id=$2 AND id=$13`, args...)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	status := 200
	if r.Method == http.MethodPost {
		status = 201
	}
	writeJSON(w, status, envelope{"data": envelope{"id": id}})
}

func (api *API) completeMaintenance(w http.ResponseWriter, r *http.Request) {
	workspace, ok := api.maintenanceScope(w, r)
	if !ok {
		return
	}
	id, ok := maintenanceID(w, r)
	if !ok {
		return
	}
	var input maintenanceCompletion
	if decodeJSON(r, &input) != nil {
		writeError(w, 400, "invalid completion request")
		return
	}
	date, valid := maintenanceDate(input.CompletedAt)
	if !valid || date.After(maintenanceToday()) || !validMaintenanceNumber(input.Cost) || !validMaintenanceNumber(input.UsageValue) || len(input.Vendor) > 200 || len(input.Notes) > 4000 {
		writeError(w, 422, "periksa tanggal penyelesaian, biaya, dan penggunaan")
		return
	}
	tx, err := api.db.Begin(r.Context())
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var itemID int64
	var name, startDate string
	var startUsage float64
	var usageInterval *float64
	// Lock the item before the rule so edits and completions share the same lock order.
	err = tx.QueryRow(r.Context(), `SELECT i.id FROM maintenance_items i JOIN maintenance_rules r ON r.item_id=i.id AND r.workspace_id=i.workspace_id WHERE r.id=$1 AND r.workspace_id=$2 FOR UPDATE OF i`, id, workspace).Scan(&itemID)
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	err = tx.QueryRow(r.Context(), `SELECT name,start_date::text,start_usage::float8,usage_interval::float8 FROM maintenance_rules WHERE id=$1 AND workspace_id=$2 AND active FOR UPDATE`, id, workspace).Scan(&name, &startDate, &startUsage, &usageInterval)
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	if input.CompletedAt < startDate || (usageInterval != nil && (input.UsageValue == nil || *input.UsageValue < startUsage)) {
		writeError(w, 422, "tanggal dan penggunaan harus sesuai acuan aturan; aturan penggunaan memerlukan nilai penggunaan")
		return
	}
	if input.UsageValue != nil {
		var inconsistent bool
		err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM maintenance_histories WHERE item_id=$1 AND workspace_id=$2 AND usage_value IS NOT NULL AND ((completed_at<$3::date AND usage_value>$4) OR (completed_at>$3::date AND usage_value<$4)))`, itemID, workspace, input.CompletedAt, *input.UsageValue).Scan(&inconsistent)
		if err != nil {
			maintenanceDBError(w, err)
			return
		}
		if inconsistent {
			writeError(w, 422, "nilai penggunaan tidak sesuai urutan riwayat")
			return
		}
	}
	var historyID int64
	err = tx.QueryRow(r.Context(), `INSERT INTO maintenance_histories(workspace_id,item_id,rule_id,rule_name,completed_at,cost,vendor,usage_value,notes) VALUES($1,$2,$3,$4,$5::date,$6,$7,$8,$9) RETURNING id`, workspace, itemID, id, name, input.CompletedAt, input.Cost, input.Vendor, input.UsageValue, input.Notes).Scan(&historyID)
	if err == nil {
		_, err = tx.Exec(r.Context(), `UPDATE maintenance_items SET current_usage=CASE WHEN $3::numeric IS NULL THEN current_usage ELSE GREATEST(COALESCE(current_usage,0),$3::numeric) END,updated_at=now() WHERE id=$1 AND workspace_id=$2`, itemID, workspace, input.UsageValue)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		maintenanceDBError(w, err)
		return
	}
	writeJSON(w, 201, envelope{"data": envelope{"id": historyID}})
}
