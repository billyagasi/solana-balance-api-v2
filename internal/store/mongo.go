package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type APIKeyStore interface {
	IsValidAPIKey(ctx context.Context, key string) (bool, error)
	Close(ctx context.Context) error
}

type mongoStore struct {
	client *mongo.Client
	col    *mongo.Collection
}

type apiKeyDoc struct {
	Key    string `bson:"key"`
	Active bool   `bson:"active"`
}

func NewMongo(ctx context.Context, uri, db, col string) (APIKeyStore, error) {
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cli, err := mongo.Connect(ctx2, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := cli.Ping(ctx2, nil); err != nil {
		return nil, err
	}
	return &mongoStore{client: cli, col: cli.Database(db).Collection(col)}, nil
}

func (m *mongoStore) Close(ctx context.Context) error { return m.client.Disconnect(ctx) }

func (m *mongoStore) IsValidAPIKey(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, errors.New("empty api key")
	}
	var doc apiKeyDoc
	err := m.col.FindOne(ctx, bson.M{"key": key, "active": true}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return err == nil && doc.Active, err
}
