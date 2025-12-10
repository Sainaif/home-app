package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
	"github.com/sainaif/holy-home/internal/utils"
)

type LoanService struct {
	loans               repository.LoanRepository
	loanPayments        repository.LoanPaymentRepository
	users               repository.UserRepository
	groups              repository.GroupRepository
	notificationService *NotificationService
}

func NewLoanService(
	loans repository.LoanRepository,
	loanPayments repository.LoanPaymentRepository,
	users repository.UserRepository,
	groups repository.GroupRepository,
	notificationService *NotificationService,
) *LoanService {
	return &LoanService{
		loans:               loans,
		loanPayments:        loanPayments,
		users:               users,
		groups:              groups,
		notificationService: notificationService,
	}
}

type CreateLoanRequest struct {
	LenderID   string     `json:"lenderId"`
	BorrowerID string     `json:"borrowerId"`
	AmountPLN  float64    `json:"amountPLN"`
	Note       *string    `json:"note,omitempty"`
	DueDate    *time.Time `json:"dueDate,omitempty"`
}

type CreateLoanPaymentRequest struct {
	LoanID    string    `json:"loanId"`
	AmountPLN float64   `json:"amountPLN"`
	PaidAt    time.Time `json:"paidAt"`
	Note      *string   `json:"note,omitempty"`
}

type CompensationResult struct {
	CompensationsPerformed int     `json:"compensationsPerformed"`
	TotalAmountCompensated float64 `json:"totalAmountCompensated"`
}

type Balance struct {
	UserID string  `json:"userId"`
	Owed   float64 `json:"owed"`  // Money this user owes to others
	Owing  float64 `json:"owing"` // Money others owe to this user
}

