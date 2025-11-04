package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

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

type CompensationResult struct {
	CompensationsPerformed int     `json:"compensationsPerformed"`
	TotalAmountCompensated float64 `json:"totalAmountCompensated"`
}

type Balance struct {
	UserID primitive.ObjectID `json:"userId"`
	Owed   float64            `json:"owed"`  // Money this user owes to others
	Owing  float64            `json:"owing"` // Money others owe to this user
}

type PairwiseBalance struct {
	FromUserId        primitive.ObjectID   `json:"fromUserId"`
	ToUserId          primitive.ObjectID   `json:"toUserId"`
	FromUserName      string               `json:"fromUserName"`
	ToUserName        string               `json:"toUserName"`
	FromUserGroupID   *primitive.ObjectID  `json:"fromUserGroupId,omitempty"`
	FromUserGroupName *string              `json:"fromUserGroupName,omitempty"`
	ToUserGroupID     *primitive.ObjectID  `json:"toUserGroupId,omitempty"`
	ToUserGroupName   *string              `json:"toUserGroupName,omitempty"`
	NetAmount         primitive.Decimal128 `json:"netAmount"`
}

// CreateLoan creates a new loan with automatic debt offsetting
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

	// Perform group compensation on existing loans first
	_, err := s.PerformGroupCompensation(ctx)
	if err != nil {
		return nil, fmt.Errorf("group compensation failed: %w", err)
	}

	// Check for reverse debt (borrower owes lender)
	// Find open/partial loans where new borrower is the lender and new lender is the borrower
	cursor, err := s.db.Collection("loans").Find(ctx, bson.M{
		"lender_id":   req.BorrowerID,
		"borrower_id": req.LenderID,
		"status":      bson.M{"$in": []string{"open", "partial"}},
	})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var reverseLoans []models.Loan
	if err := cursor.All(ctx, &reverseLoans); err != nil {
		return nil, fmt.Errorf("failed to decode reverse loans: %w", err)
	}

	// If there are reverse debts, offset them
	remainingAmount := req.AmountPLN
	for _, reverseLoan := range reverseLoans {
		if remainingAmount <= 0 {
			break
		}

		// Calculate how much is remaining on the reverse loan
		reverseLoanAmount, _ := utils.DecimalToFloat(reverseLoan.AmountPLN)
		totalPaid, err := s.getTotalPaidForLoan(ctx, reverseLoan.ID)
		if err != nil {
			return nil, err
		}
		reverseRemaining := reverseLoanAmount - totalPaid

		if reverseRemaining <= 0 {
			continue
		}

		// Offset amount is the minimum of remaining on both sides
		offsetAmount := remainingAmount
		if reverseRemaining < offsetAmount {
			offsetAmount = reverseRemaining
		}

		// Create a payment to offset the reverse loan
		offsetAmountDec, _ := utils.DecimalFromFloat(offsetAmount)
		payment := models.LoanPayment{
			ID:        primitive.NewObjectID(),
			LoanID:    reverseLoan.ID,
			AmountPLN: offsetAmountDec,
			PaidAt:    time.Now(),
			Note:      getStringPtr("Automatyczne rozliczenie długów"),
		}

		_, err = s.db.Collection("loan_payments").InsertOne(ctx, payment)
		if err != nil {
			return nil, fmt.Errorf("failed to create offset payment: %w", err)
		}

		// Update reverse loan status
		newTotalPaid := totalPaid + offsetAmount
		var newStatus string
		if newTotalPaid >= reverseLoanAmount {
			newStatus = "settled"
		} else {
			newStatus = "partial"
		}

		_, err = s.db.Collection("loans").UpdateOne(
			ctx,
			bson.M{"_id": reverseLoan.ID},
			bson.M{"$set": bson.M{"status": newStatus}},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update reverse loan status: %w", err)
		}

		remainingAmount -= offsetAmount
	}

	// If there's still remaining amount, create the new loan
	if remainingAmount > 0 {
		amountDec, err := utils.DecimalFromFloat(remainingAmount)
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

	// All debt was offset, save settled loan to database
	amountDec, _ := utils.DecimalFromFloat(req.AmountPLN)

	// Append offset message to user's note if they provided one
	var settledNote *string
	if req.Note != nil && *req.Note != "" {
		combined := *req.Note + " (Całkowicie rozliczone z istniejącymi długami)"
		settledNote = &combined
	} else {
		settledNote = getStringPtr("Całkowicie rozliczone z istniejącymi długami")
	}

	settledLoan := models.Loan{
		ID:         primitive.NewObjectID(),
		LenderID:   req.LenderID,
		BorrowerID: req.BorrowerID,
		AmountPLN:  amountDec,
		Note:       settledNote,
		DueDate:    req.DueDate,
		Status:     "settled",
		CreatedAt:  time.Now(),
	}

	_, err = s.db.Collection("loans").InsertOne(ctx, settledLoan)
	if err != nil {
		return nil, fmt.Errorf("failed to create settled loan: %w", err)
	}

	return &settledLoan, nil
}

