package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/utils"
)

type LoanService struct {
	db *mongo.Database
}

func NewLoanService(db *mongo.Database) *LoanService {
	return &LoanService{db: db}
}

type CreateLoanRequest struct {
	LenderID   primitive.ObjectID `json:"lenderId"`
	BorrowerID primitive.ObjectID `json:"borrowerId"`
	AmountPLN  float64            `json:"amountPLN"`
	Note       *string            `json:"note,omitempty"`
	DueDate    *time.Time         `json:"dueDate,omitempty"`
}

type CreateLoanPaymentRequest struct {
	LoanID    primitive.ObjectID `json:"loanId"`
	AmountPLN float64            `json:"amountPLN"`
	PaidAt    time.Time          `json:"paidAt"`
	Note      *string            `json:"note,omitempty"`
}

type Balance struct {
	UserID primitive.ObjectID `json:"userId"`
	Owed   float64            `json:"owed"`   // Money this user owes to others
	Owing  float64            `json:"owing"`  // Money others owe to this user
}

type PairwiseBalance struct {
	FromUserId   primitive.ObjectID   `json:"fromUserId"`
	ToUserId     primitive.ObjectID   `json:"toUserId"`
	FromUserName string               `json:"fromUserName"`
	ToUserName   string               `json:"toUserName"`
	NetAmount    primitive.Decimal128 `json:"netAmount"`
}

// CreateLoan creates a new loan
func (s *LoanService) CreateLoan(ctx context.Context, req CreateLoanRequest) (*models.Loan, error) {
	if req.LenderID == req.BorrowerID {
		return nil, errors.New("lender and borrower cannot be the same user")
	}

	if req.AmountPLN <= 0 {
		return nil, errors.New("loan amount must be positive")
	}

	// Verify users exist
	for _, userID := range []primitive.ObjectID{req.LenderID, req.BorrowerID} {
		var user models.User
		err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, errors.New("user not found")
			}
			return nil, fmt.Errorf("database error: %w", err)
		}
	}

	amountDec, err := utils.DecimalFromFloat(req.AmountPLN)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	loan := models.Loan{
		ID:         primitive.NewObjectID(),
		LenderID:   req.LenderID,
		BorrowerID: req.BorrowerID,
		AmountPLN:  amountDec,
		Note:       req.Note,
		DueDate:    req.DueDate,
		Status:     "open",
		CreatedAt:  time.Now(),
	}

	_, err = s.db.Collection("loans").InsertOne(ctx, loan)
	if err != nil {
		return nil, fmt.Errorf("failed to create loan: %w", err)
	}

	return &loan, nil
}

// CreateLoanPayment records a loan repayment
func (s *LoanService) CreateLoanPayment(ctx context.Context, req CreateLoanPaymentRequest) (*models.LoanPayment, error) {
	// Get loan
	var loan models.Loan
	err := s.db.Collection("loans").FindOne(ctx, bson.M{"_id": req.LoanID}).Decode(&loan)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("loan not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if loan.Status == "settled" {
		return nil, errors.New("loan is already settled")
	}

	if req.AmountPLN <= 0 {
		return nil, errors.New("payment amount must be positive")
	}

	// Get total paid so far
	totalPaid, err := s.getTotalPaidForLoan(ctx, req.LoanID)
	if err != nil {
		return nil, err
	}

	loanAmount, _ := utils.DecimalToFloat(loan.AmountPLN)
	remaining := loanAmount - totalPaid

	if req.AmountPLN > remaining {
		return nil, fmt.Errorf("payment amount (%.2f) exceeds remaining balance (%.2f)", req.AmountPLN, remaining)
	}

	amountDec, err := utils.DecimalFromFloat(req.AmountPLN)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	payment := models.LoanPayment{
		ID:        primitive.NewObjectID(),
		LoanID:    req.LoanID,
		AmountPLN: amountDec,
		PaidAt:    req.PaidAt,
		Note:      req.Note,
	}

	_, err = s.db.Collection("loan_payments").InsertOne(ctx, payment)
	if err != nil {
		return nil, fmt.Errorf("failed to create loan payment: %w", err)
	}

	// Update loan status
	newTotalPaid := totalPaid + req.AmountPLN
	var newStatus string
	if newTotalPaid >= loanAmount {
		newStatus = "settled"
	} else {
		newStatus = "partial"
	}

	_, err = s.db.Collection("loans").UpdateOne(
		ctx,
		bson.M{"_id": req.LoanID},
		bson.M{"$set": bson.M{"status": newStatus}},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update loan status: %w", err)
	}

	return &payment, nil
}

