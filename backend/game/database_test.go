package game

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

var testDBCounter int64

// setupTestDB creates a temporary SQLite database for testing and returns
// a cleanup function to remove it afterwards.
func setupTestDB(t *testing.T) func() {
	t.Helper()
	n := atomic.AddInt64(&testDBCounter, 1)
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("zhajinhua_test_%d_%d.db", os.Getpid(), n))
	InitDB(tmpFile)
	return func() {
		if DB != nil {
			DB.Close()
		}
		os.Remove(tmpFile)
		os.Remove(tmpFile + "-wal")
		os.Remove(tmpFile + "-shm")
	}
}

func TestInitDB(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Verify tables exist by querying them
	tables := []string{"users", "transactions", "platform_fees", "daily_bonus", "config", "recharge_txns"}
	for _, table := range tables {
		_, err := DB.Exec(fmt.Sprintf("SELECT COUNT(*) FROM %s", table))
		if err != nil {
			t.Errorf("table %s should exist: %v", table, err)
		}
	}
}

func TestGetOrCreateUser_New(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	chips, err := GetOrCreateUser("player1", "Alice")
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}
	if chips != 1500 {
		t.Errorf("new user should have 1500 chips including signup bonus, got %d", chips)
	}
}

func TestGetOrCreateUser_Existing(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _ = GetOrCreateUser("player1", "Alice")
	// Update chips to a different value
	_ = UpdateUserChips("player1", 2000)

	chips, err := GetOrCreateUser("player1", "Alice")
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}
	if chips != 2000 {
		t.Errorf("existing user should have 2000 chips, got %d", chips)
	}
}

func TestUpdateUserChips(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _ = GetOrCreateUser("player1", "Alice")
	err := UpdateUserChips("player1", 5000)
	if err != nil {
		t.Fatalf("UpdateUserChips failed: %v", err)
	}

	chips, _ := GetOrCreateUser("player1", "Alice")
	if chips != 5000 {
		t.Errorf("expected 5000 chips, got %d", chips)
	}
}

func TestUpdateUserChips_NonExistent(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	err := UpdateUserChips("nonexistent", 100)
	if err == nil {
		t.Error("expected error for nonexistent player")
	}
}

func TestRechargeUser(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _ = GetOrCreateUser("player1", "Alice")
	newChips, err := RechargeUser("player1", 500)
	if err != nil {
		t.Fatalf("RechargeUser failed: %v", err)
	}
	if newChips != 2000 {
		t.Errorf("expected 2000 chips after recharge, got %d", newChips)
	}

	// Verify via direct query
	chips, _ := GetOrCreateUser("player1", "Alice")
	if chips != 2000 {
		t.Errorf("expected 2000 chips in DB, got %d", chips)
	}

	// Verify transaction recorded
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE player_id = ? AND type = 'recharge'", "player1").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 recharge transaction, got %d", count)
	}
}

func TestClaimDailyBonus(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _ = GetOrCreateUser("player1", "Alice")

	// First claim
	newChips, remaining, err := ClaimDailyBonus("player1")
	if err != nil {
		t.Fatalf("first claim failed: %v", err)
	}
	if newChips != 2000 {
		t.Errorf("expected 2000 chips after first bonus, got %d", newChips)
	}
	if remaining != 2 {
		t.Errorf("expected 2 remaining claims, got %d", remaining)
	}

	// Second claim
	newChips, remaining, err = ClaimDailyBonus("player1")
	if err != nil {
		t.Fatalf("second claim failed: %v", err)
	}
	if newChips != 2500 {
		t.Errorf("expected 2500 chips after second bonus, got %d", newChips)
	}
	if remaining != 1 {
		t.Errorf("expected 1 remaining claim, got %d", remaining)
	}

	// Third claim
	newChips, remaining, err = ClaimDailyBonus("player1")
	if err != nil {
		t.Fatalf("third claim failed: %v", err)
	}
	if newChips != 3000 {
		t.Errorf("expected 3000 chips after third bonus, got %d", newChips)
	}
	if remaining != 0 {
		t.Errorf("expected 0 remaining claims, got %d", remaining)
	}
}

