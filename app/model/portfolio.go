package model

import "time"

// PortfolioProject merepresentasikan proyek dalam portofolio domain
type PortfolioProject struct {
	ID           string   `json:"id"`
	ProjectID    string   `json:"project_id"`
	Title        string   `json:"title"`
	Technologies []string `json:"technologies"`
	Link         string   `json:"link"`
	Year         int      `json:"year"`
}

// PortfolioCertificate merepresentasikan sertifikat dalam portofolio domain
type PortfolioCertificate struct {
	ID            string `json:"id"`
	CertificateID string `json:"certificate_id"`
	Name          string `json:"name"`
	Issuer        string `json:"issuer"`
	Date          string `json:"date"`
}

// StudentPortfolio adalah model domain utama untuk portofolio mahasiswa.
// Bebas dari dependensi BSON/driver MongoDB.
type StudentPortfolio struct {
	ID           string                 `json:"id"`
	StudentID    int                    `json:"student_id"`
	Headline     string                 `json:"headline"`
	Bio          string                 `json:"bio"`
	Projects     []PortfolioProject     `json:"projects"`
	Certificates []PortfolioCertificate `json:"certificates"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// UpsertPortfolioRequest digunakan untuk PUT /students/:id/portfolio
type UpsertPortfolioRequest struct {
	Headline     string                         `json:"headline" validate:"required,min=3,max=150"`
	Bio          string                         `json:"bio" validate:"required,min=10,max=2000"`
	Projects     []UpsertProjectItemRequest     `json:"projects" validate:"omitempty,max=20,dive"`
	Certificates []UpsertCertificateItemRequest `json:"certificates" validate:"omitempty,max=50,dive"`
}

type UpsertProjectItemRequest struct {
	Title        string   `json:"title" validate:"required,min=2,max=150"`
	Technologies []string `json:"technologies" validate:"required,min=1,max=20,dive,min=1,max=50"`
	Link         string   `json:"link" validate:"omitempty,url,max=255"`
	Year         int      `json:"year" validate:"required,min=1990,max=2100"`
}

type UpsertCertificateItemRequest struct {
	Name   string `json:"name" validate:"required,min=2,max=150"`
	Issuer string `json:"issuer" validate:"required,min=2,max=150"`
	Date   string `json:"date" validate:"required,datetime=2006-01-02"`
}

// AddProjectRequest digunakan untuk POST /students/:id/portfolio/projects
type AddProjectRequest struct {
	Title        string   `json:"title" validate:"required,min=2,max=150"`
	Technologies []string `json:"technologies" validate:"required,min=1,max=20,dive,min=1,max=50"`
	Link         string   `json:"link" validate:"omitempty,url,max=255"`
	Year         int      `json:"year" validate:"required,min=1990,max=2100"`
}

// TechnologyStat mendeskripsikan ringkasan penggunaan teknologi
type TechnologyStat struct {
	Technology string `json:"technology"`
	Total      int    `json:"total"`
}

// TechnologyStatsResponse adalah struktur response agregasi
type TechnologyStatsResponse struct {
	Technologies []TechnologyStat `json:"technologies"`
}