func getStringPtr(s string) *string {
	return &s
}

// PerformGroupCompensation performs debt compensation for group members
// When GroupMember1 owes External and External owes GroupMember2 (same group),
// the debts are offset without creating internal group debt
func (s *LoanService) PerformGroupCompensation(ctx context.Context) (*CompensationResult, error) {
	// Get all users with their group memberships
	usersCursor, err := s.db.Collection("users").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	defer usersCursor.Close(ctx)

	var users []models.User
	if err := usersCursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	// Build map: userID -> groupID
	userGroupMap := make(map[primitive.ObjectID]*primitive.ObjectID)
	for _, user := range users {
		userGroupMap[user.ID] = user.GroupID
	}

	// Get all open/partial loans, sorted by creation date (oldest first)
	cursor, err := s.db.Collection("loans").Find(ctx, bson.M{
		"status": bson.M{"$in": []string{"open", "partial"}},
	}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var loans []models.Loan
	if err := cursor.All(ctx, &loans); err != nil {
		return nil, fmt.Errorf("failed to decode loans: %w", err)
	}

	// Calculate remaining amounts for each loan
	type loanWithRemaining struct {
		loan      models.Loan
		remaining float64
	}

	loansWithRemaining := []loanWithRemaining{}
	for _, loan := range loans {
		loanAmount, _ := utils.DecimalToFloat(loan.AmountPLN)
		totalPaid, err := s.getTotalPaidForLoan(ctx, loan.ID)
		if err != nil {
			return nil, err
		}
		remaining := loanAmount - totalPaid
		if remaining > 0 {
			loansWithRemaining = append(loansWithRemaining, loanWithRemaining{
				loan:      loan,
				remaining: remaining,
			})
		}
	}

	compensationsPerformed := 0
	totalAmountCompensated := 0.0

	// Find compensation opportunities
	// Pattern: GroupMemberA owes External, External owes GroupMemberB (same group)
	// Loan1: Lender=External, Borrower=GroupMemberA
	// Loan2: Lender=GroupMemberB, Borrower=External
	for i := range loansWithRemaining {
		if loansWithRemaining[i].remaining <= 0 {
			continue
		}

		loan1 := loansWithRemaining[i].loan
		external := loan1.LenderID
		groupMemberA := loan1.BorrowerID

		// Check if groupMemberA is in a group
		groupMemberAGroupID := userGroupMap[groupMemberA]
		if groupMemberAGroupID == nil {
			continue
		}

		// Check if external is NOT in the same group (or no group)
		externalGroupID := userGroupMap[external]
		if externalGroupID != nil && *externalGroupID == *groupMemberAGroupID {
			continue
		}

		// Find loans where External is borrower and lender is in same group as GroupMemberA
		for j := range loansWithRemaining {
			if i == j || loansWithRemaining[j].remaining <= 0 {
				continue
			}

			loan2 := loansWithRemaining[j].loan

			// Check if External is the borrower in loan2
			if loan2.BorrowerID != external {
				continue
			}

			groupMemberB := loan2.LenderID
			groupMemberBGroupID := userGroupMap[groupMemberB]

			// Check if GroupMemberB is in the same group as GroupMemberA
			if groupMemberBGroupID == nil || *groupMemberBGroupID != *groupMemberAGroupID {
				continue
			}

			// Found a compensation opportunity!
			compensationAmount := loansWithRemaining[i].remaining
			if loansWithRemaining[j].remaining < compensationAmount {
				compensationAmount = loansWithRemaining[j].remaining
			}

			// Create payments with compensation note
			compensationNote := getStringPtr("Kompensacja grupowa")

			// Payment on loan1 (GroupMemberA -> External)
			amountDec, _ := utils.DecimalFromFloat(compensationAmount)
			payment1 := models.LoanPayment{
				ID:        primitive.NewObjectID(),
				LoanID:    loan1.ID,
				AmountPLN: amountDec,
				PaidAt:    time.Now(),
				Note:      compensationNote,
			}

			_, err = s.db.Collection("loan_payments").InsertOne(ctx, payment1)
			if err != nil {
				return nil, fmt.Errorf("failed to create compensation payment 1: %w", err)
			}

			// Update loan1 status
			newTotalPaid1, _ := s.getTotalPaidForLoan(ctx, loan1.ID)
			loanAmount1, _ := utils.DecimalToFloat(loan1.AmountPLN)
			var newStatus1 string
			if newTotalPaid1 >= loanAmount1 {
				newStatus1 = "settled"
			} else {
				newStatus1 = "partial"
			}

			_, err = s.db.Collection("loans").UpdateOne(
				ctx,
				bson.M{"_id": loan1.ID},
				bson.M{"$set": bson.M{"status": newStatus1}},
			)
			if err != nil {
				return nil, fmt.Errorf("failed to update loan1 status: %w", err)
			}

			// Payment on loan2 (External -> GroupMemberB)
			payment2 := models.LoanPayment{
				ID:        primitive.NewObjectID(),
				LoanID:    loan2.ID,
				AmountPLN: amountDec,
				PaidAt:    time.Now(),
				Note:      compensationNote,
			}

			_, err = s.db.Collection("loan_payments").InsertOne(ctx, payment2)
			if err != nil {
				return nil, fmt.Errorf("failed to create compensation payment 2: %w", err)
			}

			// Update loan2 status
			newTotalPaid2, _ := s.getTotalPaidForLoan(ctx, loan2.ID)
			loanAmount2, _ := utils.DecimalToFloat(loan2.AmountPLN)
			var newStatus2 string
			if newTotalPaid2 >= loanAmount2 {
				newStatus2 = "settled"
			} else {
				newStatus2 = "partial"
			}

			_, err = s.db.Collection("loans").UpdateOne(
				ctx,
				bson.M{"_id": loan2.ID},
				bson.M{"$set": bson.M{"status": newStatus2}},
			)
			if err != nil {
				return nil, fmt.Errorf("failed to update loan2 status: %w", err)
			}

			// Update remaining amounts
			loansWithRemaining[i].remaining -= compensationAmount
			loansWithRemaining[j].remaining -= compensationAmount

			compensationsPerformed++
			totalAmountCompensated += compensationAmount

			// If loan1 is fully settled, break to outer loop
			if loansWithRemaining[i].remaining <= 0 {
				break
			}
		}
	}

	return &CompensationResult{
		CompensationsPerformed: compensationsPerformed,
		TotalAmountCompensated: totalAmountCompensated,
	}, nil
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
	userGroupMap := make(map[primitive.ObjectID]*primitive.ObjectID)
	for _, user := range users {
		userMap[user.ID] = user.Name
		userGroupMap[user.ID] = user.GroupID
	}

	// Get all groups for name lookup
	groupsCursor, err := s.db.Collection("groups").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch groups: %w", err)
	}
	defer groupsCursor.Close(ctx)

	var groups []models.Group
	if err := groupsCursor.All(ctx, &groups); err != nil {
		return nil, fmt.Errorf("failed to decode groups: %w", err)
	}

	groupMap := make(map[primitive.ObjectID]string)
	for _, group := range groups {
		groupMap[group.ID] = group.Name
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

			balance := PairwiseBalance{
				FromUserId:   fromID,
				ToUserId:     toID,
				FromUserName: userMap[fromID],
				ToUserName:   userMap[toID],
				NetAmount:    amountDec,
			}

			// Add group information if user belongs to a group
			if groupID := userGroupMap[fromID]; groupID != nil {
				balance.FromUserGroupID = groupID
				groupName := groupMap[*groupID]
				balance.FromUserGroupName = &groupName
			}
			if groupID := userGroupMap[toID]; groupID != nil {
				balance.ToUserGroupID = groupID
				groupName := groupMap[*groupID]
				balance.ToUserGroupName = &groupName
			}

			result = append(result, balance)
		}
	}

	return result, nil
}

