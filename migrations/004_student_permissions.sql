-- 1. Mendaftarkan permission baru untuk students
INSERT INTO permissions (name, description) VALUES
    ('student:list', 'Melihat daftar seluruh student'),
    ('student:read:any', 'Melihat data student mana pun'),
    ('student:create', 'Mendaftarkan student baru'),
    ('student:update:any', 'Mengubah data student mana pun'),
    ('student:delete', 'Menghapus student')
ON CONFLICT (name) DO NOTHING;

-- 2. Memasangkan permission kepada role admin dan staff
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

-- 3. Menambahkan owner_id ke tabel students
-- Tambahkan kolomnya dulu sebagai opsional (boleh NULL) agar tidak bentrok dengan data lama
ALTER TABLE students 
ADD COLUMN IF NOT EXISTS owner_id INTEGER REFERENCES users(id) ON DELETE CASCADE;

-- Update data lama yang belum punya owner_id (berikan ke user pertama di database)
UPDATE students 
SET owner_id = (SELECT id FROM users ORDER BY id ASC LIMIT 1) 
WHERE owner_id IS NULL;

-- Setelah semua baris lama diisi owner_id, kunci kolomnya menjadi wajib (NOT NULL)
ALTER TABLE students ALTER COLUMN owner_id SET NOT NULL;