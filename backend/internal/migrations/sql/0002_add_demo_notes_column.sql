-- Demo migration (harmless): proves the manual upgrade pipeline works
-- independently of GORM AutoMigrate. Adds an inert, unused column.
ALTER TABLE product_lists ADD COLUMN IF NOT EXISTS migration_demo_note text;
