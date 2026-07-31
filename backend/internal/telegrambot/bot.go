package telegrambot

import (
	"bytes"
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const pendingLifetime = 30 * time.Minute

type Config struct {
	Token        string
	PairingCode  string
	LocalUserID  int64
	TimezoneName string
}

type Bot struct {
	db          *pgxpool.Pool
	logger      *slog.Logger
	client      *http.Client
	token       string
	pairingCode string
	localUserID int64
	location    *time.Location
	baseURL     string
}

type apiResponse[T any] struct {
	OK          bool   `json:"ok"`
	Result      T      `json:"result"`
	Description string `json:"description"`
}

type update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *message       `json:"message"`
	CallbackQuery *callbackQuery `json:"callback_query"`
}

type message struct {
	MessageID int64  `json:"message_id"`
	Chat      chat   `json:"chat"`
	From      user   `json:"from"`
	Text      string `json:"text"`
}

type chat struct {
	ID int64 `json:"id"`
}

type user struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (u user) displayName() string {
	return strings.TrimSpace(strings.Join([]string{u.FirstName, u.LastName}, " "))
}

type callbackQuery struct {
	ID      string   `json:"id"`
	From    user     `json:"from"`
	Message *message `json:"message"`
	Data    string   `json:"data"`
}

type inlineKeyboardMarkup struct {
	InlineKeyboard [][]inlineKeyboardButton `json:"inline_keyboard"`
}

type inlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type linkedUser struct {
	UserID            int64
	ActiveWorkspaceID sql.NullInt64
}

type namedItem struct {
	ID   int64
	Name string
}

func New(db *pgxpool.Pool, logger *slog.Logger, cfg Config) (*Bot, error) {
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, errors.New("telegram bot token is required")
	}
	if strings.TrimSpace(cfg.PairingCode) == "" {
		return nil, errors.New("telegram pairing code is required")
	}
	if cfg.LocalUserID <= 0 {
		return nil, errors.New("telegram local user ID must be positive")
	}
	location, err := time.LoadLocation(cfg.TimezoneName)
	if err != nil {
		return nil, fmt.Errorf("load telegram timezone: %w", err)
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return &Bot{
		db:     db,
		logger: logger,
		client: &http.Client{
			Timeout: 40 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, address string) (net.Conn, error) {
					return dialer.DialContext(ctx, "tcp4", address)
				},
			},
		},
		token:       cfg.Token,
		pairingCode: cfg.PairingCode,
		localUserID: cfg.LocalUserID,
		location:    location,
		baseURL:     "https://api.telegram.org/bot" + cfg.Token,
	}, nil
}

func (b *Bot) Run(ctx context.Context) error {
	if err := b.setCommands(ctx); err != nil {
		b.logger.Warn("failed to register Telegram commands", "error", err)
	}
	b.logger.Info("Telegram bot started", "mode", "long-polling")

	var offset int64
	for {
		updates, err := b.getUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			b.logger.Warn("Telegram polling failed", "error", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Second):
				continue
			}
		}
		for _, item := range updates {
			if item.UpdateID >= offset {
				offset = item.UpdateID + 1
			}
			if err := b.handleUpdate(ctx, item); err != nil {
				b.logger.Error("Telegram update failed", "update_id", item.UpdateID, "error", err)
			}
		}
	}
}

func (b *Bot) getUpdates(ctx context.Context, offset int64) ([]update, error) {
	query := url.Values{}
	query.Set("offset", strconv.FormatInt(offset, 10))
	query.Set("timeout", "25")
	query.Set("allowed_updates", `["message","callback_query"]`)
	endpoint := b.baseURL + "/getUpdates?" + query.Encode()
	return doAPI[[]update](ctx, b, http.MethodGet, endpoint, nil)
}

func (b *Bot) setCommands(ctx context.Context) error {
	commands := []map[string]string{
		{"command": "ruang", "description": "Pilih ruang keuangan"},
		{"command": "ruangaktif", "description": "Lihat ruang yang aktif"},
		{"command": "keluar", "description": "Catat pengeluaran"},
		{"command": "masuk", "description": "Catat pemasukan"},
		{"command": "hariini", "description": "Ringkasan transaksi hari ini"},
		{"command": "bulanini", "description": "Ringkasan transaksi bulan ini"},
		{"command": "saldo", "description": "Lihat saldo rekening"},
		{"command": "batal", "description": "Batalkan transaksi tertunda"},
		{"command": "help", "description": "Lihat petunjuk"},
	}
	_, err := doAPI[json.RawMessage](ctx, b, http.MethodPost, b.baseURL+"/setMyCommands", map[string]any{"commands": commands})
	return err
}

