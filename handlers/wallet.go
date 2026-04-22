package handlers

import (
	"net/http"

	"fintech_wallet_api/database"
	"fintech_wallet_api/models"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- Request / Response types ---

// TransactionRequest is the JSON body for deposit and withdraw endpoints.
type TransactionRequest struct {
	UserID   string          `json:"user_id" binding:"required"`
	Amount   decimal.Decimal `json:"amount" binding:"required"`
	Currency string          `json:"currency"` // defaults to USD
}

// WalletResponse is the standard JSON response for wallet operations.
type WalletResponse struct {
	Message string             `json:"message"`
	Wallet  *models.UserWallet `json:"wallet,omitempty"`
}

// --- Handlers ---

// Deposit adds funds to a user's wallet. Creates the wallet if it doesn't exist.
//
// POST /api/v1/deposit
// Body: { "user_id": "abc", "amount": "100.00" }
func Deposit(c *gin.Context) {
	var req TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be positive"})
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	// Use an explicit transaction on the primary.
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var wallet models.UserWallet
	result := tx.Where("user_id = ?", req.UserID).First(&wallet)

	if result.Error == gorm.ErrRecordNotFound {
		// Create a new wallet for this user.
		wallet = models.UserWallet{
			UserID:   req.UserID,
			Balance:  req.Amount,
			Currency: currency,
		}
		if err := tx.Create(&wallet).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create wallet"})
			return
		}
	} else if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	} else {
		// Add to existing balance.
		wallet.Balance = wallet.Balance.Add(req.Amount)
		if err := tx.Save(&wallet).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update balance"})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, WalletResponse{
		Message: "deposit successful",
		Wallet:  &wallet,
	})
}

// Withdraw deducts funds from a user's wallet using row-level locking to
// prevent race conditions (double-spending).
//
// POST /api/v1/withdraw
// Body: { "user_id": "abc", "amount": "50.00" }
//
// How row-level locking works:
//   - SELECT ... FOR UPDATE acquires an exclusive lock on the wallet row.
//   - Any concurrent transaction attempting to read the same row FOR UPDATE
//     will BLOCK until this transaction commits or rolls back.
//   - This guarantees that two simultaneous withdrawals cannot both see
//     the same balance and each deduct from it — preventing double-spending.
func Withdraw(c *gin.Context) {
	var req TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be positive"})
		return
	}

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var wallet models.UserWallet

	// Lock the row — prevents concurrent modifications.
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", req.UserID).
		First(&wallet).Error

	if err == gorm.ErrRecordNotFound {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	} else if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Check sufficient funds.
	if wallet.Balance.LessThan(req.Amount) {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "insufficient funds",
			"balance": wallet.Balance.String(),
		})
		return
	}

	// Deduct the amount.
	wallet.Balance = wallet.Balance.Sub(req.Amount)
	if err := tx.Save(&wallet).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update balance"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, WalletResponse{
		Message: "withdrawal successful",
		Wallet:  &wallet,
	})
}

// GetBalance reads the wallet balance from the replica via DBResolver.
//
// GET /api/v1/balance/:user_id
func GetBalance(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	var wallet models.UserWallet
	// This is a read query — DBResolver routes it to the replica automatically.
	err := database.DB.Where("user_id = ?", userID).First(&wallet).Error

	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, WalletResponse{
		Message: "balance retrieved",
		Wallet:  &wallet,
	})
}