type PairwiseBalance struct {
	FromUserId        string  `json:"fromUserId"`
	ToUserId          string  `json:"toUserId"`
	FromUserName      string  `json:"fromUserName"`
	ToUserName        string  `json:"toUserName"`
	FromUserGroupID   *string `json:"fromUserGroupId,omitempty"`
	FromUserGroupName *string `json:"fromUserGroupName,omitempty"`
	ToUserGroupID     *string `json:"toUserGroupId,omitempty"`
	ToUserGroupName   *string `json:"toUserGroupName,omitempty"`
	NetAmount         string  `json:"netAmount"`
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
	for _, userID := range []string{req.LenderID, req.BorrowerID} {
		_, err := s.users.GetByID(ctx, userID)
		if err != nil {
			return nil, errors.New("user not found")
		}
	}

	// Perform group compensation on existing loans first
	_, err := s.PerformGroupCompensation(ctx)
	if err != nil {
		return nil, fmt.Errorf("group compensation failed: %w", err)
	}

	// Check for reverse debt (borrower owes lender)
	// Find open/partial loans where new borrower is the lender and new lender is the borrower
	reverseLoans, err := s.loans.ListOpenBetweenUsers(ctx, req.BorrowerID, req.LenderID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// If there are reverse debts, offset them
	remainingAmount := req.AmountPLN
	for _, reverseLoan := range reverseLoans {
		if remainingAmount <= 0 {
			break
		}

		// Calculate how much is remaining on the reverse loan
		reverseLoanAmount := utils.DecimalStringToFloat(reverseLoan.AmountPLN)
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
		payment := models.LoanPayment{
			ID:        uuid.New().String(),
			LoanID:    reverseLoan.ID,
			AmountPLN: utils.FloatToDecimalString(offsetAmount),
			PaidAt:    time.Now(),
			Note:      getStringPtr("Automatyczne rozliczenie długów"),
		}

		if err := s.loanPayments.Create(ctx, &payment); err != nil {
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

		reverseLoan.Status = newStatus
		if err := s.loans.Update(ctx, &reverseLoan); err != nil {
			return nil, fmt.Errorf("failed to update reverse loan status: %w", err)
		}

		remainingAmount -= offsetAmount
	}

	// If there's still remaining amount, create the new loan
	if remainingAmount > 0 {
		loan := models.Loan{
			ID:         uuid.New().String(),
			LenderID:   req.LenderID,
			BorrowerID: req.BorrowerID,
			AmountPLN:  utils.FloatToDecimalString(remainingAmount),
			Note:       req.Note,
			DueDate:    req.DueDate,
			Status:     "open",
			CreatedAt:  time.Now(),
		}

		if err := s.loans.Create(ctx, &loan); err != nil {
			return nil, fmt.Errorf("failed to create loan: %w", err)
		}

		// Notify borrower about new loan
		if s.notificationService != nil {
			lender, _ := s.users.GetByID(ctx, req.LenderID)
			lenderName := "Ktoś"
			if lender != nil {
				lenderName = lender.Name
			}
			borrowerID := req.BorrowerID
			_ = s.notificationService.CreateNotification(ctx, &models.Notification{
				UserID:     &borrowerID,
				TemplateID: "loan_created",
				Title:      "Nowa pożyczka",
				Body:       fmt.Sprintf("%s pożyczył/a Ci %.2f zł", lenderName, remainingAmount),
			})
		}

		return &loan, nil
	}

	// All debt was offset, save settled loan to database
	// Append offset message to user's note if they provided one
	var settledNote *string
	if req.Note != nil && *req.Note != "" {
		combined := *req.Note + " (Całkowicie rozliczone z istniejącymi długami)"
		settledNote = &combined
	} else {
		settledNote = getStringPtr("Całkowicie rozliczone z istniejącymi długami")
	}

	settledLoan := models.Loan{
		ID:         uuid.New().String(),
		LenderID:   req.LenderID,
		BorrowerID: req.BorrowerID,
		AmountPLN:  utils.FloatToDecimalString(req.AmountPLN),
		Note:       settledNote,
		DueDate:    req.DueDate,
		Status:     "settled",
		CreatedAt:  time.Now(),
	}

	if err := s.loans.Create(ctx, &settledLoan); err != nil {
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
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	// Build map: userID -> groupID
	userGroupMap := make(map[string]*string)
	for _, user := range users {
		userGroupMap[user.ID] = user.GroupID
	}

	// Get all open/partial loans, sorted by creation date (oldest first)
	openLoans, err := s.loans.ListByStatus(ctx, "open")
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	partialLoans, err := s.loans.ListByStatus(ctx, "partial")
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	loans := append(openLoans, partialLoans...)
	// Sort by creation date
	sort.Slice(loans, func(i, j int) bool {
		return loans[i].CreatedAt.Before(loans[j].CreatedAt)
	})

	// Calculate remaining amounts for each loan
	type loanWithRemaining struct {
		loan      models.Loan
		remaining float64
	}

	loansWithRemaining := []loanWithRemaining{}
	for _, loan := range loans {
		loanAmount := utils.DecimalStringToFloat(loan.AmountPLN)
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
			payment1 := models.LoanPayment{
				ID:        uuid.New().String(),
				LoanID:    loan1.ID,
				AmountPLN: utils.FloatToDecimalString(compensationAmount),
				PaidAt:    time.Now(),
				Note:      compensationNote,
			}

			if err := s.loanPayments.Create(ctx, &payment1); err != nil {
				return nil, fmt.Errorf("failed to create compensation payment 1: %w", err)
			}

			// Update loan1 status
			newTotalPaid1, _ := s.getTotalPaidForLoan(ctx, loan1.ID)
			loanAmount1 := utils.DecimalStringToFloat(loan1.AmountPLN)
			var newStatus1 string
			if newTotalPaid1 >= loanAmount1 {
				newStatus1 = "settled"
			} else {
				newStatus1 = "partial"
			}

			loan1.Status = newStatus1
			if err := s.loans.Update(ctx, &loan1); err != nil {
				return nil, fmt.Errorf("failed to update loan1 status: %w", err)
			}

			// Payment on loan2 (External -> GroupMemberB)
			payment2 := models.LoanPayment{
				ID:        uuid.New().String(),
				LoanID:    loan2.ID,
				AmountPLN: utils.FloatToDecimalString(compensationAmount),
				PaidAt:    time.Now(),
				Note:      compensationNote,
			}

			if err := s.loanPayments.Create(ctx, &payment2); err != nil {
				return nil, fmt.Errorf("failed to create compensation payment 2: %w", err)
			}

			// Update loan2 status
			newTotalPaid2, _ := s.getTotalPaidForLoan(ctx, loan2.ID)
			loanAmount2 := utils.DecimalStringToFloat(loan2.AmountPLN)
			var newStatus2 string
			if newTotalPaid2 >= loanAmount2 {
				newStatus2 = "settled"
			} else {
				newStatus2 = "partial"
			}

			loan2.Status = newStatus2
			if err := s.loans.Update(ctx, &loan2); err != nil {
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
	loan, err := s.loans.GetByID(ctx, req.LoanID)
	if err != nil {
		return nil, errors.New("loan not found")
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

	loanAmount := utils.DecimalStringToFloat(loan.AmountPLN)
	remaining := loanAmount - totalPaid

	if req.AmountPLN > remaining {
		return nil, fmt.Errorf("payment amount (%.2f) exceeds remaining balance (%.2f)", req.AmountPLN, remaining)
	}

	payment := models.LoanPayment{
		ID:        uuid.New().String(),
		LoanID:    req.LoanID,
		AmountPLN: utils.FloatToDecimalString(req.AmountPLN),
		PaidAt:    req.PaidAt,
		Note:      req.Note,
	}

	if err := s.loanPayments.Create(ctx, &payment); err != nil {
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

	loan.Status = newStatus
	if err := s.loans.Update(ctx, loan); err != nil {
		return nil, fmt.Errorf("failed to update loan status: %w", err)
	}

	// Notify lender about payment received
	if s.notificationService != nil {
		borrower, _ := s.users.GetByID(ctx, loan.BorrowerID)
		borrowerName := "Ktoś"
		if borrower != nil {
			borrowerName = borrower.Name
		}
		lenderID := loan.LenderID
		_ = s.notificationService.CreateNotification(ctx, &models.Notification{
			UserID:     &lenderID,
			TemplateID: "loan_payment_received",
			Title:      "Otrzymano spłatę pożyczki",
			Body:       fmt.Sprintf("%s spłacił/a %.2f zł", borrowerName, req.AmountPLN),
		})
	}

	return &payment, nil
}

// GetBalances calculates pairwise balances for all users
func (s *LoanService) GetBalances(ctx context.Context) ([]PairwiseBalance, error) {
	// Get all loans
	loans, err := s.loans.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Calculate net balances
	balances := make(map[string]float64) // key: "lenderID-borrowerID"

	for _, loan := range loans {
		if loan.Status == "settled" {
			continue
		}

		loanAmount := utils.DecimalStringToFloat(loan.AmountPLN)
		totalPaid, err := s.getTotalPaidForLoan(ctx, loan.ID)
		if err != nil {
			return nil, err
		}

		remaining := loanAmount - totalPaid
		if remaining <= 0 {
			continue
		}

		key := fmt.Sprintf("%s-%s", loan.BorrowerID, loan.LenderID)
		reverseKey := fmt.Sprintf("%s-%s", loan.LenderID, loan.BorrowerID)

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
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	userMap := make(map[string]string)
	userGroupMap := make(map[string]*string)
	for _, user := range users {
		userMap[user.ID] = user.Name
		userGroupMap[user.ID] = user.GroupID
	}

	// Get all groups for name lookup
	groups, err := s.groups.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch groups: %w", err)
	}

	groupMap := make(map[string]string)
	for _, group := range groups {
		groupMap[group.ID] = group.Name
	}

	// Convert to pairwise balances
	result := []PairwiseBalance{}
	for key, amount := range balances {
		// Parse the IDs properly
		parts := parseBalanceKey(key)
		if len(parts) == 2 {
			fromID := parts[0]
			toID := parts[1]

			balance := PairwiseBalance{
				FromUserId:   fromID,
				ToUserId:     toID,
				FromUserName: userMap[fromID],
				ToUserName:   userMap[toID],
				NetAmount:    utils.FloatToDecimalString(amount),
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
	FromUserName      string  `json:"fromUserName"`
	ToUserName        string  `json:"toUserName"`
	FromUserGroupID   *string `json:"fromUserGroupId,omitempty"`
	FromUserGroupName *string `json:"fromUserGroupName,omitempty"`
	ToUserGroupID     *string `json:"toUserGroupId,omitempty"`
	ToUserGroupName   *string `json:"toUserGroupName,omitempty"`
	RemainingPLN      string  `json:"remainingPLN"`
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
	loans, err := s.loans.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Get all users for name lookup
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	userMap := make(map[string]string)
	userGroupMap := make(map[string]*string)
	for _, user := range users {
		userMap[user.ID] = user.Name
		userGroupMap[user.ID] = user.GroupID
	}

	// Get all groups for name lookup
	groups, err := s.groups.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch groups: %w", err)
	}

	groupMap := make(map[string]string)
	for _, group := range groups {
		groupMap[group.ID] = group.Name
	}

	// Enrich with user names, group info, and remaining amounts
	result := make([]LoanWithNames, len(loans))
	for i, loan := range loans {
		// Calculate remaining amount
		loanAmount := utils.DecimalStringToFloat(loan.AmountPLN)
		totalPaid, err := s.getTotalPaidForLoan(ctx, loan.ID)
		if err != nil {
			totalPaid = 0
		}
		remaining := loanAmount - totalPaid

		loanWithNames := LoanWithNames{
			Loan:         loan,
			FromUserName: userMap[loan.LenderID],
			ToUserName:   userMap[loan.BorrowerID],
			RemainingPLN: utils.FloatToDecimalString(remaining),
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

	// Sort results
	sortOrder := -1 // desc by default
	if opts.Order == "asc" {
		sortOrder = 1
	}

	switch opts.SortBy {
	case "amountPLN":
		sort.Slice(result, func(i, j int) bool {
			amtI := utils.DecimalStringToFloat(result[i].AmountPLN)
			amtJ := utils.DecimalStringToFloat(result[j].AmountPLN)
			if sortOrder == 1 {
				return amtI < amtJ
			}
			return amtI > amtJ
		})
	case "status":
		sort.Slice(result, func(i, j int) bool {
			if sortOrder == 1 {
				return result[i].Status < result[j].Status
			}
			return result[i].Status > result[j].Status
		})
	case "remainingPLN":
		sort.Slice(result, func(i, j int) bool {
			remI := utils.DecimalStringToFloat(result[i].RemainingPLN)
			remJ := utils.DecimalStringToFloat(result[j].RemainingPLN)
			if sortOrder == 1 {
				return remI < remJ
			}
			return remI > remJ
		})
	default: // createdAt
		sort.Slice(result, func(i, j int) bool {
			if sortOrder == 1 {
				return result[i].CreatedAt.Before(result[j].CreatedAt)
			}
			return result[i].CreatedAt.After(result[j].CreatedAt)
		})
	}

	// Apply pagination
	if opts.Limit > 0 {
		start := opts.Offset
		if start > len(result) {
			start = len(result)
		}
		end := start + opts.Limit
		if end > len(result) {
			end = len(result)
		}
		result = result[start:end]
	}

	return result, nil
}

// GetUserBalance calculates balance for a specific user - returns pairwise balances
func (s *LoanService) GetUserBalance(ctx context.Context, userID string) ([]PairwiseBalance, error) {
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
func (s *LoanService) DeleteLoan(ctx context.Context, loanID string) error {
	// Check if loan exists
	_, err := s.loans.GetByID(ctx, loanID)
	if err != nil {
		return errors.New("loan not found")
	}

	// Delete all payments for this loan - we need to list and delete each
	payments, err := s.loanPayments.ListByLoanID(ctx, loanID)
	if err != nil {
		return fmt.Errorf("failed to list loan payments: %w", err)
	}

	for _, payment := range payments {
		if err := s.loanPayments.Delete(ctx, payment.ID); err != nil {
			return fmt.Errorf("failed to delete loan payment: %w", err)
		}
	}

	// Delete the loan
	if err := s.loans.Delete(ctx, loanID); err != nil {
		return fmt.Errorf("failed to delete loan: %w", err)
	}

	return nil
}

// Helper functions
func (s *LoanService) getTotalPaidForLoan(ctx context.Context, loanID string) (float64, error) {
	sumStr, err := s.loanPayments.SumByLoanID(ctx, loanID)
	if err != nil {
		return 0, fmt.Errorf("database error: %w", err)
	}
	return utils.DecimalStringToFloat(sumStr), nil
}

// GetLoanPayments retrieves all payments for a specific loan
func (s *LoanService) GetLoanPayments(ctx context.Context, loanID string) ([]models.LoanPayment, error) {
	// Check if loan exists
	_, err := s.loans.GetByID(ctx, loanID)
	if err != nil {
		return nil, errors.New("loan not found")
	}

	payments, err := s.loanPayments.ListByLoanID(ctx, loanID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
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