func (b *Bot) handleUpdate(ctx context.Context, item update) error {
	if item.Message != nil && item.Message.Text != "" {
		return b.handleMessage(ctx, item.Message)
	}
	if item.CallbackQuery != nil {
		return b.handleCallback(ctx, item.CallbackQuery)
	}
	return nil
}

func (b *Bot) handleMessage(ctx context.Context, msg *message) error {
	command, args := parseCommand(msg.Text)
	if command == "start" {
		return b.handleStart(ctx, msg, args)
	}

	linked, err := b.linkedUser(ctx, msg.From.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return b.sendText(ctx, msg.Chat.ID, "Akun Telegram ini belum terhubung.\n\nGunakan:\n/start KODE_PASANGAN")
	}
	if err != nil {
		return err
	}
	if err := b.touchTelegramUser(ctx, msg); err != nil {
		b.logger.Warn("failed to update Telegram profile", "telegram_user_id", msg.From.ID, "error", err)
	}

	switch command {
	case "help", "":
		return b.sendText(ctx, msg.Chat.ID, helpText())
	case "ruang":
		return b.sendWorkspacePicker(ctx, msg.Chat.ID, msg.From.ID, linked.UserID)
	case "ruangaktif":
		return b.sendActiveWorkspace(ctx, msg.Chat.ID, msg.From.ID)
	case "keluar":
		return b.startTransaction(ctx, msg.Chat.ID, msg.From.ID, linked, "expense", args)
	case "masuk":
		return b.startTransaction(ctx, msg.Chat.ID, msg.From.ID, linked, "income", args)
	case "hariini":
		return b.sendSummary(ctx, msg.Chat.ID, linked, "today")
	case "bulanini":
		return b.sendSummary(ctx, msg.Chat.ID, linked, "month")
	case "saldo":
		return b.sendBalances(ctx, msg.Chat.ID, linked)
	case "batal":
		_, err := b.db.Exec(ctx, `DELETE FROM telegram_pending_transactions WHERE telegram_user_id=$1`, msg.From.ID)
		if err != nil {
			return err
		}
		return b.sendText(ctx, msg.Chat.ID, "✅ Semua transaksi tertunda dibatalkan.")
	default:
		return b.sendText(ctx, msg.Chat.ID, "Perintah belum dikenali.\n\n"+helpText())
	}
}

