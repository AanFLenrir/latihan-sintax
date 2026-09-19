-- 1. Tambahkan permission baru
INSERT INTO permissions (name, description) VALUES
('student:list', 'Melihat daftar seluruh mahasiswa'),
('student:read:any', 'Melihat data mahasiswa mana pun'),
('student:create', 'Menambahkan data mahasiswa'),
('student:update:any', 'Mengubah data mahasiswa mana pun'),
('student:delete', 'Menghapus data mahasiswa')
ON CONFLICT (name) DO NOTHING;

-- 2. Pasangkan ke role admin dan staff
INSERT INTO role_permissions (role_name, permission_name) VALUES
('admin', 'student:list'),
('admin', 'student:read:any'),
('admin', 'student:create'),
('admin', 'student:update:any'),
('admin', 'student:delete'),
('staff', 'student:list'),
('staff', 'student:read:any'),
('staff', 'student:create')
ON CONFLICT DO NOTHING;

-- 3. Tambah kolom owner_id di tabel students
ALTER TABLE students ADD COLUMN IF NOT EXISTS owner_id INTEGER;

-- 4. Amankan data lama: isi owner_id yang masih kosong dengan ID Admin (misal id = 1)
UPDATE students SET owner_id = 1 WHERE owner_id IS NULL;

-- 5. Baru pasang constraint Foreign Key
ALTER TABLE students DROP CONSTRAINT IF EXISTS students_owner_id_fkey;
ALTER TABLE students
ADD CONSTRAINT students_owner_id_fkey
FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE SET NULL;