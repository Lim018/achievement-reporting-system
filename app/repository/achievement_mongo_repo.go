package repository

import (
	"context"
	"time"

	"go-fiber/app/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AchievementMongoRepo struct {
	Coll *mongo.Collection
}

func NewAchievementMongoRepo(db *mongo.Database) *AchievementMongoRepo {
	return &AchievementMongoRepo{
		Coll: db.Collection("achievement_records"),
	}
}

func (r *AchievementMongoRepo) Create(ctx context.Context, ach model.Achievement) (string, error) {
	now := time.Now()
	ach.CreatedAt = now
	ach.UpdatedAt = now

	res, err := r.Coll.InsertOne(ctx, ach)
	if err != nil {
		return "", err
	}

	oid := res.InsertedID.(primitive.ObjectID)
	return oid.Hex(), nil
}

func (r *AchievementMongoRepo) UpdateByHexID(ctx context.Context, hexId string, update bson.M) error {
	oid, err := primitive.ObjectIDFromHex(hexId)
	if err != nil {
		return err
	}
	update["updatedAt"] = time.Now()
	_, err = r.Coll.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": update})
	return err
}

func (r *AchievementMongoRepo) DeleteByHexID(ctx context.Context, hexId string) error {
	oid, err := primitive.ObjectIDFromHex(hexId)
	if err != nil {
		return err
	}
	_, err = r.Coll.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}

func (r *AchievementMongoRepo) FindByHexID(ctx context.Context, hexId string) (*model.Achievement, error) {
	oid, err := primitive.ObjectIDFromHex(hexId)
	if err != nil {
		return nil, err
	}
	var out model.Achievement
	err = r.Coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *AchievementMongoRepo) AddAttachments(ctx context.Context, hexId string, atts []model.Attachment) error {
	oid, err := primitive.ObjectIDFromHex(hexId)
	if err != nil {
		return err
	}
	for i := range atts {
		atts[i].UploadedAt = time.Now()
	}
	_, err = r.Coll.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$push": bson.M{"attachments": bson.M{"$each": atts}}})
	return err
}