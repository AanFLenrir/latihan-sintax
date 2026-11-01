package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

type PortfolioService struct {
	portfolios repository.PortfolioRepository
	students   repository.StudentRepository
	perms      *helper.PermissionSet
}

func NewPortfolioService(
	portfolios repository.PortfolioRepository,
	students repository.StudentRepository,
	perms *helper.PermissionSet,
) *PortfolioService {
	return &PortfolioService{
		portfolios: portfolios,
		students:   students,
		perms:      perms,
	}
}

// GET /students/:id/portfolio
func (s *PortfolioService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id mahasiswa harus berupa angka positif")
	}

	// Periksa keberadaan mahasiswa di PostgreSQL
	exists, err := s.students.Exists(ctx, studentID)
	if err != nil {
		return helper.Internal(err)
	}
	if !exists {
		return helper.NotFound("mahasiswa tidak ditemukan")
	}

	// Ambil portofolio dari MongoDB
	portfolio, err := s.portfolios.FindByStudentID(ctx, studentID)
	if err != nil {
		return translatePortfolioError(err)
	}

	return helper.Success(c, fiber.StatusOK, "portofolio mahasiswa ditemukan", portfolio)
}

// PUT /students/:id/portfolio
func (s *PortfolioService) Upsert(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id mahasiswa harus berupa angka positif")
	}

	// Decode body request ke struct bertipe tetap
	var req model.UpsertPortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	req.Headline = strings.TrimSpace(req.Headline)
	req.Bio = strings.TrimSpace(req.Bio)

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// Periksa keberadaan mahasiswa di PostgreSQL
	exists, err := s.students.Exists(ctx, studentID)
	if err != nil {
		return helper.Internal(err)
	}
	if !exists {
		return helper.NotFound("mahasiswa tidak ditemukan")
	}

	// Upsert ke MongoDB (mempertahankan satu dokumen per mahasiswa dengan index unik)
	portfolio, err := s.portfolios.Upsert(ctx, studentID, req)
	if err != nil {
		return translatePortfolioError(err)
	}

	return helper.Success(c, fiber.StatusOK, "portofolio mahasiswa berhasil disimpan", portfolio)
}

// POST /students/:id/portfolio/projects
func (s *PortfolioService) AddProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id mahasiswa harus berupa angka positif")
	}

	var req model.AddProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Link = strings.TrimSpace(req.Link)

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// Periksa mahasiswa di PostgreSQL
	exists, err := s.students.Exists(ctx, studentID)
	if err != nil {
		return helper.Internal(err)
	}
	if !exists {
		return helper.NotFound("mahasiswa tidak ditemukan")
	}

	// Tambahkan proyek ke MongoDB secara atomik dengan batas 20 proyek
	project, err := s.portfolios.AddProject(ctx, studentID, req)
	if err != nil {
		return translatePortfolioError(err)
	}

	loc := fmt.Sprintf("/api/v1/students/%d/portfolio/projects/%s", studentID, project.ID)
	return helper.Created(c, "proyek berhasil ditambahkan ke portofolio", project, loc)
}

// DELETE /students/:id/portfolio/projects/:projectId
func (s *PortfolioService) DeleteProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id mahasiswa harus berupa angka positif")
	}

	projectID := strings.TrimSpace(c.Params("projectId"))
	if projectID == "" {
		return helper.BadRequest("id proyek tidak boleh kosong")
	}

	// Periksa mahasiswa di PostgreSQL
	exists, err := s.students.Exists(ctx, studentID)
	if err != nil {
		return helper.Internal(err)
	}
	if !exists {
		return helper.NotFound("mahasiswa tidak ditemukan")
	}

	// Hapus proyek dari MongoDB dengan $pull atomik
	if err := s.portfolios.DeleteProject(ctx, studentID, projectID); err != nil {
		return translatePortfolioError(err)
	}

	return helper.NoContent(c)
}

// GET /portfolios/stats/technologies
func (s *PortfolioService) GetTechnologyStats(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	stats, err := s.portfolios.GetTechnologyStats(ctx)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "statistik teknologi berhasil diambil", model.TechnologyStatsResponse{
		Technologies: stats,
	})
}

func translatePortfolioError(err error) error {
	switch {
	case errors.Is(err, repository.ErrInvalidID):
		return helper.BadRequest("id proyek tidak valid")
	case errors.Is(err, repository.ErrPortfolioNotFound):
		return helper.NotFound("portofolio mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrProjectNotFound):
		return helper.NotFound("proyek tidak ditemukan dalam portofolio")
	case errors.Is(err, repository.ErrProjectLimitReached):
		return helper.NewError(fiber.StatusConflict, "PROJECT_LIMIT", "jumlah proyek sudah mencapai batas maksimal 20")
	case errors.Is(err, repository.ErrDuplicatePortfolio):
		return helper.NewError(fiber.StatusConflict, "DUPLICATE_PORTFOLIO", "portofolio mahasiswa sudah ada")
	default:
		return helper.Internal(err)
	}
}
