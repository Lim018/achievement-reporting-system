package repository

import (
	"context"
	"time"

	"go-fiber/app/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AchievementMongoRepo struct {
	Coll *mongo.Collection
}

func NewAchievementMongoRepo(db *mongo.Database) *AchievementMongoRepo {
	repo := &AchievementMongoRepo{
		Coll: db.Collection("achievement_records"),
	}
	
	repo.ensureIndexes()
	
	return repo
}

// indexes
func (r *AchievementMongoRepo) ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "studentId", Value: 1},
			},
			Options: options.Index().SetName("idx_studentId"),
		},
		{
			Keys: bson.D{
				{Key: "achievementType", Value: 1},
			},
			Options: options.Index().SetName("idx_achievementType"),
		},
		{
			Keys: bson.D{
				{Key: "createdAt", Value: -1},
			},
			Options: options.Index().SetName("idx_createdAt"),
		},
		{
			Keys: bson.D{
				{Key: "studentId", Value: 1},
				{Key: "achievementType", Value: 1},
			},
			Options: options.Index().SetName("idx_student_type"),
		},
		{
			Keys: bson.D{
				{Key: "tags", Value: 1},
			},
			Options: options.Index().SetName("idx_tags"),
		},
	}

	_, _ = r.Coll.Indexes().CreateMany(ctx, indexes)
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

func (r *AchievementMongoRepo) FindByHexIDWithProjection(ctx context.Context, hexId string, projection bson.M) (*model.Achievement, error) {
	oid, err := primitive.ObjectIDFromHex(hexId)
	if err != nil {
		return nil, err
	}
	
	opts := options.FindOne()
	if projection != nil {
		opts.SetProjection(projection)
	}
	
	var out model.Achievement
	err = r.Coll.FindOne(ctx, bson.M{"_id": oid}, opts).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *AchievementMongoRepo) BatchFindByHexIDs(ctx context.Context, hexIds []string, projection bson.M) (map[string]*model.Achievement, error) {
	if len(hexIds) == 0 {
		return make(map[string]*model.Achievement), nil
	}

	oids := make([]primitive.ObjectID, 0, len(hexIds))
	for _, hexId := range hexIds {
		oid, err := primitive.ObjectIDFromHex(hexId)
		if err != nil {
			continue
		}
		oids = append(oids, oid)
	}

	if len(oids) == 0 {
		return make(map[string]*model.Achievement), nil
	}

	filter := bson.M{"_id": bson.M{"$in": oids}}
	opts := options.Find()
	if projection != nil {
		opts.SetProjection(projection)
	}

	cursor, err := r.Coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make(map[string]*model.Achievement)
	for cursor.Next(ctx) {
		var ach model.Achievement
		if err := cursor.Decode(&ach); err != nil {
			continue
		}
		result[ach.ID.Hex()] = &ach
	}

	return result, nil
}

func (r *AchievementMongoRepo) FindByStudentID(ctx context.Context, studentID string, limit int64) ([]*model.Achievement, error) {
	filter := bson.M{"studentId": studentID}
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(limit)

	cursor, err := r.Coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []*model.Achievement
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

func (r *AchievementMongoRepo) CountByStudentID(ctx context.Context, studentID string) (int64, error) {
	filter := bson.M{"studentId": studentID}
	return r.Coll.CountDocuments(ctx, filter)
}

func (r *AchievementMongoRepo) FindByType(ctx context.Context, achievementType string, skip, limit int64) ([]*model.Achievement, error) {
	filter := bson.M{"achievementType": achievementType}
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(skip).
		SetLimit(limit)

	cursor, err := r.Coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []*model.Achievement
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
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

func (r *AchievementMongoRepo) GetStatsByStudent(ctx context.Context, studentID string) (map[string]interface{}, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "studentId", Value: studentID}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$achievementType"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "totalPoints", Value: bson.D{{Key: "$sum", Value: "$points"}}},
		}}},
	}

	cursor, err := r.Coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	stats["byType"] = results
	
	return stats, nil
}