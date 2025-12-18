package repository

import (
	"context"
	"database/sql"
	"sync"

	"go-fiber/app/model"

	"github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ReportsRepo struct {
	PG    *sql.DB
	Mongo *mongo.Database
}

func NewReportsRepo(pg *sql.DB, mongoDB *mongo.Database) *ReportsRepo {
	return &ReportsRepo{
		PG:    pg,
		Mongo: mongoDB,
	}
}

func (r *ReportsRepo) GetTotalCounts(ctx context.Context) (model.TotalsData, error) {
	var totals model.TotalsData
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	wg.Add(3)

	go func() {
		defer wg.Done()
		err := r.PG.QueryRowContext(ctx, `SELECT COUNT(*) FROM students`).Scan(&totals.Students)
		if err != nil {
			errChan <- err
		}
	}()

	go func() {
		defer wg.Done()
		err := r.PG.QueryRowContext(ctx, `SELECT COUNT(*) FROM lecturers`).Scan(&totals.Lecturers)
		if err != nil {
			errChan <- err
		}
	}()

	go func() {
		defer wg.Done()
		err := r.PG.QueryRowContext(ctx, `SELECT COUNT(*) FROM achievement_references WHERE status != 'deleted'`).Scan(&totals.Achievements)
		if err != nil {
			errChan <- err
		}
	}()

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return totals, <-errChan
	}

	return totals, nil
}

