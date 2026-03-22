package game

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB is the singleton database connection.
var DB *sql.DB

// InitDB opens (or creates) the SQLite database at the given path and
// ensures the required tables exist.
func InitDB(path string) {
	var err error
	DB, err = sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := DB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Fatalf("failed to set WAL mode: %v", err)
	}

	createTables := `
	CREATE TABLE IF NOT EXISTS users (
		id         INTEGER  PRIMARY KEY,
		player_id  TEXT     UNIQUE,
		name       TEXT,
		chips      INTEGER  DEFAULT 1000,
		created_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id         INTEGER  PRIMARY KEY,
		player_id  TEXT,
		type       TEXT,
		amount     INTEGER,
		room_id    TEXT,
		detail     TEXT,
		created_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS platform_fees (
		id         INTEGER  PRIMARY KEY,
		room_id    TEXT,
		winner_id  TEXT,
		amount     INTEGER,
		players    INTEGER,
		base_bet   INTEGER,
		created_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS daily_bonus (
		id         INTEGER  PRIMARY KEY,
		player_id  TEXT,
		claimed_at DATETIME,
		amount     INTEGER
	);

	CREATE TABLE IF NOT EXISTS config (
		key   TEXT PRIMARY KEY,
		value TEXT
	);

	CREATE TABLE IF NOT EXISTS recharge_txns (
		id         INTEGER  PRIMARY KEY,
		tx_hash    TEXT     UNIQUE,
		chain      TEXT,
		player_id  TEXT,
		amount     INTEGER,
		points     INTEGER,
		status     TEXT     DEFAULT 'pending',
		created_at DATETIME,
		verified_at DATETIME
	);
	`
	if _, err := DB.Exec(createTables); err != nil {
		log.Fatalf("failed to create tables: %v", err)
	}

	// Migrate: add wallet columns if they don't exist
	migrations := []string{
		"ALTER TABLE users ADD COLUMN wallet_address TEXT DEFAULT ''",
		"ALTER TABLE users ADD COLUMN wallet_chain TEXT DEFAULT ''",
	}
	for _, m := range migrations {
		DB.Exec(m) // Ignore errors (column may already exist)
	}

	log.Println("database initialised:", path)
}

// GetOrCreateUser returns the chip count for the given player. If the player
// does not exist yet a new row is inserted with the default 1000 chips.
func GetOrCreateUser(playerID, name string) (chips int, err error) {
	row := DB.QueryRow("SELECT chips FROM users WHERE player_id = ?", playerID)
	if err = row.Scan(&chips); err == sql.ErrNoRows {
		chips = 1000
		now := time.Now()
		_, err = DB.Exec(
			"INSERT INTO users (player_id, name, chips, created_at) VALUES (?, ?, ?, ?)",
			playerID, name, chips, now,
		)
		if err != nil {
			return 0, fmt.Errorf("create user: %w", err)
		}
		return chips, nil
	}
	return chips, err
}

