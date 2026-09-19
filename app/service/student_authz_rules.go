package service

import (
	"latihan-sintaks/app/model"
	"latihan-sintaks/helper"
)

// CanAccessStudent memutuskan apakah seseorang boleh menyentuh data mahasiswa.
// Fungsi murni ini tidak boleh mengimpor fiber maupun repository.
func CanAccessStudent(
	current model.AuthUser,
	ownerID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	// 1. Kepemilikan (ownership): data itu didaftarkan oleh dirinya sendiri
	if current.UserID == ownerID {
		return true
	}
	
	// 2. Jika bukan miliknya, periksa apakah role-nya punya permission ':any'
	return perms.Can(current.Role, anyPermission)
}