func TestSignupBonusDoesNotConsumeDailyClaims(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	chips, err := GetOrCreateUser("player1", "Alice")
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}
	if chips != 1500 {
		t.Fatalf("expected signup bonus chips to be 1500, got %d", chips)
	}

	claimed, remaining, canClaim, err := GetDailyBonusStatus("player1")
	if err != nil {
		t.Fatalf("GetDailyBonusStatus failed: %v", err)
	}
	if claimed != 0 {
		t.Fatalf("expected signup bonus not to count as daily claim, got %d", claimed)
	}
	if remaining != 3 || !canClaim {
		t.Fatalf("expected 3 daily claims remaining, got remaining=%d canClaim=%v", remaining, canClaim)
	}
}

func TestDailyBonusStatusUsesUTCDay(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _ = GetOrCreateUser("player1", "Alice")

	yesterdayUTC := utcNow().Add(-24 * time.Hour)
	_, err := DB.Exec(
		"INSERT INTO daily_bonus (player_id, claimed_at, amount, bonus_type) VALUES (?, ?, ?, ?)",
		"player1", yesterdayUTC, 500, "daily",
	)
	if err != nil {
		t.Fatalf("failed to seed previous UTC day bonus: %v", err)
	}

	claimed, remaining, canClaim, err := GetDailyBonusStatus("player1")
	if err != nil {
		t.Fatalf("GetDailyBonusStatus failed: %v", err)
	}
	if claimed != 0 {
		t.Fatalf("expected previous UTC day claim not to count today, got %d", claimed)
	}
	if remaining != 3 || !canClaim {
		t.Fatalf("expected full daily claims after UTC rollover, got remaining=%d canClaim=%v", remaining, canClaim)
	}
}

func TestClaimDailyBonus_MaxExceeded(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _ = GetOrCreateUser("player1", "Alice")

	// Claim 3 times
	for i := 0; i < 3; i++ {
		_, _, err := ClaimDailyBonus("player1")
		if err != nil {
			t.Fatalf("claim %d failed: %v", i+1, err)
		}
	}

	// 4th claim should fail
	_, _, err := ClaimDailyBonus("player1")
	if err == nil {
		t.Error("expected error on 4th daily bonus claim")
	}
}

func TestSettleGame(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _ = GetOrCreateUser("winner1", "Winner")
	_, _ = GetOrCreateUser("loser1", "Loser1")
	_, _ = GetOrCreateUser("loser2", "Loser2")

	err := SettleGame("winner1", 500, []string{"loser1", "loser2"}, []int{200, 300}, "room1")
	if err != nil {
		t.Fatalf("SettleGame failed: %v", err)
	}

	// Check winner chips: 1000 + 500 = 1500
	// New users start with 1500 because signup bonus does not consume daily claims.
	winnerChips, _ := GetOrCreateUser("winner1", "Winner")
	if winnerChips != 2000 {
		t.Errorf("expected winner chips 2000, got %d", winnerChips)
	}

	// Check loser1 chips: 1500 - 200 = 1300
	loser1Chips, _ := GetOrCreateUser("loser1", "Loser1")
	if loser1Chips != 1300 {
		t.Errorf("expected loser1 chips 1300, got %d", loser1Chips)
	}

	// Check loser2 chips: 1500 - 300 = 1200
	loser2Chips, _ := GetOrCreateUser("loser2", "Loser2")
	if loser2Chips != 1200 {
		t.Errorf("expected loser2 chips 1200, got %d", loser2Chips)
	}

	// Check transactions recorded
	var txCount int
	DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE room_id = 'room1'").Scan(&txCount)
	if txCount != 3 { // 1 win + 2 loss
		t.Errorf("expected 3 transactions, got %d", txCount)
	}
}

func TestSettleGame_LengthMismatch(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	err := SettleGame("w", 100, []string{"a", "b"}, []int{50}, "r1")
	if err == nil {
		t.Error("expected error for losers/lossAmounts length mismatch")
	}
}

func TestConfig_SetGetRoundTrip(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	err := SetConfig("theme", "dark")
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	val := GetConfig("theme", "light")
	if val != "dark" {
		t.Errorf("expected 'dark', got %q", val)
	}
}

func TestConfig_DefaultValue(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	val := GetConfig("nonexistent", "default_val")
	if val != "default_val" {
		t.Errorf("expected 'default_val', got %q", val)
	}
}

func TestConfig_Upsert(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_ = SetConfig("key1", "value1")
	_ = SetConfig("key1", "value2")

	val := GetConfig("key1", "")
	if val != "value2" {
		t.Errorf("expected 'value2' after upsert, got %q", val)
	}
}