func (b *Bot) handleStart(ctx context.Context, msg *message, args string) error {
	if _, err := b.linkedUser(ctx, msg.From.ID); err == nil {
		if err := b.touchTelegramUser(ctx, msg); err != nil {
			return err
		}
		return b.sendText(ctx, msg.Chat.ID, "Akun Telegram sudah terhubung dengan Hubby.\n\n"+helpText())
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	provided := strings.TrimSpace(args)
	if subtle.ConstantTimeCompare([]byte(provided), []byte(b.pairingCode)) != 1 {
		return b.sendText(ctx, msg.Chat.ID, "Kode pasangan tidak valid.\n\nGunakan /start diikuti kode dari konfigurasi backend.")
	}

	result, err := b.db.Exec(ctx, `
		INSERT INTO telegram_users(
			telegram_user_id,user_id,chat_id,username,display_name,active_workspace_id
		)
		SELECT $1,u.id,$2,$3,$4,u.current_workspace_id
		FROM users u WHERE u.id=$5
		ON CONFLICT(telegram_user_id) DO UPDATE SET
			user_id=EXCLUDED.user_id,
			chat_id=EXCLUDED.chat_id,
			username=EXCLUDED.username,
			display_name=EXCLUDED.display_name,
			updated_at=now()
	`, msg.From.ID, msg.Chat.ID, msg.From.Username, msg.From.displayName(), b.localUserID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return b.sendText(ctx, msg.Chat.ID, "User Hubby yang dikonfigurasi tidak ditemukan.")
	}
	return b.sendText(ctx, msg.Chat.ID, "✅ Telegram berhasil dihubungkan dengan Hubby.\n\nRuang aktif mengikuti pilihan awal di website. Gunakan /ruang untuk menggantinya.\n\n"+helpText())
}

func (b *Bot) linkedUser(ctx context.Context, telegramUserID int64) (linkedUser, error) {
	var linked linkedUser
	err := b.db.QueryRow(ctx, `
		SELECT user_id,active_workspace_id
		FROM telegram_users WHERE telegram_user_id=$1
	`, telegramUserID).Scan(&linked.UserID, &linked.ActiveWorkspaceID)
	return linked, err
}

func (b *Bot) touchTelegramUser(ctx context.Context, msg *message) error {
	_, err := b.db.Exec(ctx, `
		UPDATE telegram_users
		SET chat_id=$2,username=$3,display_name=$4,updated_at=now()
		WHERE telegram_user_id=$1
	`, msg.From.ID, msg.Chat.ID, msg.From.Username, msg.From.displayName())
	return err
}

func (b *Bot) sendWorkspacePicker(ctx context.Context, chatID, telegramUserID, userID int64) error {
	rows, err := b.db.Query(ctx, `
		SELECT w.id,w.name
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id=w.id
		WHERE wm.user_id=$1
		ORDER BY w.created_at,w.id
	`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var items []namedItem
	for rows.Next() {
		var item namedItem
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(items) == 0 {
		return b.sendText(ctx, chatID, "Belum ada ruang keuangan yang dapat dipilih.")
	}

	keyboard := keyboardForItems(items, "ws:")
	return b.sendMessage(ctx, chatID, "Pilih ruang keuangan yang akan digunakan:", keyboard)
}

func (b *Bot) sendActiveWorkspace(ctx context.Context, chatID, telegramUserID int64) error {
	var name string
	err := b.db.QueryRow(ctx, `
		SELECT w.name
		FROM telegram_users tu
		JOIN workspaces w ON w.id=tu.active_workspace_id
		JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.user_id=tu.user_id
		WHERE tu.telegram_user_id=$1
	`, telegramUserID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return b.sendText(ctx, chatID, "Belum ada ruang aktif. Gunakan /ruang untuk memilih.")
	}
	if err != nil {
		return err
	}
	return b.sendText(ctx, chatID, "Ruang aktif: "+name)
}

func (b *Bot) startTransaction(ctx context.Context, chatID, telegramUserID int64, linked linkedUser, kind, args string) error {
	amountText, description, ok := strings.Cut(strings.TrimSpace(args), " ")
	if !ok || strings.TrimSpace(description) == "" {
		command := "/keluar"
		if kind == "income" {
			command = "/masuk"
		}
		return b.sendText(ctx, chatID, "Format:\n"+command+" NOMINAL KETERANGAN\n\nContoh:\n"+command+" 25000 makan siang")
	}
	amount, err := parseAmount(amountText)
	if err != nil {
		return b.sendText(ctx, chatID, "Nominal tidak valid. Contoh yang didukung: 25000, 25.000, 25rb, atau 1,5jt.")
	}
	if !linked.ActiveWorkspaceID.Valid {
		return b.sendText(ctx, chatID, "Belum ada ruang aktif. Gunakan /ruang terlebih dahulu.")
	}

	var workspaceName string
	err = b.db.QueryRow(ctx, `
		SELECT w.name
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id=w.id
		WHERE w.id=$1 AND wm.user_id=$2
	`, linked.ActiveWorkspaceID.Int64, linked.UserID).Scan(&workspaceName)
	if errors.Is(err, pgx.ErrNoRows) {
		return b.sendText(ctx, chatID, "Ruang aktif sudah tidak tersedia. Gunakan /ruang untuk memilih kembali.")
	}
	if err != nil {
		return err
	}

	var pendingID int64
	err = b.db.QueryRow(ctx, `
		INSERT INTO telegram_pending_transactions(
			telegram_user_id,workspace_id,type,amount,description,occurred_at,expires_at
		) VALUES($1,$2,$3,$4,$5,$6,$7)
		RETURNING id
	`, telegramUserID, linked.ActiveWorkspaceID.Int64, kind, amount,
		strings.TrimSpace(description), time.Now().In(b.location).Format("2006-01-02"),
		time.Now().Add(pendingLifetime)).Scan(&pendingID)
	if err != nil {
		return err
	}
	return b.sendCategoryPicker(ctx, chatID, telegramUserID, pendingID, kind, workspaceName)
}

func (b *Bot) sendCategoryPicker(ctx context.Context, chatID, telegramUserID, pendingID int64, kind, workspaceName string) error {
	items, err := b.pendingItems(ctx, telegramUserID, pendingID, "categories", kind)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return b.sendText(ctx, chatID, "Tidak ada kategori yang sesuai di ruang "+workspaceName+". Tambahkan kategori melalui website terlebih dahulu.")
	}
	return b.sendMessage(ctx, chatID, "Ruang: "+workspaceName+"\nPilih kategori:", keyboardForItems(items, "cat:"+strconv.FormatInt(pendingID, 10)+":"))
}

func (b *Bot) sendAccountPicker(ctx context.Context, chatID, telegramUserID, pendingID int64) error {
	items, err := b.pendingItems(ctx, telegramUserID, pendingID, "accounts", "")
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return b.sendText(ctx, chatID, "Tidak ada rekening di ruang ini. Tambahkan rekening melalui website terlebih dahulu.")
	}
	return b.sendMessage(ctx, chatID, "Pilih rekening yang digunakan:", keyboardForItems(items, "acc:"+strconv.FormatInt(pendingID, 10)+":"))
}

func (b *Bot) pendingItems(ctx context.Context, telegramUserID, pendingID int64, table, kind string) ([]namedItem, error) {
	var query string
	var args []any
	switch table {
	case "categories":
		query = `
			SELECT c.id,c.name
			FROM categories c
			JOIN telegram_pending_transactions p ON p.workspace_id=c.workspace_id
			WHERE p.id=$1 AND p.telegram_user_id=$2 AND p.expires_at>now() AND c.type=$3
			ORDER BY c.id
		`
		args = []any{pendingID, telegramUserID, kind}
	case "accounts":
		query = `
			SELECT a.id,a.name
			FROM accounts a
			JOIN telegram_pending_transactions p ON p.workspace_id=a.workspace_id
			WHERE p.id=$1 AND p.telegram_user_id=$2 AND p.expires_at>now()
			ORDER BY a.id
		`
		args = []any{pendingID, telegramUserID}
	default:
		return nil, errors.New("unsupported pending item type")
	}

	rows, err := b.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []namedItem
	for rows.Next() {
		var item namedItem
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (b *Bot) handleCallback(ctx context.Context, callback *callbackQuery) error {
	if callback.Message == nil {
		return b.answerCallback(ctx, callback.ID, "Pesan asal tidak tersedia.")
	}
	acknowledgement := ""
	if strings.HasPrefix(callback.Data, "save:") {
		acknowledgement = "Menyimpan transaksi..."
	}
	if err := b.answerCallback(ctx, callback.ID, acknowledgement); err != nil {
		// A callback acknowledgement can expire or fail independently. Continue
		// processing so a temporary Telegram API issue never drops a transaction.
		b.logger.Warn("failed to acknowledge Telegram callback", "callback_data", callback.Data, "error", err)
	}

	linked, err := b.linkedUser(ctx, callback.From.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return b.sendText(ctx, callback.Message.Chat.ID, "Hubungkan akun dengan /start terlebih dahulu.")
	}
	if err != nil {
		return b.reportCallbackFailure(ctx, callback, err)
	}

	parts := strings.Split(callback.Data, ":")
	if len(parts) < 2 {
		return b.sendText(ctx, callback.Message.Chat.ID, "Pilihan tidak valid.")
	}

	var response string
	switch parts[0] {
	case "ws":
		response, err = b.selectWorkspace(ctx, callback.From.ID, linked.UserID, parts)
	case "cat":
		response, err = b.selectCategory(ctx, callback.Message.Chat.ID, callback.From.ID, parts)
	case "acc":
		response, err = b.selectAccount(ctx, callback.Message.Chat.ID, callback.From.ID, parts)
	case "save":
		response, err = b.savePending(ctx, callback.From.ID, parts)
	case "cancel":
		response, err = b.cancelPending(ctx, callback.From.ID, parts)
	default:
		response = "Pilihan tidak dikenali."
	}
	if err != nil {
		return b.reportCallbackFailure(ctx, callback, err)
	}
	if response != "" {
		return b.sendText(ctx, callback.Message.Chat.ID, response)
	}
	return nil
}

func (b *Bot) reportCallbackFailure(ctx context.Context, callback *callbackQuery, cause error) error {
	b.logger.Error(
		"Telegram callback failed",
		"callback_data", callback.Data,
		"telegram_user_id", callback.From.ID,
		"error", cause,
	)
	message := "⚠️ Pilihan tidak dapat diproses. Silakan coba lagi."
	if strings.HasPrefix(callback.Data, "save:") {
		message = "⚠️ Transaksi belum tersimpan karena terjadi kesalahan. Data sementara tetap tersedia; silakan tekan Simpan lagi."
	}
	if err := b.sendText(ctx, callback.Message.Chat.ID, message); err != nil {
		return errors.Join(cause, err)
	}
	return nil
}

func (b *Bot) selectWorkspace(ctx context.Context, telegramUserID, userID int64, parts []string) (string, error) {
	if len(parts) != 2 {
		return "Ruang tidak valid.", nil
	}
	workspaceID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "Ruang tidak valid.", nil
	}
	var name string
	err = b.db.QueryRow(ctx, `
		UPDATE telegram_users tu
		SET active_workspace_id=$2,updated_at=now()
		WHERE tu.telegram_user_id=$1 AND tu.user_id=$3
		  AND EXISTS (
			SELECT 1 FROM workspace_members wm
			WHERE wm.workspace_id=$2 AND wm.user_id=tu.user_id
		  )
		RETURNING (SELECT name FROM workspaces WHERE id=$2)
	`, telegramUserID, workspaceID, userID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "Ruang tidak tersedia untuk akun ini.", nil
	}
	if err != nil {
		return "", err
	}
	return "✅ Ruang aktif: " + name, nil
}

func (b *Bot) selectCategory(ctx context.Context, chatID, telegramUserID int64, parts []string) (string, error) {
	pendingID, itemID, ok := parseCallbackIDs(parts)
	if !ok {
		return "Kategori tidak valid.", nil
	}
	result, err := b.db.Exec(ctx, `
		UPDATE telegram_pending_transactions p
		SET category_id=$3
		WHERE p.id=$1 AND p.telegram_user_id=$2 AND p.expires_at>now()
		  AND EXISTS (
			SELECT 1 FROM categories c
			WHERE c.id=$3 AND c.workspace_id=p.workspace_id AND c.type=p.type
		  )
	`, pendingID, telegramUserID, itemID)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() == 0 {
		return "Transaksi sudah kedaluwarsa atau kategori tidak valid.", nil
	}
	if err := b.sendAccountPicker(ctx, chatID, telegramUserID, pendingID); err != nil {
		return "", err
	}
	return "", nil
}

func (b *Bot) selectAccount(ctx context.Context, chatID, telegramUserID int64, parts []string) (string, error) {
	pendingID, itemID, ok := parseCallbackIDs(parts)
	if !ok {
		return "Rekening tidak valid.", nil
	}
	result, err := b.db.Exec(ctx, `
		UPDATE telegram_pending_transactions p
		SET account_id=$3
		WHERE p.id=$1 AND p.telegram_user_id=$2 AND p.expires_at>now()
		  AND EXISTS (
			SELECT 1 FROM accounts a WHERE a.id=$3 AND a.workspace_id=p.workspace_id
		  )
	`, pendingID, telegramUserID, itemID)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() == 0 {
		return "Transaksi sudah kedaluwarsa atau rekening tidak valid.", nil
	}
	return "", b.sendConfirmation(ctx, chatID, telegramUserID, pendingID)
}

func (b *Bot) sendConfirmation(ctx context.Context, chatID, telegramUserID, pendingID int64) error {
	var workspace, kind, category, account, description string
	var amount int64
	var occurred time.Time
	err := b.db.QueryRow(ctx, `
		SELECT w.name,p.type::text,p.amount,p.description,p.occurred_at,c.name,a.name
		FROM telegram_pending_transactions p
		JOIN workspaces w ON w.id=p.workspace_id
		JOIN categories c ON c.id=p.category_id AND c.workspace_id=p.workspace_id
		JOIN accounts a ON a.id=p.account_id AND a.workspace_id=p.workspace_id
		WHERE p.id=$1 AND p.telegram_user_id=$2 AND p.expires_at>now()
	`, pendingID, telegramUserID).Scan(&workspace, &kind, &amount, &description, &occurred, &category, &account)
	if errors.Is(err, pgx.ErrNoRows) {
		return b.sendText(ctx, chatID, "Transaksi tertunda sudah kedaluwarsa. Silakan catat kembali.")
	}
	if err != nil {
		return err
	}
	kindLabel := "Pengeluaran"
	if kind == "income" {
		kindLabel = "Pemasukan"
	}
	text := fmt.Sprintf(
		"Konfirmasi transaksi\n\nRuang: %s\nJenis: %s\nJumlah: %s\nKategori: %s\nRekening: %s\nTanggal: %s\nKeterangan: %s",
		workspace, kindLabel, formatRupiah(amount), category, account,
		occurred.In(b.location).Format("02-01-2006"), description,
	)
	keyboard := inlineKeyboardMarkup{InlineKeyboard: [][]inlineKeyboardButton{{
		{Text: "✅ Simpan", CallbackData: "save:" + strconv.FormatInt(pendingID, 10)},
		{Text: "❌ Batal", CallbackData: "cancel:" + strconv.FormatInt(pendingID, 10)},
	}}}
	return b.sendMessage(ctx, chatID, text, keyboard)
}

func (b *Bot) savePending(ctx context.Context, telegramUserID int64, parts []string) (string, error) {
	if len(parts) != 2 {
		return "Transaksi tidak valid.", nil
	}
	pendingID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "Transaksi tidak valid.", nil
	}

	tx, err := b.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var workspaceID, userID, amount int64
	var categoryID, accountID sql.NullInt64
	var kind, description string
	var occurred time.Time
	err = tx.QueryRow(ctx, `
		SELECT p.workspace_id,tu.user_id,p.type::text,p.amount,p.description,p.occurred_at,
		       p.category_id,p.account_id
		FROM telegram_pending_transactions p
		JOIN telegram_users tu ON tu.telegram_user_id=p.telegram_user_id
		WHERE p.id=$1 AND p.telegram_user_id=$2 AND p.expires_at>now()
		FOR UPDATE OF p
	`, pendingID, telegramUserID).Scan(
		&workspaceID, &userID, &kind, &amount, &description, &occurred, &categoryID, &accountID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return "Transaksi sudah diproses atau kedaluwarsa.", nil
	}
	if err != nil {
		return "", err
	}
	if !categoryID.Valid || !accountID.Valid {
		return "Kategori atau rekening belum dipilih.", nil
	}

	var workspace, category, account string
	err = tx.QueryRow(ctx, `
		SELECT w.name,c.name,a.name
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.user_id=$2
		JOIN categories c ON c.id=$3 AND c.workspace_id=w.id AND c.type=$5
		JOIN accounts a ON a.id=$4 AND a.workspace_id=w.id
		WHERE w.id=$1
	`, workspaceID, userID, categoryID.Int64, accountID.Int64, kind).Scan(&workspace, &category, &account)
	if errors.Is(err, pgx.ErrNoRows) {
		return "Ruang, kategori, atau rekening sudah tidak valid.", nil
	}
	if err != nil {
		return "", err
	}

	var transactionID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions(
			workspace_id,type,category_id,account_id,amount,description,occurred_at,is_debt_payment
		) VALUES($1,$2,$3,$4,$5,$6,$7,FALSE)
		RETURNING id
	`, workspaceID, kind, categoryID.Int64, accountID.Int64, amount, description, occurred).Scan(&transactionID)
	if err != nil {
		return "", err
	}
	delta := transactionBalanceDelta(kind, amount)
	result, err := tx.Exec(ctx, `
		UPDATE accounts
		SET current_balance=current_balance + $2,
		    updated_at=now()
		WHERE id=$1 AND workspace_id=$3
	`, accountID.Int64, delta, workspaceID)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() != 1 {
		return "", fmt.Errorf("update transaction account balance: account not found")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM telegram_pending_transactions WHERE id=$1`, pendingID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"✅ Transaksi #%d tersimpan\n\nRuang: %s\nJumlah: %s\nKategori: %s\nRekening: %s",
		transactionID, workspace, formatRupiah(amount), category, account,
	), nil
}

