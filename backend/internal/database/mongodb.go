package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDB(uri, database string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(database)

	// Create indexes
	if err := createIndexes(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return &MongoDB{
		Client:   client,
		Database: db,
	}, nil
}

func (m *MongoDB) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}

func createIndexes(ctx context.Context, db *mongo.Database) error {
	// users.email unique
	_, err := db.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create users.email index: %w", err)
	}

	// bills.type, bills.period_start
	_, err = db.Collection("bills").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "type", Value: 1}, {Key: "period_start", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("failed to create bills index: %w", err)
	}

	// consumptions.bill_id
	_, err = db.Collection("consumptions").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "bill_id", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("failed to create consumptions.bill_id index: %w", err)
	}

	// consumptions.user_id+recorded_at
	_, err = db.Collection("consumptions").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "recorded_at", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("failed to create consumptions.user_id index: %w", err)
	}

	// allocations.bill_id+subject_type+subject_id
	_, err = db.Collection("allocations").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "bill_id", Value: 1}, {Key: "subject_type", Value: 1}, {Key: "subject_id", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("failed to create allocations index: %w", err)
	}

	// supply_items.status
	_, err = db.Collection("supply_items").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "status", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("failed to create supply_items.status index: %w", err)
	}

	// supply_items.bought_at (descending for recent purchases)
	_, err = db.Collection("supply_items").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "bought_at", Value: -1}},
	})
	if err != nil {
		return fmt.Errorf("failed to create supply_items.bought_at index: %w", err)
	}

	// supply_contributions.user_id+period_start
	_, err = db.Collection("supply_contributions").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "period_start", Value: -1}},
	})
	if err != nil {
		return fmt.Errorf("failed to create supply_contributions index: %w", err)
	}

	return nil
}