func (r *ReportsRepo) GetAchievementStatusBreakdown(ctx context.Context) (map[string]int, error) {
	rows, err := r.PG.QueryContext(ctx, `
		SELECT status, COUNT(*) as count
		FROM achievement_references
		WHERE status != 'deleted'
		GROUP BY status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statusMap := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		statusMap[status] = count
	}

	return statusMap, nil
}

func (r *ReportsRepo) GetAchievementTypeBreakdown(ctx context.Context) (map[string]int, error) {
	coll := r.Mongo.Collection("achievement_records")

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$achievementType"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	typeMap := make(map[string]int)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		typeMap[result.ID] = result.Count
	}

	return typeMap, nil
}

func (r *ReportsRepo) GetTopStudents(ctx context.Context, limit int) ([]model.TopStudentData, error) {
	coll := r.Mongo.Collection("achievement_records")

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$studentId"},
			{Key: "totalPoints", Value: bson.D{{Key: "$sum", Value: "$points"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "totalPoints", Value: -1}}}},
		{{Key: "$limit", Value: limit}},
	}

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	type studentPoints struct {
		ID          string
		TotalPoints int
	}
	var studentsData []studentPoints

	for cursor.Next(ctx) {
		var result struct {
			ID          string `bson:"_id"`
			TotalPoints int    `bson:"totalPoints"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		studentsData = append(studentsData, studentPoints{
			ID:          result.ID,
			TotalPoints: result.TotalPoints,
		})
	}

	if len(studentsData) == 0 {
		return []model.TopStudentData{}, nil
	}

	studentIDs := make([]string, len(studentsData))
	for i, s := range studentsData {
		studentIDs[i] = s.ID
	}

	query := `
		SELECT s.id, u.full_name 
		FROM students s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = ANY($1)
	`

	rows, err := r.PG.QueryContext(ctx, query, pq.Array(studentIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nameMap := make(map[string]string)
	for rows.Next() {
		var id, fullName string
		if err := rows.Scan(&id, &fullName); err != nil {
			continue
		}
		nameMap[id] = fullName
	}

	var topStudents []model.TopStudentData
	for _, s := range studentsData {
		fullName := nameMap[s.ID]
		if fullName == "" {
			fullName = "Unknown"
		}

		topStudents = append(topStudents, model.TopStudentData{
			StudentID:   s.ID,
			FullName:    fullName,
			TotalPoints: s.TotalPoints,
		})
	}

	return topStudents, nil
}

func (r *ReportsRepo) GetMonthlyGrowth(ctx context.Context) ([]model.MonthlyGrowthData, error) {
	rows, err := r.PG.QueryContext(ctx, `
		SELECT 
			TO_CHAR(created_at, 'YYYY-MM') as month,
			COUNT(*) as total_achievements
		FROM achievement_references
		WHERE status != 'deleted'
		GROUP BY TO_CHAR(created_at, 'YYYY-MM')
		ORDER BY month DESC
		LIMIT 12
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var growth []model.MonthlyGrowthData
	for rows.Next() {
		var item model.MonthlyGrowthData
		if err := rows.Scan(&item.Month, &item.TotalAchievements); err != nil {
			return nil, err
		}
		growth = append(growth, item)
	}

	return growth, nil
}

func (r *ReportsRepo) GetStudentByIDOrStudentID(ctx context.Context, input string) (*struct {
	ID     string
	UserID string
}, error) {
	var student struct {
		ID     string
		UserID string
	}

	err := r.PG.QueryRowContext(ctx, `
		SELECT id, user_id
		FROM students
		WHERE id::text = $1 OR student_id = $1
	`, input).Scan(&student.ID, &student.UserID)

	if err != nil {
		return nil, err
	}

	return &student, nil
}

func (r *ReportsRepo) GetAdvisorUserID(ctx context.Context, studentID string) (string, error) {
	var advisorUserID string
	err := r.PG.QueryRowContext(ctx, `
		SELECT l.user_id
		FROM students s
		JOIN lecturers l ON s.advisor_id = l.id
		WHERE s.id = $1
	`, studentID).Scan(&advisorUserID)

	return advisorUserID, err
}

func (r *ReportsRepo) GetStudentInfo(ctx context.Context, studentID string) (*model.StudentInfo, error) {
	var info model.StudentInfo
	var advisorName sql.NullString
	var studyProgram sql.NullString

	err := r.PG.QueryRowContext(ctx, `
		SELECT 
			s.id,
			u.full_name,
			s.student_id,
			s.study_program,
			u_advisor.full_name as advisor_name
		FROM students s
		JOIN users u ON s.user_id = u.id
		LEFT JOIN lecturers l ON s.advisor_id = l.id
		LEFT JOIN users u_advisor ON l.user_id = u_advisor.id
		WHERE s.id = $1
	`, studentID).Scan(
		&info.ID,
		&info.FullName,
		&info.StudentID,
		&studyProgram,
		&advisorName,
	)

	if err != nil {
		return nil, err
	}

	if studyProgram.Valid && studyProgram.String != "" {
		info.StudyProgram = studyProgram.String
	} else {
		info.StudyProgram = "Belum ada program studi"
	}

	if advisorName.Valid && advisorName.String != "" {
		info.AdvisorName = advisorName.String
	} else {
		info.AdvisorName = "Belum ada dosen wali"
	}

	return &info, nil
}

func (r *ReportsRepo) GetStudentSummary(ctx context.Context, studentID string) (model.StudentSummary, error) {
	var summary model.StudentSummary

	err := r.PG.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM achievement_references 
		WHERE student_id = $1 AND status != 'deleted'
	`, studentID).Scan(&summary.TotalAchievements)

	if err != nil {
		return summary, err
	}

	coll := r.Mongo.Collection("achievement_records")
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "studentId", Value: studentID}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "totalPoints", Value: bson.D{{Key: "$sum", Value: "$points"}}},
		}}},
	}

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return summary, err
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result struct {
			TotalPoints int `bson:"totalPoints"`
		}
		if err := cursor.Decode(&result); err != nil {
			return summary, err
		}
		summary.TotalPoints = result.TotalPoints
	}

	return summary, nil
}