// GetBalances calculates pairwise balances for all users
func (s *LoanService) GetBalances(ctx context.Context) ([]PairwiseBalance, error) {
	// Get all loans
	cursor, err := s.db.Collection("loans").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var loans []models.Loan
	if err := cursor.All(ctx, &loans); err != nil {
		return nil, fmt.Errorf("failed to decode loans: %w", err)
	}

	// Calculate net balances
	balances := make(map[string]float64) // key: "lenderID-borrowerID"

	for _, loan := range loans {
		if loan.Status == "settled" {
			continue
		}

		loanAmount, _ := utils.DecimalToFloat(loan.AmountPLN)
		totalPaid, err := s.getTotalPaidForLoan(ctx, loan.ID)
		if err != nil {
			return nil, err
		}

		remaining := loanAmount - totalPaid
		if remaining <= 0 {
			continue
		}

		key := fmt.Sprintf("%s-%s", loan.BorrowerID.Hex(), loan.LenderID.Hex())
		reverseKey := fmt.Sprintf("%s-%s", loan.LenderID.Hex(), loan.BorrowerID.Hex())

		// Net out reverse debts
		if reverseBalance, exists := balances[reverseKey]; exists {
			if remaining > reverseBalance {
				balances[key] = remaining - reverseBalance
				delete(balances, reverseKey)
			} else if remaining < reverseBalance {
				balances[reverseKey] = reverseBalance - remaining
			} else {
				delete(balances, reverseKey)
			}
		} else {
			balances[key] += remaining
		}
	}

	// Get all users for name lookup
	usersCursor, err := s.db.Collection("users").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	defer usersCursor.Close(ctx)

	var users []models.User
	if err := usersCursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	userMap := make(map[primitive.ObjectID]string)
	for _, user := range users {
		userMap[user.ID] = user.Name
	}

	// Convert to pairwise balances
	result := []PairwiseBalance{}
	for key, amount := range balances {
		// Parse the IDs properly
		parts := parseBalanceKey(key)
		if len(parts) == 2 {
			fromID, _ := primitive.ObjectIDFromHex(parts[0])
			toID, _ := primitive.ObjectIDFromHex(parts[1])

			amountDec, _ := utils.DecimalFromFloat(amount)

			result = append(result, PairwiseBalance{
				FromUserId:   fromID,
				ToUserId:     toID,
				FromUserName: userMap[fromID],
				ToUserName:   userMap[toID],
				NetAmount:    amountDec,
			})
		}
	}

	return result, nil
}

type LoanWithNames struct {
	models.Loan
	FromUserName string `json:"fromUserName"`
	ToUserName   string `json:"toUserName"`
}

// GetLoans retrieves all loans with user names
func (s *LoanService) GetLoans(ctx context.Context) ([]LoanWithNames, error) {
	cursor, err := s.db.Collection("loans").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var loans []models.Loan
	if err := cursor.All(ctx, &loans); err != nil {
		return nil, fmt.Errorf("failed to decode loans: %w", err)
	}

	// Get all users for name lookup
	usersCursor, err := s.db.Collection("users").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	defer usersCursor.Close(ctx)

	var users []models.User
	if err := usersCursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	userMap := make(map[primitive.ObjectID]string)
	for _, user := range users {
		userMap[user.ID] = user.Name
	}

	// Enrich with user names
	result := make([]LoanWithNames, len(loans))
	for i, loan := range loans {
		result[i] = LoanWithNames{
			Loan:         loan,
			FromUserName: userMap[loan.LenderID],
			ToUserName:   userMap[loan.BorrowerID],
		}
	}

	return result, nil
}

// GetUserBalance calculates balance for a specific user - returns pairwise balances
func (s *LoanService) GetUserBalance(ctx context.Context, userID primitive.ObjectID) ([]PairwiseBalance, error) {
	balances, err := s.GetBalances(ctx)
	if err != nil {
		return nil, err
	}

	// Filter balances for this user
	result := []PairwiseBalance{}
	for _, b := range balances {
		if b.FromUserId == userID || b.ToUserId == userID {
			result = append(result, b)
		}
	}

	return result, nil
}

// Helper functions
func (s *LoanService) getTotalPaidForLoan(ctx context.Context, loanID primitive.ObjectID) (float64, error) {
	cursor, err := s.db.Collection("loan_payments").Find(ctx, bson.M{"loan_id": loanID})
	if err != nil {
		return 0, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var payments []models.LoanPayment
	if err := cursor.All(ctx, &payments); err != nil {
		return 0, fmt.Errorf("failed to decode payments: %w", err)
	}

	total := 0.0
	for _, p := range payments {
		amount, _ := utils.DecimalToFloat(p.AmountPLN)
		total += amount
	}

	return total, nil
}

func parseBalanceKey(key string) []string {
	result := make([]string, 2)
	parts := []rune{}
	partIdx := 0

	for _, c := range key {
		if c == '-' {
			result[partIdx] = string(parts)
			partIdx++
			parts = []rune{}
			if partIdx >= 2 {
				break
			}
		} else {
			parts = append(parts, c)
		}
	}

	if partIdx == 1 && len(parts) > 0 {
		result[1] = string(parts)
	}

	return result
}