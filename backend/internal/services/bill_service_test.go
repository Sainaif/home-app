package services

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sainaif/holy-home/internal/config"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupTestDB(t *testing.T) (*mongo.Database, func()) {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	require.NoError(t, err)

	db := client.Database("holy-home-test")

	cleanup := func() {
		err := db.Drop(context.Background())
		require.NoError(t, err)
		err = client.Disconnect(context.Background())
		require.NoError(t, err)
	}

	return db, cleanup
}

func TestDeleteBill_DeletesAllocations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create services
	cfg := &config.Config{}
	eventService := NewEventService()
	webPushService := NewWebPushService(db)
	notificationPreferenceService := NewNotificationPreferenceService(db)
	notificationService := NewNotificationService(db, eventService, webPushService, notificationPreferenceService, cfg)
	billService := NewBillService(db, notificationService)

	// Create a user
	user := &models.User{ID: primitive.NewObjectID(), Name: "Test User", Email: "test@example.com"}
	_, err := db.Collection("users").InsertOne(context.Background(), user)
	require.NoError(t, err)

	// Create a bill
	bill := &models.Bill{
		ID:          primitive.NewObjectID(),
		Type:        "electricity",
		PeriodStart: time.Now(),
		PeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
	}
	_, err = db.Collection("bills").InsertOne(context.Background(), bill)
	require.NoError(t, err)

	// Create an allocation
	allocation := bson.M{
		"_id":           primitive.NewObjectID(),
		"bill_id":       bill.ID,
		"subject_id":    user.ID,
		"subject_type":  "user",
		"allocated_pln": 100,
	}

	_, err = db.Collection("allocations").InsertOne(context.Background(), allocation)
	require.NoError(t, err)

	// Delete the bill
	err = billService.DeleteBill(context.Background(), bill.ID)
	require.NoError(t, err)

	// Check if the bill is deleted
	count, err := db.Collection("bills").CountDocuments(context.Background(), bson.M{"_id": bill.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "Bill should be deleted")

	// Check if the allocation is deleted
	count, err = db.Collection("allocations").CountDocuments(context.Background(), bson.M{"bill_id": bill.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "Allocation should be deleted")
}

func TestDeleteBill_Atomicity(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create services
	notificationService := NewNotificationService(db, nil, nil, nil, nil)

	// Create a user, bill, and allocation for the test
	userID := primitive.NewObjectID()
	_, err := db.Collection("users").InsertOne(context.Background(), bson.M{"_id": userID, "name": "Test User"})
	require.NoError(t, err)

	billID := primitive.NewObjectID()
	_, err = db.Collection("bills").InsertOne(context.Background(), bson.M{"_id": billID, "type": "electricity"})
	require.NoError(t, err)

	_, err = db.Collection("consumptions").InsertOne(context.Background(), bson.M{"bill_id": billID})
	require.NoError(t, err)

	// Create a service that will fail
	failingBillService := NewBillService(db, notificationService)
	failingBillService.db.Client().Disconnect(context.Background())

	// Attempt to delete the bill, expecting a failure
	err = failingBillService.DeleteBill(context.Background(), billID)
	require.Error(t, err)

	// Create a new client to verify the data was not deleted
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}
	newClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	require.NoError(t, err)
	defer newClient.Disconnect(context.Background())
	verificationDB := newClient.Database("holy-home-test")

	// Verify that the bill still exists
	var bill models.Bill
	err = verificationDB.Collection("bills").FindOne(context.Background(), bson.M{"_id": billID}).Decode(&bill)
	require.NoError(t, err, "Bill should not have been deleted")

	// Verify that the consumptions were NOT deleted
	count, err := verificationDB.Collection("consumptions").CountDocuments(context.Background(), bson.M{"_id": billID})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "Consumptions should not have been deleted")
}