func transactionBalanceDelta(kind string, amount int64) int64 {
	if kind == "expense" {
		return -amount
	}
	return amount
}

func (b *Bot) cancelPending(ctx context.Context, telegramUserID int64, parts []string) (string, error) {
	if len(parts) != 2 {
		return "Transaksi tidak valid.", nil
	}
	pendingID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "Transaksi tidak valid.", nil
	}
	result, err := b.db.Exec(ctx, `
		DELETE FROM telegram_pending_transactions WHERE id=$1 AND telegram_user_id=$2
	`, pendingID, telegramUserID)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() == 0 {
		return "Transaksi sudah diproses atau kedaluwarsa.", nil
	}
	return "❌ Transaksi dibatalkan.", nil
}

func (b *Bot) sendSummary(ctx context.Context, chatID int64, linked linkedUser, period string) error {
	if !linked.ActiveWorkspaceID.Valid {
		return b.sendText(ctx, chatID, "Belum ada ruang aktif. Gunakan /ruang terlebih dahulu.")
	}
	now := time.Now().In(b.location)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, b.location)
	label := "Hari ini"
	if period == "month" {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, b.location)
		label = "Bulan ini"
	}
	end := now.Add(time.Second)

	var workspace string
	var income, expense int64
	err := b.db.QueryRow(ctx, `
		SELECT w.name,
		       COALESCE(SUM(t.amount) FILTER (WHERE t.type='income'),0)::bigint,
		       COALESCE(SUM(t.amount) FILTER (WHERE t.type='expense'),0)::bigint
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.user_id=$2
		LEFT JOIN transactions t ON t.workspace_id=w.id
			AND t.occurred_at >= $3::date AND t.occurred_at <= $4::date
		WHERE w.id=$1
		GROUP BY w.name
	`, linked.ActiveWorkspaceID.Int64, linked.UserID, start, end).Scan(&workspace, &income, &expense)
	if errors.Is(err, pgx.ErrNoRows) {
		return b.sendText(ctx, chatID, "Ruang aktif sudah tidak tersedia. Gunakan /ruang untuk memilih kembali.")
	}
	if err != nil {
		return err
	}
	return b.sendText(ctx, chatID, fmt.Sprintf(
		"%s — %s\n\nPemasukan: %s\nPengeluaran: %s\nSelisih: %s",
		label, workspace, formatRupiah(income), formatRupiah(expense), formatRupiah(income-expense),
	))
}

