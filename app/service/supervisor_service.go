package service

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

type SupervisorService struct {
	supervisors repository.SupervisorRepository
	tx          *repository.TxManager
}

func NewSupervisorService(supervisors repository.SupervisorRepository, tx *repository.TxManager) *SupervisorService {
	return &SupervisorService{supervisors: supervisors, tx: tx}
}

// GET /supervisors
func (s *SupervisorService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	list, err := s.supervisors.FindAll(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "daftar dosen pembimbing berhasil diambil", list)
}

// GET /supervisors/:id
func (s *SupervisorService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	sup, err := s.supervisors.FindByID(ctx, id)
	if err != nil {
		return translateSupervisorError(err)
	}
	return helper.Success(c, fiber.StatusOK, "dosen pembimbing ditemukan", sup)
}

// POST /supervisors/:id/assign
func (s *SupervisorService) Assign(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	supervisorID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id dosen pembimbing harus berupa angka positif")
	}

	var req model.AssignSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	var assigned model.Supervisor
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.supervisors.WithTx(tx)

		// 1. Kunci baris students untuk mencegah penetapan serentak mahasiswa yang sama
		if err := repo.LockStudent(ctx, req.StudentID); err != nil {
			return err
		}

		// 2. Tambah jumlah bimbingan secara atomik bersyarat
		sup, err := repo.IncrementSupervised(ctx, supervisorID)
		if err != nil {
			return err
		}
		assigned = sup

		// 3. Catat penetapan bimbingan (UNIQUE student_id akan membatalkan bila sudah punya pembimbing)
		return repo.InsertSupervision(ctx, req.StudentID, supervisorID)
	})

	if err != nil {
		return translateSupervisorError(err)
	}

	return helper.Created(c, "berhasil menetapkan dosen pembimbing", assigned,
		fmt.Sprintf("/api/v1/supervisors/%d", supervisorID))
}

// POST /supervisions/switch
func (s *SupervisorService) Switch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.SwitchSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	var switched model.Supervisor
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.supervisors.WithTx(tx)

		// 1. Kunci baris student
		if err := repo.LockStudent(ctx, req.StudentID); err != nil {
			return err
		}

		// 2. Dapatkan pembimbing saat ini
		currentSupID, err := repo.GetSupervision(ctx, req.StudentID)
		if err != nil {
			return err
		}

		if currentSupID == req.NewSupervisorID {
			return helper.Validation(map[string]string{
				"new_supervisor_id": "mahasiswa sudah dibimbing oleh dosen tersebut",
			})
		}

		// 3. Kunci kedua supervisor dengan urutan ID menaik untuk mencegah deadlock
		if err := repo.LockSupervisors(ctx, currentSupID, req.NewSupervisorID); err != nil {
			return err
		}

		// 4. Hapus bimbingan lama
		if err := repo.DeleteSupervision(ctx, req.StudentID); err != nil {
			return err
		}

		// 5. Kurangi jumlah bimbingan dosen lama (dalam tx!)
		if _, err := repo.DecrementSupervised(ctx, currentSupID); err != nil {
			return err
		}

		// 6. Tambah kuota dosen baru secara atomik bersyarat
		newSup, err := repo.IncrementSupervised(ctx, req.NewSupervisorID)
		if err != nil {
			return err
		}
		switched = newSup

		// 7. Catat bimbingan baru
		return repo.InsertSupervision(ctx, req.StudentID, req.NewSupervisorID)
	})

	if err != nil {
		return translateSupervisorError(err)
	}

	return helper.Success(c, fiber.StatusOK, "berhasil berganti dosen pembimbing", switched)
}

// PATCH /supervisors/:id
func (s *SupervisorService) UpdateQuota(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.UpdateSupervisorQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	updated, err := s.supervisors.UpdateQuota(ctx, id, req.Quota, req.Version)
	if err != nil {
		return translateSupervisorError(err)
	}

	return helper.Success(c, fiber.StatusOK, "kuota pembimbing berhasil diubah", updated)
}

func translateSupervisorError(err error) error {
	var appErr *helper.AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("data tidak ditemukan")
	case errors.Is(err, repository.ErrSupervisorFull):
		return helper.NewError(fiber.StatusConflict, "SUPERVISOR_FULL", "kuota pembimbing sudah penuh")
	case errors.Is(err, repository.ErrAlreadySupervised):
		return helper.NewError(fiber.StatusConflict, "ALREADY_SUPERVISED", "mahasiswa sudah memiliki pembimbing")
	case errors.Is(err, repository.ErrNoSupervision):
		return helper.NewError(fiber.StatusConflict, "NO_SUPERVISION", "mahasiswa belum memiliki pembimbing")
	case errors.Is(err, repository.ErrVersionConflict):
		return helper.NewError(fiber.StatusConflict, "VERSION_CONFLICT", "data pembimbing sudah diubah pihak lain, muat ulang lalu coba lagi")
	case errors.Is(err, repository.ErrQuotaBelowFilled):
		return helper.NewError(fiber.StatusConflict, "QUOTA_BELOW_SUPERVISED", "kuota tidak boleh lebih kecil dari jumlah mahasiswa yang dibimbing")
	default:
		return helper.Internal(err)
	}
}