func TestGetAllConfig(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_ = SetConfig("key1", "val1")
	_ = SetConfig("key2", "val2")

	all := GetAllConfig()
	if len(all) != 2 {
		t.Errorf("expected 2 config entries, got %d", len(all))
	}
	if all["key1"] != "val1" {
		t.Errorf("expected key1=val1, got %q", all["key1"])
	}
	if all["key2"] != "val2" {
		t.Errorf("expected key2=val2, got %q", all["key2"])
	}
}

func TestAdminSetChips(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _ = GetOrCreateUser("player1", "Alice")

	err := AdminSetChips("player1", 9999)
	if err != nil {
		t.Fatalf("AdminSetChips failed: %v", err)
	}

	chips, _ := GetOrCreateUser("player1", "Alice")
	if chips != 9999 {
		t.Errorf("expected 9999 chips, got %d", chips)
	}

	// Check transaction recorded
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE player_id = 'player1' AND type = 'admin_adjust'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 admin_adjust transaction, got %d", count)
	}
}

func TestAdminSetChips_NonExistent(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	err := AdminSetChips("nonexistent", 100)
	if err == nil {
		t.Error("expected error for nonexistent player")
	}
}

func TestSubmitRecharge(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _ = GetOrCreateUser("player1", "Alice")
	err := SubmitRecharge("player1", "0xabc123", "evm", 100, 1000)
	if err != nil {
		t.Fatalf("SubmitRecharge failed: %v", err)
	}

	// Verify record exists
	var status string
	DB.QueryRow("SELECT status FROM recharge_txns WHERE tx_hash = '0xabc123'").Scan(&status)
	if status != "pending" {
		t.Errorf("expected status 'pending', got %q", status)
	}
}

func TestSubmitRecharge_DuplicateTxHash(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _ = GetOrCreateUser("player1", "Alice")
	_ = SubmitRecharge("player1", "0xdup", "evm", 100, 1000)
	err := SubmitRecharge("player1", "0xdup", "evm", 200, 2000)
	if err == nil {
		t.Error("expected error for duplicate tx_hash")
	}
}

func TestConfirmRecharge(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _ = GetOrCreateUser("player1", "Alice") // 1500 chips including signup bonus
	_ = SubmitRecharge("player1", "0xtx1", "evm", 50, 500)

	playerID, points, err := ConfirmRecharge("0xtx1")
	if err != nil {
		t.Fatalf("ConfirmRecharge failed: %v", err)
	}
	if playerID != "player1" {
		t.Errorf("expected playerID 'player1', got %q", playerID)
	}
	if points != 500 {
		t.Errorf("expected 500 points, got %d", points)
	}

	// Check chips updated
	chips, _ := GetOrCreateUser("player1", "Alice")
	if chips != 2000 {
		t.Errorf("expected 2000 chips after confirm, got %d", chips)
	}

	// Check status updated
	var status string
	DB.QueryRow("SELECT status FROM recharge_txns WHERE tx_hash = '0xtx1'").Scan(&status)
	if status != "confirmed" {
		t.Errorf("expected status 'confirmed', got %q", status)
	}
}

func TestConfirmRecharge_AlreadyConfirmed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _ = GetOrCreateUser("player1", "Alice")
	_ = SubmitRecharge("player1", "0xtx2", "evm", 50, 500)
	_, _, _ = ConfirmRecharge("0xtx2")

	_, _, err := ConfirmRecharge("0xtx2")
	if err == nil {
		t.Error("expected error for already confirmed recharge")
	}
}

func TestConfirmRecharge_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _, err := ConfirmRecharge("0xnonexistent")
	if err == nil {
		t.Error("expected error for nonexistent tx_hash")
	}
}

func TestGetDashboardStats(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, _ = GetOrCreateUser("player1", "Alice")
	_, _ = GetOrCreateUser("player2", "Bob")

	stats := GetDashboardStats()

	totalUsers, ok := stats["totalUsers"]
	if !ok {
		t.Fatal("missing totalUsers in stats")
	}
	if totalUsers.(int) != 2 {
		t.Errorf("expected totalUsers 2, got %v", totalUsers)
	}

	// Check that expected keys exist
	expectedKeys := []string{"totalUsers", "totalPlatformFees", "totalRecharges", "totalBonus", "todayActiveUsers", "totalGames"}
	for _, key := range expectedKeys {
		if _, ok := stats[key]; !ok {
			t.Errorf("missing key %q in dashboard stats", key)
		}
	}
}