// UpdateUserChips sets the chip count for the given player.
func UpdateUserChips(playerID string, chips int) error {
	res, err := DB.Exec("UPDATE users SET chips = ? WHERE player_id = ?", chips, playerID)
	if err != nil {
		return fmt.Errorf("update chips: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("update chips: player %s not found", playerID)
	}
	return nil
}

// AddTransaction records a single transaction.
// Valid txType values: "recharge", "win", "loss", "bot_replenish".
func AddTransaction(playerID, txType string, amount int, roomID, detail string) error {
	_, err := DB.Exec(
		"INSERT INTO transactions (player_id, type, amount, room_id, detail, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		playerID, txType, amount, roomID, detail, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("add transaction: %w", err)
	}
	return nil
}

// RechargeUser adds the given amount of chips and records a recharge
// transaction. It returns the new chip total.
func RechargeUser(playerID string, amount int) (newChips int, err error) {
	tx, err := DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("recharge begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var current int
	row := tx.QueryRow("SELECT chips FROM users WHERE player_id = ?", playerID)
	if err = row.Scan(&current); err != nil {
		return 0, fmt.Errorf("recharge query: %w", err)
	}

	newChips = current + amount
	if _, err = tx.Exec("UPDATE users SET chips = ? WHERE player_id = ?", newChips, playerID); err != nil {
		return 0, fmt.Errorf("recharge update: %w", err)
	}

	now := time.Now()
	if _, err = tx.Exec(
		"INSERT INTO transactions (player_id, type, amount, room_id, detail, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		playerID, "recharge", amount, "", "recharge", now,
	); err != nil {
		return 0, fmt.Errorf("recharge tx record: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("recharge commit: %w", err)
	}
	return newChips, nil
}

// GetOrCreateUserWithWallet creates or retrieves a user with wallet info.
func GetOrCreateUserWithWallet(playerID, name, walletAddress, walletChain string) (chips int, err error) {
	row := DB.QueryRow("SELECT chips FROM users WHERE player_id = ?", playerID)
	if err = row.Scan(&chips); err == sql.ErrNoRows {
		chips = 1000
		now := time.Now()
		_, err = DB.Exec(
			"INSERT INTO users (player_id, name, chips, wallet_address, wallet_chain, created_at) VALUES (?, ?, ?, ?, ?, ?)",
			playerID, name, chips, walletAddress, walletChain, now,
		)
		if err != nil {
			return 0, fmt.Errorf("create user with wallet: %w", err)
		}
		return chips, nil
	}
	if err == nil && walletAddress != "" {
		// Update wallet info if provided
		DB.Exec("UPDATE users SET wallet_address = ?, wallet_chain = ? WHERE player_id = ?",
			walletAddress, walletChain, playerID)
	}
	return chips, err
}

// RecordRake records a platform rake fee.
func RecordRake(roomID, winnerID string, amount, players, baseBet int) error {
	_, err := DB.Exec(
		"INSERT INTO platform_fees (room_id, winner_id, amount, players, base_bet, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		roomID, winnerID, amount, players, baseBet, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("record rake: %w", err)
	}
	return nil
}

// GetDailyBonusInfo returns how many times the player has claimed the daily bonus today.
func GetDailyBonusInfo(playerID string) (claimed int, err error) {
	row := DB.QueryRow(
		"SELECT COUNT(*) FROM daily_bonus WHERE player_id = ? AND DATE(claimed_at) = DATE('now')",
		playerID,
	)
	if err = row.Scan(&claimed); err != nil {
		return 0, fmt.Errorf("get daily bonus info: %w", err)
	}
	return claimed, nil
}

// ClaimDailyBonus grants 500 chips to the player if they have remaining claims today (max 3).
// It returns the new chip total and the number of remaining claims.
func ClaimDailyBonus(playerID string) (newChips int, remainingClaims int, err error) {
	const maxClaims = 3
	const bonusAmount = 500

	tx, err := DB.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("bonus begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var claimed int
	row := tx.QueryRow(
		"SELECT COUNT(*) FROM daily_bonus WHERE player_id = ? AND DATE(claimed_at) = DATE('now')",
		playerID,
	)
	if err = row.Scan(&claimed); err != nil {
		return 0, 0, fmt.Errorf("bonus count query: %w", err)
	}

	if claimed >= maxClaims {
		return 0, 0, fmt.Errorf("No bonus remaining today")
	}

	// Insert bonus record
	now := time.Now()
	if _, err = tx.Exec(
		"INSERT INTO daily_bonus (player_id, claimed_at, amount) VALUES (?, ?, ?)",
		playerID, now, bonusAmount,
	); err != nil {
		return 0, 0, fmt.Errorf("bonus insert: %w", err)
	}

	// Add chips
	var current int
	row = tx.QueryRow("SELECT chips FROM users WHERE player_id = ?", playerID)
	if err = row.Scan(&current); err != nil {
		return 0, 0, fmt.Errorf("bonus chips query: %w", err)
	}

	newChips = current + bonusAmount
	if _, err = tx.Exec("UPDATE users SET chips = ? WHERE player_id = ?", newChips, playerID); err != nil {
		return 0, 0, fmt.Errorf("bonus chips update: %w", err)
	}

	// Record transaction
	if _, err = tx.Exec(
		"INSERT INTO transactions (player_id, type, amount, room_id, detail, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		playerID, "bonus", bonusAmount, "", "daily bonus", now,
	); err != nil {
		return 0, 0, fmt.Errorf("bonus tx record: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("bonus commit: %w", err)
	}

	remainingClaims = maxClaims - claimed - 1
	return newChips, remainingClaims, nil
}

// SettleGame records win/loss transactions for every participant and updates
// their chip balances inside a single database transaction.
// losers and lossAmounts must have the same length.
func SettleGame(winnerID string, winAmount int, losers []string, lossAmounts []int, roomID string) error {
	if len(losers) != len(lossAmounts) {
		return fmt.Errorf("settle: losers/lossAmounts length mismatch")
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("settle begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now()

	// --- winner ---
	var winnerChips int
	if err = tx.QueryRow("SELECT chips FROM users WHERE player_id = ?", winnerID).Scan(&winnerChips); err != nil {
		return fmt.Errorf("settle winner query: %w", err)
	}
	winnerChips += winAmount
	if _, err = tx.Exec("UPDATE users SET chips = ? WHERE player_id = ?", winnerChips, winnerID); err != nil {
		return fmt.Errorf("settle winner update: %w", err)
	}
	if _, err = tx.Exec(
		"INSERT INTO transactions (player_id, type, amount, room_id, detail, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		winnerID, "win", winAmount, roomID, "game win", now,
	); err != nil {
		return fmt.Errorf("settle winner tx: %w", err)
	}

	// --- losers ---
	for i, loserID := range losers {
		loss := lossAmounts[i]
		var loserChips int
		if err = tx.QueryRow("SELECT chips FROM users WHERE player_id = ?", loserID).Scan(&loserChips); err != nil {
			return fmt.Errorf("settle loser query %s: %w", loserID, err)
		}
		loserChips -= loss
		if loserChips < 0 {
			loserChips = 0
		}
		if _, err = tx.Exec("UPDATE users SET chips = ? WHERE player_id = ?", loserChips, loserID); err != nil {
			return fmt.Errorf("settle loser update %s: %w", loserID, err)
		}
		if _, err = tx.Exec(
			"INSERT INTO transactions (player_id, type, amount, room_id, detail, created_at) VALUES (?, ?, ?, ?, ?, ?)",
			loserID, "loss", loss, roomID, "game loss", now,
		); err != nil {
			return fmt.Errorf("settle loser tx %s: %w", loserID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("settle commit: %w", err)
	}
	return nil
}

// GetConfig returns the config value for the given key, or defaultVal if not set.
func GetConfig(key string, defaultVal string) string {
	var val string
	row := DB.QueryRow("SELECT value FROM config WHERE key = ?", key)
	if err := row.Scan(&val); err != nil {
		return defaultVal
	}
	return val
}

// SetConfig stores a config key-value pair (upsert).
func SetConfig(key, value string) error {
	_, err := DB.Exec(
		"INSERT INTO config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?",
		key, value, value,
	)
	return err
}

// GetAllConfig returns all config key-value pairs as a map.
func GetAllConfig() map[string]string {
	result := make(map[string]string)
	rows, err := DB.Query("SELECT key, value FROM config")
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) == nil {
			result[k] = v
		}
	}
	return result
}

// GetDashboardStats returns aggregate stats for the admin dashboard.
func GetDashboardStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Total users
	var totalUsers int
	DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	stats["totalUsers"] = totalUsers

	// Total platform fees
	var totalFees int
	DB.QueryRow("SELECT COALESCE(SUM(amount),0) FROM platform_fees").Scan(&totalFees)
	stats["totalPlatformFees"] = totalFees

	// Total recharges
	var totalRecharges int
	DB.QueryRow("SELECT COALESCE(SUM(amount),0) FROM transactions WHERE type='recharge'").Scan(&totalRecharges)
	stats["totalRecharges"] = totalRecharges

	// Total bonus given
	var totalBonus int
	DB.QueryRow("SELECT COALESCE(SUM(amount),0) FROM transactions WHERE type='bonus'").Scan(&totalBonus)
	stats["totalBonus"] = totalBonus

	// Today's active users (users who have transactions today)
	var todayActive int
	DB.QueryRow("SELECT COUNT(DISTINCT player_id) FROM transactions WHERE DATE(created_at) = DATE('now')").Scan(&todayActive)
	stats["todayActiveUsers"] = todayActive

	// Total games played
	var totalGames int
	DB.QueryRow("SELECT COUNT(*) FROM platform_fees").Scan(&totalGames)
	stats["totalGames"] = totalGames

	return stats
}

// GetUserList returns paginated user list for admin.
func GetUserList(offset, limit int) ([]map[string]interface{}, int) {
	var total int
	DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)

	rows, err := DB.Query("SELECT player_id, name, chips, wallet_address, wallet_chain, created_at FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, total
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var pid, name, wallet, chain string
		var chips int
		var createdAt time.Time
		if rows.Scan(&pid, &name, &chips, &wallet, &chain, &createdAt) == nil {
			users = append(users, map[string]interface{}{
				"playerId":  pid,
				"name":      name,
				"chips":     chips,
				"wallet":    wallet,
				"chain":     chain,
				"createdAt": createdAt.Format("2006-01-02 15:04"),
			})
		}
	}
	return users, total
}

// AdminSetChips sets chips for a specific player (admin override).
func AdminSetChips(playerID string, chips int) error {
	res, err := DB.Exec("UPDATE users SET chips = ? WHERE player_id = ?", chips, playerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("player not found: %s", playerID)
	}
	// Record admin adjustment transaction
	DB.Exec("INSERT INTO transactions (player_id, type, amount, room_id, detail, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		playerID, "admin_adjust", chips, "", "admin set chips", time.Now())
	return nil
}

// SubmitRecharge records a pending on-chain recharge transaction.
// Returns error if tx_hash already exists (prevents double-spend).
func SubmitRecharge(playerID, txHash, chain string, amount, points int) error {
	_, err := DB.Exec(
		"INSERT INTO recharge_txns (tx_hash, chain, player_id, amount, points, status, created_at) VALUES (?, ?, ?, ?, ?, 'pending', ?)",
		txHash, chain, playerID, amount, points, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("submit recharge: %w", err)
	}
	return nil
}

// ConfirmRecharge marks a recharge as confirmed, credits the points to the player.
// Returns error if already confirmed or not found. Uses a DB transaction to atomically
// update status, set verified_at, add chips to user, and record a transaction.
func ConfirmRecharge(txHash string) (playerID string, points int, err error) {
	tx, err := DB.Begin()
	if err != nil {
		return "", 0, fmt.Errorf("confirm recharge begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Fetch the recharge record
	var status string
	row := tx.QueryRow("SELECT player_id, points, status FROM recharge_txns WHERE tx_hash = ?", txHash)
	if err = row.Scan(&playerID, &points, &status); err != nil {
		return "", 0, fmt.Errorf("confirm recharge: tx not found: %w", err)
	}
	if status == "confirmed" {
		return "", 0, fmt.Errorf("confirm recharge: already confirmed")
	}

	// Update recharge status
	now := time.Now()
	if _, err = tx.Exec("UPDATE recharge_txns SET status = 'confirmed', verified_at = ? WHERE tx_hash = ?", now, txHash); err != nil {
		return "", 0, fmt.Errorf("confirm recharge update status: %w", err)
	}

	// Add chips to user
	var currentChips int
	row = tx.QueryRow("SELECT chips FROM users WHERE player_id = ?", playerID)
	if err = row.Scan(&currentChips); err != nil {
		return "", 0, fmt.Errorf("confirm recharge query user: %w", err)
	}
	newChips := currentChips + points
	if _, err = tx.Exec("UPDATE users SET chips = ? WHERE player_id = ?", newChips, playerID); err != nil {
		return "", 0, fmt.Errorf("confirm recharge update chips: %w", err)
	}

	// Record transaction
	if _, err = tx.Exec(
		"INSERT INTO transactions (player_id, type, amount, room_id, detail, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		playerID, "recharge", points, "", "on-chain recharge: "+txHash, now,
	); err != nil {
		return "", 0, fmt.Errorf("confirm recharge tx record: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return "", 0, fmt.Errorf("confirm recharge commit: %w", err)
	}
	return playerID, points, nil
}

// GetPendingRecharges returns all pending recharges for admin view.
func GetPendingRecharges() []map[string]interface{} {
	rows, err := DB.Query("SELECT tx_hash, chain, player_id, amount, points, status, created_at FROM recharge_txns WHERE status = 'pending' ORDER BY created_at DESC")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var txHash, chain, pid, status string
		var amount, points int
		var createdAt time.Time
		if rows.Scan(&txHash, &chain, &pid, &amount, &points, &status, &createdAt) == nil {
			result = append(result, map[string]interface{}{
				"txHash":    txHash,
				"chain":     chain,
				"playerId":  pid,
				"amount":    amount,
				"points":    points,
				"status":    status,
				"createdAt": createdAt.Format("2006-01-02 15:04:05"),
			})
		}
	}
	return result
}

// GetRechargeList returns all recharges ordered by created_at DESC with a limit.
func GetRechargeList(limit int) []map[string]interface{} {
	rows, err := DB.Query("SELECT tx_hash, chain, player_id, amount, points, status, created_at, verified_at FROM recharge_txns ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var txHash, chain, pid, status string
		var amount, points int
		var createdAt time.Time
		var verifiedAt sql.NullTime
		if rows.Scan(&txHash, &chain, &pid, &amount, &points, &status, &createdAt, &verifiedAt) == nil {
			entry := map[string]interface{}{
				"txHash":    txHash,
				"chain":     chain,
				"playerId":  pid,
				"amount":    amount,
				"points":    points,
				"status":    status,
				"createdAt": createdAt.Format("2006-01-02 15:04:05"),
			}
			if verifiedAt.Valid {
				entry["verifiedAt"] = verifiedAt.Time.Format("2006-01-02 15:04:05")
			} else {
				entry["verifiedAt"] = ""
			}
			result = append(result, entry)
		}
	}
	return result
}

// GetRecentTransactions returns the most recent transactions.
func GetRecentTransactions(limit int) []map[string]interface{} {
	rows, err := DB.Query("SELECT player_id, type, amount, room_id, detail, created_at FROM transactions ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var txns []map[string]interface{}
	for rows.Next() {
		var pid, txType, roomID, detail string
		var amount int
		var createdAt time.Time
		if rows.Scan(&pid, &txType, &amount, &roomID, &detail, &createdAt) == nil {
			txns = append(txns, map[string]interface{}{
				"playerId":  pid,
				"type":      txType,
				"amount":    amount,
				"roomId":    roomID,
				"detail":    detail,
				"createdAt": createdAt.Format("2006-01-02 15:04:05"),
			})
		}
	}
	return txns
}
