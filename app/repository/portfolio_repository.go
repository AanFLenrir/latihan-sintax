package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"latihan-fiber/app/model"
)

var (
	ErrPortfolioNotFound   = errors.New("portofolio tidak ditemukan")
	ErrProjectNotFound     = errors.New("proyek tidak ditemukan")
	ErrProjectLimitReached = errors.New("jumlah proyek sudah mencapai batas")
	ErrDuplicatePortfolio  = errors.New("portofolio mahasiswa sudah ada")
	ErrInvalidID           = errors.New("format id tidak valid")
)

const MaxProjectsPerPortfolio = 20

type PortfolioRepository interface {
	EnsureIndexes(ctx context.Context) error
	FindByStudentID(ctx context.Context, studentID int) (model.StudentPortfolio, error)
	Upsert(ctx context.Context, studentID int, req model.UpsertPortfolioRequest) (model.StudentPortfolio, error)
	AddProject(ctx context.Context, studentID int, req model.AddProjectRequest) (model.PortfolioProject, error)
	DeleteProject(ctx context.Context, studentID int, projectID string) error
	GetTechnologyStats(ctx context.Context) ([]model.TechnologyStat, error)
}

// BSON document models (internal to repository package)
type portfolioProjectDocument struct {
	ID           bson.ObjectID `bson:"project_id"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"`
	Link         string        `bson:"link"`
	Year         int           `bson:"year"`
}

func (p portfolioProjectDocument) toModel() model.PortfolioProject {
	techs := p.Technologies
	if techs == nil {
		techs = []string{}
	}
	hexID := p.ID.Hex()
	return model.PortfolioProject{
		ID:           hexID,
		ProjectID:    hexID,
		Title:        p.Title,
		Technologies: techs,
		Link:         p.Link,
		Year:         p.Year,
	}
}

type portfolioCertificateDocument struct {
	ID     bson.ObjectID `bson:"certificate_id"`
	Name   string        `bson:"name"`
	Issuer string        `bson:"issuer"`
	Date   string        `bson:"date"`
}

func (c portfolioCertificateDocument) toModel() model.PortfolioCertificate {
	hexID := c.ID.Hex()
	return model.PortfolioCertificate{
		ID:            hexID,
		CertificateID: hexID,
		Name:          c.Name,
		Issuer:        c.Issuer,
		Date:          c.Date,
	}
}

type portfolioDocument struct {
	ID           bson.ObjectID                  `bson:"_id,omitempty"`
	StudentID    int                            `bson:"student_id"`
	Headline     string                         `bson:"headline"`
	Bio          string                         `bson:"bio"`
	Projects     []portfolioProjectDocument     `bson:"projects"`
	Certificates []portfolioCertificateDocument `bson:"certificates"`
	CreatedAt    time.Time                      `bson:"created_at"`
	UpdatedAt    time.Time                      `bson:"updated_at"`
}

func (d portfolioDocument) toModel() model.StudentPortfolio {
	projects := make([]model.PortfolioProject, 0, len(d.Projects))
	for _, p := range d.Projects {
		projects = append(projects, p.toModel())
	}

	certs := make([]model.PortfolioCertificate, 0, len(d.Certificates))
	for _, c := range d.Certificates {
		certs = append(certs, c.toModel())
	}

	return model.StudentPortfolio{
		ID:           d.ID.Hex(),
		StudentID:    d.StudentID,
		Headline:     d.Headline,
		Bio:          d.Bio,
		Projects:     projects,
		Certificates: certs,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

type portfolioMongoRepository struct {
	coll *mongo.Collection
}

func NewPortfolioRepository(db *mongo.Database) PortfolioRepository {
	return &portfolioMongoRepository{coll: db.Collection("student_portfolios")}
}

func nowMillisecond() time.Time {
	return time.Now().UTC().Truncate(time.Millisecond)
}

func (r *portfolioMongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			// Unique index menjamin 1 mahasiswa hanya punya 1 dokumen portofolio
			Keys:    bson.D{{Key: "student_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("student_portfolio_unique"),
		},
		{
			// Multikey index pada array technologies dalam projects
			Keys:    bson.D{{Key: "projects.technologies", Value: 1}},
			Options: options.Index().SetName("projects_technologies_idx"),
		},
	})
	if err != nil {
		return fmt.Errorf("membuat index student_portfolios: %w", err)
	}
	return nil
}

func (r *portfolioMongoRepository) FindByStudentID(ctx context.Context, studentID int) (model.StudentPortfolio, error) {
	var doc portfolioDocument
	err := r.coll.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.StudentPortfolio{}, ErrPortfolioNotFound
		}
		return model.StudentPortfolio{}, fmt.Errorf("mengambil portofolio mahasiswa: %w", err)
	}
	return doc.toModel(), nil
}