func (b *Bot) sendBalances(ctx context.Context, chatID int64, linked linkedUser) error {
	if !linked.ActiveWorkspaceID.Valid {
		return b.sendText(ctx, chatID, "Belum ada ruang aktif. Gunakan /ruang terlebih dahulu.")
	}
	rows, err := b.db.Query(ctx, `
		SELECT w.name,a.name,a.current_balance
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.user_id=$2
		JOIN accounts a ON a.workspace_id=w.id
		WHERE w.id=$1
		ORDER BY a.id
	`, linked.ActiveWorkspaceID.Int64, linked.UserID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var workspace string
	var lines []string
	for rows.Next() {
		var account string
		var balance int64
		if err := rows.Scan(&workspace, &account, &balance); err != nil {
			return err
		}
		lines = append(lines, account+": "+formatRupiah(balance))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(lines) == 0 {
		return b.sendText(ctx, chatID, "Tidak ada rekening pada ruang aktif.")
	}
	return b.sendText(ctx, chatID, "Saldo — "+workspace+"\n\n"+strings.Join(lines, "\n"))
}

func (b *Bot) sendText(ctx context.Context, chatID int64, text string) error {
	return b.sendMessage(ctx, chatID, text, nil)
}

func (b *Bot) sendMessage(ctx context.Context, chatID int64, text string, keyboard any) error {
	payload := map[string]any{"chat_id": chatID, "text": text}
	if keyboard != nil {
		payload["reply_markup"] = keyboard
	}
	_, err := doAPI[json.RawMessage](ctx, b, http.MethodPost, b.baseURL+"/sendMessage", payload)
	return err
}

func (b *Bot) answerCallback(ctx context.Context, callbackID, text string) error {
	payload := map[string]any{"callback_query_id": callbackID}
	if text != "" {
		payload["text"] = text
	}
	_, err := doAPI[json.RawMessage](ctx, b, http.MethodPost, b.baseURL+"/answerCallbackQuery", payload)
	return err
}

func doAPI[T any](ctx context.Context, b *Bot, method, endpoint string, payload any) (T, error) {
	var zero T
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return zero, err
		}
		body = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return zero, errors.New("create Telegram request failed")
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := b.client.Do(request)
	if err != nil {
		return zero, errors.New("Telegram request failed: " + strings.ReplaceAll(err.Error(), b.token, "[redacted]"))
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return zero, err
	}
	var decoded apiResponse[T]
	if err := json.Unmarshal(data, &decoded); err != nil {
		return zero, fmt.Errorf("decode Telegram response (HTTP %d): %w", response.StatusCode, err)
	}
	if !decoded.OK {
		return zero, fmt.Errorf("Telegram API rejected request (HTTP %d): %s", response.StatusCode, decoded.Description)
	}
	return decoded.Result, nil
}

func keyboardForItems(items []namedItem, prefix string) inlineKeyboardMarkup {
	rows := make([][]inlineKeyboardButton, 0, (len(items)+1)/2)
	for index := 0; index < len(items); index += 2 {
		row := []inlineKeyboardButton{{
			Text:         items[index].Name,
			CallbackData: prefix + strconv.FormatInt(items[index].ID, 10),
		}}
		if index+1 < len(items) {
			row = append(row, inlineKeyboardButton{
				Text:         items[index+1].Name,
				CallbackData: prefix + strconv.FormatInt(items[index+1].ID, 10),
			})
		}
		rows = append(rows, row)
	}
	return inlineKeyboardMarkup{InlineKeyboard: rows}
}

func parseCallbackIDs(parts []string) (int64, int64, bool) {
	if len(parts) != 3 {
		return 0, 0, false
	}
	pendingID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	itemID, err := strconv.ParseInt(parts[2], 10, 64)
	return pendingID, itemID, err == nil
}

func parseCommand(text string) (string, string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", text
	}
	first, rest, _ := strings.Cut(text, " ")
	command := strings.TrimPrefix(first, "/")
	command, _, _ = strings.Cut(command, "@")
	return strings.ToLower(command), strings.TrimSpace(rest)
}