type LoanWithNames struct {
	models.Loan
	FromUserName      string               `json:"fromUserName"`
	ToUserName        string               `json:"toUserName"`
	FromUserGroupID   *primitive.ObjectID  `json:"fromUserGroupId,omitempty"`
	FromUserGroupName *string              `json:"fromUserGroupName,omitempty"`
	ToUserGroupID     *primitive.ObjectID  `json:"toUserGroupId,omitempty"`
	ToUserGroupName   *string              `json:"toUserGroupName,omitempty"`
	RemainingPLN      primitive.Decimal128 `json:"remainingPLN"`
}

type GetLoansOptions struct {
	SortBy string // createdAt, amountPLN, status, remainingPLN
	Order  string // asc, desc
	Limit  int
	Offset int
}

// GetLoans retrieves all loans with user names
func (s *LoanService) GetLoans(ctx context.Context) ([]LoanWithNames, error) {
	return s.GetLoansWithOptions(ctx, GetLoansOptions{
		SortBy: "createdAt",
		Order:  "desc",
		Limit:  0, // 0 means no limit
		Offset: 0,
	})
}

// GetLoansWithOptions retrieves all loans with user names, sorting, and pagination
func (s *LoanService) GetLoansWithOptions(ctx context.Context, opts GetLoansOptions) ([]LoanWithNames, error) {
	// Build sort order
	sortOrder := -1 // desc by default
	if opts.Order == "asc" {
		sortOrder = 1
	}

	sortField := "created_at"
	switch opts.SortBy {
	case "amountPLN":
		sortField = "amount_pln"
	case "status":
		sortField = "status"
	case "createdAt":
		sortField = "created_at"
	}

	// Build find options
	findOpts := options.Find()

	// Note: We can't sort by remainingPLN in the query since it's calculated
	// We'll need to sort after fetching if sortBy is remainingPLN
	if opts.SortBy != "remainingPLN" {
		findOpts.SetSort(bson.D{{Key: sortField, Value: sortOrder}})
	}

	if opts.Limit > 0 {
		findOpts.SetLimit(int64(opts.Limit))
		findOpts.SetSkip(int64(opts.Offset))
	}

	cursor, err := s.db.Collection("loans").Find(ctx, bson.M{}, findOpts)
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
	userGroupMap := make(map[primitive.ObjectID]*primitive.ObjectID)
	for _, user := range users {
		userMap[user.ID] = user.Name
		userGroupMap[user.ID] = user.GroupID
	}

	// Get all groups for name lookup
	groupsCursor, err := s.db.Collection("groups").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch groups: %w", err)
	}
	defer groupsCursor.Close(ctx)

	var groups []models.Group
	if err := groupsCursor.All(ctx, &groups); err != nil {
		return nil, fmt.Errorf("failed to decode groups: %w", err)
	}

	groupMap := make(map[primitive.ObjectID]string)
	for _, group := range groups {
		groupMap[group.ID] = group.Name
	}

	// Enrich with user names, group info, and remaining amounts
	result := make([]LoanWithNames, len(loans))
	for i, loan := range loans {
		// Calculate remaining amount
		loanAmount, _ := utils.DecimalToFloat(loan.AmountPLN)
		totalPaid, err := s.getTotalPaidForLoan(ctx, loan.ID)
		if err != nil {
			totalPaid = 0
		}
		remaining := loanAmount - totalPaid
		remainingDec, _ := utils.DecimalFromFloat(remaining)

		loanWithNames := LoanWithNames{
			Loan:         loan,
			FromUserName: userMap[loan.LenderID],
			ToUserName:   userMap[loan.BorrowerID],
			RemainingPLN: remainingDec,
		}

		// Add group information if user belongs to a group
		if groupID := userGroupMap[loan.LenderID]; groupID != nil {
			loanWithNames.FromUserGroupID = groupID
			groupName := groupMap[*groupID]
			loanWithNames.FromUserGroupName = &groupName
		}
		if groupID := userGroupMap[loan.BorrowerID]; groupID != nil {
			loanWithNames.ToUserGroupID = groupID
			groupName := groupMap[*groupID]
			loanWithNames.ToUserGroupName = &groupName
		}

		result[i] = loanWithNames
	}

	// Sort by remainingPLN if requested (can't be done in MongoDB query)
	if opts.SortBy == "remainingPLN" {
		sort.Slice(result, func(i, j int) bool {
			remI, _ := utils.DecimalToFloat(result[i].RemainingPLN)
			remJ, _ := utils.DecimalToFloat(result[j].RemainingPLN)
			if sortOrder == 1 {
				return remI < remJ
			}
			return remI > remJ
		})
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

// DeleteLoan deletes a loan and all its payments
func (s *LoanService) DeleteLoan(ctx context.Context, loanID primitive.ObjectID) error {
	// Check if loan exists
	var loan models.Loan
	err := s.db.Collection("loans").FindOne(ctx, bson.M{"_id": loanID}).Decode(&loan)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("loan not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	// Delete all payments for this loan
	_, err = s.db.Collection("loan_payments").DeleteMany(ctx, bson.M{"loan_id": loanID})
	if err != nil {
		return fmt.Errorf("failed to delete loan payments: %w", err)
	}

	// Delete the loan
	_, err = s.db.Collection("loans").DeleteOne(ctx, bson.M{"_id": loanID})
	if err != nil {
		return fmt.Errorf("failed to delete loan: %w", err)
	}

	return nil
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

// GetLoanPayments retrieves all payments for a specific loan
func (s *LoanService) GetLoanPayments(ctx context.Context, loanID primitive.ObjectID) ([]models.LoanPayment, error) {
	// Check if loan exists
	var loan models.Loan
	err := s.db.Collection("loans").FindOne(ctx, bson.M{"_id": loanID}).Decode(&loan)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("loan not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Get all payments for this loan, sorted by payment date (oldest first)
	findOpts := options.Find().SetSort(bson.D{{Key: "paid_at", Value: 1}})
	cursor, err := s.db.Collection("loan_payments").Find(ctx, bson.M{"loan_id": loanID}, findOpts)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var payments []models.LoanPayment
	if err := cursor.All(ctx, &payments); err != nil {
		return nil, fmt.Errorf("failed to decode payments: %w", err)
	}

	return payments, nil
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