func (r *ReportsRepo) GetStudentAchievementStatusBreakdown(ctx context.Context, studentID string) (map[string]int, error) {
	rows, err := r.PG.QueryContext(ctx, `
		SELECT status, COUNT(*) as count
		FROM achievement_references
		WHERE student_id = $1 AND status != 'deleted'
		GROUP BY status
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statusMap := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		statusMap[status] = count
	}

	return statusMap, nil
}

func (r *ReportsRepo) GetStudentAchievementTypeBreakdown(ctx context.Context, studentID string) (map[string]int, error) {
	coll := r.Mongo.Collection("achievement_records")

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "studentId", Value: studentID}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$achievementType"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	typeMap := make(map[string]int)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		typeMap[result.ID] = result.Count
	}

	return typeMap, nil
}

func (r *ReportsRepo) GetStudentMonthlyGrowth(ctx context.Context, studentID string) ([]model.MonthlyGrowthData, error) {
	rows, err := r.PG.QueryContext(ctx, `
		SELECT 
			TO_CHAR(created_at, 'YYYY-MM') as month,
			COUNT(*) as total_achievements
		FROM achievement_references
		WHERE student_id = $1 AND status != 'deleted'
		GROUP BY TO_CHAR(created_at, 'YYYY-MM')
		ORDER BY month DESC
		LIMIT 12
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var growth []model.MonthlyGrowthData
	for rows.Next() {
		var item model.MonthlyGrowthData
		if err := rows.Scan(&item.Month, &item.TotalAchievements); err != nil {
			return nil, err
		}
		growth = append(growth, item)
	}

	return growth, nil
}

func (r *ReportsRepo) GetStudentAchievements(ctx context.Context, studentID string) ([]model.StudentAchievementInfo, error) {
	rows, err := r.PG.QueryContext(ctx, `
		SELECT 
			id,
			mongo_achievement_id,
			status,
			verified_at
		FROM achievement_references
		WHERE student_id = $1 AND status != 'deleted'
		ORDER BY created_at DESC
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var achievements []model.StudentAchievementInfo
	mongoIDs := []string{}

	for rows.Next() {
		var item model.StudentAchievementInfo
		var verifiedAt sql.NullTime

		if err := rows.Scan(&item.ReferenceID, &item.MongoID, &item.Status, &verifiedAt); err != nil {
			return nil, err
		}

		if verifiedAt.Valid {
			item.VerifiedAt = &verifiedAt.Time
		}

		achievements = append(achievements, item)
		mongoIDs = append(mongoIDs, item.MongoID)
	}

	if len(mongoIDs) == 0 {
		return achievements, nil
	}

	mongoData := r.batchFetchAchievementDetails(ctx, mongoIDs)

	for i := range achievements {
		if data, ok := mongoData[achievements[i].MongoID]; ok {
			achievements[i].Title = data.Title
			achievements[i].Type = data.Type
			achievements[i].Points = data.Points
		}
	}

	return achievements, nil
}

func (r *ReportsRepo) batchFetchAchievementDetails(ctx context.Context, mongoIDs []string) map[string]struct {
	Title  string
	Type   string
	Points int
} {
	result := make(map[string]struct {
		Title  string
		Type   string
		Points int
	})

	if len(mongoIDs) == 0 {
		return result
	}

	oids := make([]primitive.ObjectID, 0, len(mongoIDs))
	for _, hexID := range mongoIDs {
		if oid, err := primitive.ObjectIDFromHex(hexID); err == nil {
			oids = append(oids, oid)
		}
	}

	if len(oids) == 0 {
		return result
	}

	coll := r.Mongo.Collection("achievement_records")
	filter := bson.M{"_id": bson.M{"$in": oids}}
	projection := bson.M{
		"_id":             1,
		"title":           1,
		"achievementType": 1,
		"points":          1,
	}

	opts := options.Find().SetProjection(projection)
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return result
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var doc struct {
			ID              primitive.ObjectID `bson:"_id"`
			Title           string             `bson:"title"`
			AchievementType string             `bson:"achievementType"`
			Points          int                `bson:"points"`
		}

		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		result[doc.ID.Hex()] = struct {
			Title  string
			Type   string
			Points int
		}{
			Title:  doc.Title,
			Type:   doc.AchievementType,
			Points: doc.Points,
		}
	}

	return result
}