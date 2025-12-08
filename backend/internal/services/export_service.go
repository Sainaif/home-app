package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/sainaif/holy-home/internal/repository"
	"github.com/sainaif/holy-home/internal/utils"
)

type ExportService struct {
	bills            repository.BillRepository
	consumptions     repository.ConsumptionRepository
	loans            repository.LoanRepository
	loanPayments     repository.LoanPaymentRepository
	chores           repository.ChoreRepository
	choreAssignments repository.ChoreAssignmentRepository
	users            repository.UserRepository
	groups           repository.GroupRepository
}

func NewExportService(
	bills repository.BillRepository,
	consumptions repository.ConsumptionRepository,
	loans repository.LoanRepository,
	loanPayments repository.LoanPaymentRepository,
	chores repository.ChoreRepository,
	choreAssignments repository.ChoreAssignmentRepository,
	users repository.UserRepository,
	groups repository.GroupRepository,
) *ExportService {
	return &ExportService{
		bills:            bills,
		consumptions:     consumptions,
		loans:            loans,
		loanPayments:     loanPayments,
		chores:           chores,
		choreAssignments: choreAssignments,
		users:            users,
		groups:           groups,
	}
}

// ExportBillsCSV exports bills and allocations to CSV
func (s *ExportService) ExportBillsCSV(ctx context.Context, billType *string, from *time.Time, to *time.Time) ([]byte, error) {
	// Get bills with optional filtering
	bills, err := s.bills.ListFiltered(ctx, billType, from, to)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Create CSV buffer
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{"Bill ID", "Type", "Period Start", "Period End", "Total Amount (PLN)", "Total Units", "Status", "Notes"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write bill rows
	for _, bill := range bills {
		amount := utils.DecimalStringToFloat(bill.TotalAmountPLN)

		var units float64
		if bill.TotalUnits != "" {
			units = utils.DecimalStringToFloat(bill.TotalUnits)
		}

		notes := ""
		if bill.Notes != nil {
			notes = *bill.Notes
		}

		row := []string{
			bill.ID,
			bill.Type,
			bill.PeriodStart.Format("2006-01-02"),
			bill.PeriodEnd.Format("2006-01-02"),
			fmt.Sprintf("%.2f", amount),
			fmt.Sprintf("%.3f", units),
			bill.Status,
			notes,
		}

		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportBalancesCSV exports loan balances
func (s *ExportService) ExportBalancesCSV(ctx context.Context) ([]byte, error) {
	// Get all loans
	loans, err := s.loans.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Get user details
	userMap := make(map[string]string) // userID -> email
	for _, loan := range loans {
		for _, userID := range []string{loan.LenderID, loan.BorrowerID} {
			if _, exists := userMap[userID]; !exists {
				user, err := s.users.GetByID(ctx, userID)
				if err == nil && user != nil {
					userMap[userID] = user.Email
				}
			}
		}
	}

	// Get payments for each loan
	loanPaymentsMap := make(map[string]float64)
	for _, loan := range loans {
		payments, err := s.loanPayments.ListByLoanID(ctx, loan.ID)
		if err == nil {
			for _, payment := range payments {
				amount := utils.DecimalStringToFloat(payment.AmountPLN)
				loanPaymentsMap[loan.ID] += amount
			}
		}
	}

	// Create CSV buffer
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{"Loan ID", "Lender", "Borrower", "Original Amount (PLN)", "Paid Amount (PLN)", "Remaining (PLN)", "Status", "Created Date"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write loan rows
	for _, loan := range loans {
		lenderEmail := userMap[loan.LenderID]
		borrowerEmail := userMap[loan.BorrowerID]

		originalAmount := utils.DecimalStringToFloat(loan.AmountPLN)
		paidAmount := loanPaymentsMap[loan.ID]
		remaining := originalAmount - paidAmount

		row := []string{
			loan.ID,
			lenderEmail,
			borrowerEmail,
			fmt.Sprintf("%.2f", originalAmount),
			fmt.Sprintf("%.2f", paidAmount),
			fmt.Sprintf("%.2f", remaining),
			loan.Status,
			loan.CreatedAt.Format("2006-01-02"),
		}

		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportChoresCSV exports chore assignments
func (s *ExportService) ExportChoresCSV(ctx context.Context, userID *string, status *string) ([]byte, error) {
	// Get assignments with optional filtering
	assignments, err := s.choreAssignments.ListFiltered(ctx, userID, status)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Get chore and user details
	choreMap := make(map[string]string)
	userMap := make(map[string]string)

	for _, assign := range assignments {
		// Get chore name
		if _, exists := choreMap[assign.ChoreID]; !exists {
			chore, err := s.chores.GetByID(ctx, assign.ChoreID)
			if err == nil && chore != nil {
				choreMap[assign.ChoreID] = chore.Name
			}
		}

		// Get user email
		if _, exists := userMap[assign.AssigneeUserID]; !exists {
			user, err := s.users.GetByID(ctx, assign.AssigneeUserID)
			if err == nil && user != nil {
				userMap[assign.AssigneeUserID] = user.Email
			}
		}
	}

	// Create CSV buffer
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{"Assignment ID", "Chore", "Assignee", "Due Date", "Status", "Completed Date"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write assignment rows
	for _, assign := range assignments {
		choreName := choreMap[assign.ChoreID]
		userEmail := userMap[assign.AssigneeUserID]

		completedDate := ""
		if assign.CompletedAt != nil {
			completedDate = assign.CompletedAt.Format("2006-01-02 15:04")
		}

		row := []string{
			assign.ID,
			choreName,
			userEmail,
			assign.DueDate.Format("2006-01-02"),
			assign.Status,
			completedDate,
		}

		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportConsumptionsCSV exports consumption history
func (s *ExportService) ExportConsumptionsCSV(ctx context.Context, subjectID *string, from *time.Time, to *time.Time) ([]byte, error) {
	// Get consumptions with optional filtering
	// Note: subjectID filter applies to user type by default
	var subjectType *string
	if subjectID != nil {
		st := "user"
		subjectType = &st
	}
	consumptions, err := s.consumptions.ListFiltered(ctx, subjectType, subjectID, from, to)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Get bill and subject details
	billMap := make(map[string]billInfo)
	subjectMap := make(map[string]string)

	for _, cons := range consumptions {
		// Get bill
		if _, exists := billMap[cons.BillID]; !exists {
			bill, err := s.bills.GetByID(ctx, cons.BillID)
			if err == nil && bill != nil {
				billMap[cons.BillID] = billInfo{
					Type:        bill.Type,
					PeriodStart: bill.PeriodStart,
					PeriodEnd:   bill.PeriodEnd,
				}
			}
		}

		// Get subject (user or group) name
		if _, exists := subjectMap[cons.SubjectID]; !exists {
			if cons.SubjectType == "group" {
				group, err := s.groups.GetByID(ctx, cons.SubjectID)
				if err == nil && group != nil {
					subjectMap[cons.SubjectID] = group.Name
				}
			} else {
				user, err := s.users.GetByID(ctx, cons.SubjectID)
				if err == nil && user != nil {
					subjectMap[cons.SubjectID] = user.Email
				}
			}
		}
	}

	// Create CSV buffer
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{"Consumption ID", "Bill Type", "Period", "User", "Units", "Meter Value", "Recorded Date", "Source"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write consumption rows
	for _, cons := range consumptions {
		bill := billMap[cons.BillID]
		subjectName := subjectMap[cons.SubjectID]

		units := utils.DecimalStringToFloat(cons.Units)

		meterValue := ""
		if cons.MeterValue != nil {
			mv := utils.DecimalStringToFloat(*cons.MeterValue)
			meterValue = strconv.FormatFloat(mv, 'f', 3, 64)
		}

		period := fmt.Sprintf("%s to %s", bill.PeriodStart.Format("2006-01-02"), bill.PeriodEnd.Format("2006-01-02"))

		row := []string{
			cons.ID,
			bill.Type,
			period,
			subjectName,
			fmt.Sprintf("%.3f", units),
			meterValue,
			cons.RecordedAt.Format("2006-01-02 15:04"),
			cons.Source,
		}

		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

type billInfo struct {
	Type        string
	PeriodStart time.Time
	PeriodEnd   time.Time
}