func (r *portfolioMongoRepository) Upsert(ctx context.Context, studentID int, req model.UpsertPortfolioRequest) (model.StudentPortfolio, error) {
	t := nowMillisecond()

	projectsDocs := make([]portfolioProjectDocument, 0, len(req.Projects))
	for _, p := range req.Projects {
		cleanTechs := normalizeTechnologies(p.Technologies)
		projectsDocs = append(projectsDocs, portfolioProjectDocument{
			ID:           bson.NewObjectID(),
			Title:        strings.TrimSpace(p.Title),
			Technologies: cleanTechs,
			Link:         strings.TrimSpace(p.Link),
			Year:         p.Year,
		})
	}

	certDocs := make([]portfolioCertificateDocument, 0, len(req.Certificates))
	for _, c := range req.Certificates {
		certDocs = append(certDocs, portfolioCertificateDocument{
			ID:     bson.NewObjectID(),
			Name:   strings.TrimSpace(c.Name),
			Issuer: strings.TrimSpace(c.Issuer),
			Date:   strings.TrimSpace(c.Date),
		})
	}

	filter := bson.M{"student_id": studentID}
	update := bson.M{
		"$set": bson.M{
			"headline":     strings.TrimSpace(req.Headline),
			"bio":          strings.TrimSpace(req.Bio),
			"projects":     projectsDocs,
			"certificates": certDocs,
			"updated_at":   t,
		},
		"$setOnInsert": bson.M{
			"student_id": studentID,
			"created_at": t,
		},
	}

	opts := options.UpdateOne().SetUpsert(true)
	_, err := r.coll.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			// Race condition pada concurrent insert -> update tanpa upsert
			_, err = r.coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(false))
			if err != nil {
				return model.StudentPortfolio{}, fmt.Errorf("upsert portofolio retry: %w", err)
			}
		} else {
			return model.StudentPortfolio{}, fmt.Errorf("upsert portofolio: %w", err)
		}
	}

	return r.FindByStudentID(ctx, studentID)
}

func (r *portfolioMongoRepository) AddProject(ctx context.Context, studentID int, req model.AddProjectRequest) (model.PortfolioProject, error) {
	t := nowMillisecond()
	cleanTechs := normalizeTechnologies(req.Technologies)

	projectDoc := portfolioProjectDocument{
		ID:           bson.NewObjectID(),
		Title:        strings.TrimSpace(req.Title),
		Technologies: cleanTechs,
		Link:         strings.TrimSpace(req.Link),
		Year:         req.Year,
	}

	// Pola atomic conditional update:
	// Batas 20 proyek -> elemen ke-20 memiliki indeks 19 ("projects.19").
	// Filter "projects.19": {"$exists": false} memastikan array belum mencapai 20 item.
	limitField := fmt.Sprintf("projects.%d", MaxProjectsPerPortfolio-1)
	filter := bson.M{
		"student_id": studentID,
		limitField:   bson.M{"$exists": false},
	}
	update := bson.M{
		"$push": bson.M{"projects": projectDoc},
		"$set":  bson.M{"updated_at": t},
	}

	var updatedDoc portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updatedDoc)
	if err == nil {
		return projectDoc.toModel(), nil
	}

	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.PortfolioProject{}, fmt.Errorf("menambah proyek: %w", err)
	}

	// Tidak ada dokumen yang cocok. Periksa apakah portofolio tidak ada atau batas 20 proyek tercapai.
	count, err := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if err != nil {
		return model.PortfolioProject{}, fmt.Errorf("memeriksa portofolio: %w", err)
	}
	if count == 0 {
		return model.PortfolioProject{}, ErrPortfolioNotFound
	}
	return model.PortfolioProject{}, ErrProjectLimitReached
}

func (r *portfolioMongoRepository) DeleteProject(ctx context.Context, studentID int, projectID string) error {
	oid, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return ErrInvalidID
	}

	t := nowMillisecond()
	// Cari dokumen milik studentID yang memiliki project_id yang sesuai
	filter := bson.M{
		"student_id":          studentID,
		"projects.project_id": oid,
	}
	update := bson.M{
		"$pull": bson.M{"projects": bson.M{"project_id": oid}},
		"$set":  bson.M{"updated_at": t},
	}

	res, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("menghapus proyek: %w", err)
	}

	if res.ModifiedCount == 0 {
		// Evaluasi apakah portofolio tidak ada atau proyek tidak ditemukan dalam portofolio
		count, err := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
		if err != nil {
			return fmt.Errorf("memeriksa portofolio: %w", err)
		}
		if count == 0 {
			return ErrPortfolioNotFound
		}
		return ErrProjectNotFound
	}

	return nil
}

func (r *portfolioMongoRepository) GetTechnologyStats(ctx context.Context) ([]model.TechnologyStat, error) {
	// Aggregation pipeline dengan dua kali $unwind sesuai petunjuk Modul 10:
	// 1. $unwind pada projects
	// 2. $unwind pada projects.technologies
	// 3. $group berdasarkan projects.technologies dan hitung total ($sum: 1)
	// 4. $sort total DESC, tie-breaker nama ASC
	// 5. $limit 10
	// 6. $project format response
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$projects"}},
		{{Key: "$unwind", Value: "$projects.technologies"}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$projects.technologies",
			"total": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "total", Value: -1},
			{Key: "_id", Value: 1},
		}}},
		{{Key: "$limit", Value: 10}},
		{{Key: "$project", Value: bson.M{
			"_id":        0,
			"technology": "$_id",
			"total":      1,
		}}},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("agregasi statistik teknologi: %w", err)
	}
	defer cursor.Close(ctx)

	var results []model.TechnologyStat
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("membaca hasil agregasi statistik teknologi: %w", err)
	}

	if results == nil {
		results = []model.TechnologyStat{}
	}
	return results, nil
}

func normalizeTechnologies(techs []string) []string {
	result := make([]string, 0, len(techs))
	seen := make(map[string]struct{}, len(techs))
	for _, t := range techs {
		clean := strings.TrimSpace(t)
		if clean == "" {
			continue
		}
		lower := strings.ToLower(clean)
		if _, exists := seen[lower]; exists {
			continue
		}
		seen[lower] = struct{}{}
		result = append(result, clean)
	}
	return result
}