func parseAmount(value string) (int64, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "rp")
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("empty amount")
	}

	multiplier := float64(1)
	suffixes := []struct {
		value      string
		multiplier float64
	}{
		{"juta", 1_000_000},
		{"jt", 1_000_000},
		{"rb", 1_000},
		{"k", 1_000},
	}
	hasSuffix := false
	for _, suffix := range suffixes {
		if strings.HasSuffix(value, suffix.value) {
			value = strings.TrimSpace(strings.TrimSuffix(value, suffix.value))
			multiplier = suffix.multiplier
			hasSuffix = true
			break
		}
	}

	if hasSuffix {
		numberText := strings.ReplaceAll(value, ",", ".")
		number, err := strconv.ParseFloat(numberText, 64)
		amount := int64(number * multiplier)
		if err != nil || number <= 0 || amount <= 0 {
			return 0, errors.New("invalid amount")
		}
		return amount, nil
	}

	value = strings.NewReplacer(".", "", ",", "", "_", "").Replace(value)
	amount, err := strconv.ParseInt(value, 10, 64)
	if err != nil || amount <= 0 {
		return 0, errors.New("invalid amount")
	}
	return amount, nil
}

func formatRupiah(amount int64) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}
	digits := strconv.FormatInt(amount, 10)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "." + digits[index:]
	}
	return sign + "Rp" + digits
}

func helpText() string {
	return `Perintah Hubby:

/ruang — pilih ruang keuangan
/ruangaktif — lihat ruang aktif
/keluar 25000 makan siang
/masuk 5jt gaji bulanan
/hariini — ringkasan hari ini
/bulanini — ringkasan bulan ini
/saldo — daftar saldo rekening
/batal — batalkan transaksi tertunda

Setiap transaksi akan meminta kategori, rekening, dan konfirmasi sebelum disimpan.`
}
