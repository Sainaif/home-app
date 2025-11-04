package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/utils"
)

type ExportService struct {
	db *mongo.Database
}

func NewExportService(db *mongo.Database) *ExportService {
	return &ExportService{db: db}
}

// ExportBillsCSV exports bills and allocations to CSV
func (s *ExportService) ExportBillsCSV(ctx context.Context, billType *string, from *time.Time, to *time.Time) ([]byte, error) {
	// Build filter
	filter := bson.M{}
	if billType != nil {
		filter["type"] = *billType
	}
	if from != nil || to != nil {
		dateFilter := bson.M{}
		if from != nil {
			dateFilter["$gte"] = *from
		}
		if to != nil {
			dateFilter["$lte"] = *to
		}
		filter["period_start"] = dateFilter
	}

	// Get bills
	cursor, err := s.db.Collection("bills").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var bills []models.Bill
	if err := cursor.All(ctx, &bills); err != nil {
		return nil, fmt.Errorf("failed to decode bills: %w", err)
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
		amount, _ := utils.DecimalToFloat(bill.TotalAmountPLN)
		units, _ := utils.DecimalToFloat(bill.TotalUnits)

		notes := ""
		if bill.Notes != nil {
			notes = *bill.Notes
		}

		row := []string{
			bill.ID.Hex(),
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
	cursor, err := s.db.Collection("loans").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var loans []models.Loan
	if err := cursor.All(ctx, &loans); err != nil {
		return nil, fmt.Errorf("failed to decode loans: %w", err)
	}

	// Get user details
	userMap := make(map[primitive.ObjectID]string)
	for _, loan := range loans {
		for _, userID := range []primitive.ObjectID{loan.LenderID, loan.BorrowerID} {
			if _, exists := userMap[userID]; !exists {
				var user models.User
				err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
				if err == nil {
					userMap[userID] = user.Email
				}
			}
		}
	}

	// Get payments for each loan
	loanPayments := make(map[primitive.ObjectID]float64)
	paymentCursor, err := s.db.Collection("loan_payments").Find(ctx, bson.M{})
	if err == nil {
		defer paymentCursor.Close(ctx)
		var payments []models.LoanPayment
		if paymentCursor.All(ctx, &payments) == nil {
			for _, payment := range payments {
				amount, _ := utils.DecimalToFloat(payment.AmountPLN)
				loanPayments[payment.LoanID] += amount
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

		originalAmount, _ := utils.DecimalToFloat(loan.AmountPLN)
		paidAmount := loanPayments[loan.ID]
		remaining := originalAmount - paidAmount

		row := []string{
			loan.ID.Hex(),
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
func (s *ExportService) ExportChoresCSV(ctx context.Context, userID *primitive.ObjectID, status *string) ([]byte, error) {
	// Build filter
	filter := bson.M{}
	if userID != nil {
		filter["assignee_user_id"] = *userID
	}
	if status != nil {
		filter["status"] = *status
	}

	// Get assignments
	cursor, err := s.db.Collection("chore_assignments").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var assignments []models.ChoreAssignment
	if err := cursor.All(ctx, &assignments); err != nil {
		return nil, fmt.Errorf("failed to decode assignments: %w", err)
	}

	// Get chore and user details
	choreMap := make(map[primitive.ObjectID]string)
	userMap := make(map[primitive.ObjectID]string)

	for _, assign := range assignments {
		// Get chore name
		if _, exists := choreMap[assign.ChoreID]; !exists {
			var chore models.Chore
			err := s.db.Collection("chores").FindOne(ctx, bson.M{"_id": assign.ChoreID}).Decode(&chore)
			if err == nil {
				choreMap[assign.ChoreID] = chore.Name
			}
		}

		// Get user email
		if _, exists := userMap[assign.AssigneeUserID]; !exists {
			var user models.User
			err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": assign.AssigneeUserID}).Decode(&user)
			if err == nil {
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
			assign.ID.Hex(),
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
func (s *ExportService) ExportConsumptionsCSV(ctx context.Context, userID *primitive.ObjectID, from *time.Time, to *time.Time) ([]byte, error) {
	// Build filter
	filter := bson.M{}
	if userID != nil {
		filter["user_id"] = *userID
	}
	if from != nil || to != nil {
		dateFilter := bson.M{}
		if from != nil {
			dateFilter["$gte"] = *from
		}
		if to != nil {
			dateFilter["$lte"] = *to
		}
		filter["recorded_at"] = dateFilter
	}

	// Get consumptions
	cursor, err := s.db.Collection("consumptions").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var consumptions []models.Consumption
	if err := cursor.All(ctx, &consumptions); err != nil {
		return nil, fmt.Errorf("failed to decode consumptions: %w", err)
	}

	// Get bill and user details
	billMap := make(map[primitive.ObjectID]models.Bill)
	userMap := make(map[primitive.ObjectID]string)

	for _, cons := range consumptions {
		// Get bill
		if _, exists := billMap[cons.BillID]; !exists {
			var bill models.Bill
			err := s.db.Collection("bills").FindOne(ctx, bson.M{"_id": cons.BillID}).Decode(&bill)
			if err == nil {
				billMap[cons.BillID] = bill
			}
		}

		// Get subject (user or group) name
		if _, exists := userMap[cons.SubjectID]; !exists {
			if cons.SubjectType == "group" {
				var group models.Group
				err := s.db.Collection("groups").FindOne(ctx, bson.M{"_id": cons.SubjectID}).Decode(&group)
				if err == nil {
					userMap[cons.SubjectID] = group.Name
				}
			} else {
				var user models.User
				err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": cons.SubjectID}).Decode(&user)
				if err == nil {
					userMap[cons.SubjectID] = user.Email
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
		subjectName := userMap[cons.SubjectID]

		units, _ := utils.DecimalToFloat(cons.Units)

		meterValue := ""
		if cons.MeterValue != nil {
			mv, _ := utils.DecimalToFloat(*cons.MeterValue)
			meterValue = strconv.FormatFloat(mv, 'f', 3, 64)
		}

		period := fmt.Sprintf("%s to %s", bill.PeriodStart.Format("2006-01-02"), bill.PeriodEnd.Format("2006-01-02"))

		row := []string{
			cons.ID.Hex(),
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
