package repository

import (
	"context"
	"database/sql"
	"go-fiber/app/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

// Get total counts for system statistics
func (r *ReportsRepo) GetTotalCounts() (model.TotalsData, error) {
	var totals model.TotalsData

	err := r.PG.QueryRow(`SELECT COUNT(*) FROM students`).Scan(&totals.Students)
	if err != nil {
		return totals, err
	}

	err = r.PG.QueryRow(`SELECT COUNT(*) FROM lecturers`).Scan(&totals.Lecturers)
	if err != nil {
		return totals, err
	}

	err = r.PG.QueryRow(`SELECT COUNT(*) FROM achievement_references WHERE status != 'deleted'`).Scan(&totals.Achievements)
	if err != nil {
		return totals, err
	}

	return totals, nil
}

// Get achievement status breakdown
func (r *ReportsRepo) GetAchievementStatusBreakdown() (map[string]int, error) {
	rows, err := r.PG.Query(`
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

// Get achievement type breakdown from MongoDB
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

// Get top students by total points
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

	var topStudents []model.TopStudentData
	for cursor.Next(ctx) {
		var result struct {
			ID          string `bson:"_id"`
			TotalPoints int    `bson:"totalPoints"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}

		// Get student full name from PostgreSQL
		// result.ID adalah students.id (UUID dari tabel students)
		var fullName string
		err := r.PG.QueryRow(`
			SELECT u.full_name 
			FROM users u
			JOIN students s ON u.id = s.user_id
			WHERE s.id = $1
		`, result.ID).Scan(&fullName)

		if err != nil {
			// Jika gagal, coba fallback dengan query langsung ke students
			err2 := r.PG.QueryRow(`
				SELECT full_name FROM users WHERE id = $1
			`, result.ID).Scan(&fullName)
			
			if err2 != nil {
				fullName = "Unknown"
			}
		}

		topStudents = append(topStudents, model.TopStudentData{
			StudentID:   result.ID,
			FullName:    fullName,
			TotalPoints: result.TotalPoints,
		})
	}

	return topStudents, nil
}

// Get monthly growth statistics
func (r *ReportsRepo) GetMonthlyGrowth() ([]model.MonthlyGrowthData, error) {
	rows, err := r.PG.Query(`
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

// Get student info
func (r *ReportsRepo) GetStudentInfo(studentID string) (*model.StudentInfo, error) {
	var info model.StudentInfo
	var advisorName sql.NullString
	var programStudy sql.NullString

	err := r.PG.QueryRow(`
		SELECT 
			s.id,
			u.full_name,
			s.student_id,
			COALESCE(s.program_study, 'Belum ada program studi') as program_study,
			COALESCE(u_advisor.full_name, 'Belum ada dosen wali') as advisor_name
		FROM students s
		JOIN users u ON s.user_id = u.id
		LEFT JOIN lecturers l ON s.advisor_id = l.id
		LEFT JOIN users u_advisor ON l.user_id = u_advisor.id
		WHERE s.id = $1
	`, studentID).Scan(
		&info.ID,
		&info.FullName,
		&info.StudentID,
		&programStudy,
		&advisorName,
	)

	if err != nil {
		return nil, err
	}

	if programStudy.Valid {
		info.StudyProgram = programStudy.String
	} else {
		info.StudyProgram = "Belum ada program studi"
	}

	if advisorName.Valid {
		info.AdvisorName = advisorName.String
	} else {
		info.AdvisorName = "Belum ada dosen wali"
	}

	return &info, nil
}

// Get student summary
func (r *ReportsRepo) GetStudentSummary(ctx context.Context, studentID string) (model.StudentSummary, error) {
	var summary model.StudentSummary

	err := r.PG.QueryRow(`
		SELECT COUNT(*) 
		FROM achievement_references 
		WHERE student_id = $1 AND status != 'deleted'
	`, studentID).Scan(&summary.TotalAchievements)

	if err != nil {
		return summary, err
	}

	// Get total points from MongoDB
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

// Get student achievement status breakdown
func (r *ReportsRepo) GetStudentAchievementStatusBreakdown(studentID string) (map[string]int, error) {
	rows, err := r.PG.Query(`
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

// Get student achievement type breakdown
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

// Get student monthly growth
func (r *ReportsRepo) GetStudentMonthlyGrowth(studentID string) ([]model.MonthlyGrowthData, error) {
	rows, err := r.PG.Query(`
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

// Get student achievements list
func (r *ReportsRepo) GetStudentAchievements(ctx context.Context, studentID string) ([]model.StudentAchievementInfo, error) {
	rows, err := r.PG.Query(`
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
	coll := r.Mongo.Collection("achievement_records")

	for rows.Next() {
		var item model.StudentAchievementInfo
		var verifiedAt sql.NullTime

		if err := rows.Scan(&item.ReferenceID, &item.MongoID, &item.Status, &verifiedAt); err != nil {
			return nil, err
		}

		if verifiedAt.Valid {
			item.VerifiedAt = &verifiedAt.Time
		}

		// Get title, type, and points from MongoDB using hex string
		objID, err := primitive.ObjectIDFromHex(item.MongoID)
		if err == nil {
			var achDoc struct {
				Title           string `bson:"title"`
				AchievementType string `bson:"achievementType"`
				Points          int    `bson:"points"`
			}

			err = coll.FindOne(ctx, bson.M{"_id": objID}).Decode(&achDoc)
			if err == nil {
				item.Title = achDoc.Title
				item.Type = achDoc.AchievementType
				item.Points = achDoc.Points
			}
		}

		achievements = append(achievements, item)
	}

	return achievements, nil
}