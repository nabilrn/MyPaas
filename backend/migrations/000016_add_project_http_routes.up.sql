ALTER TABLE projects
ADD COLUMN additional_routes JSONB NOT NULL DEFAULT '[]'::jsonb;
