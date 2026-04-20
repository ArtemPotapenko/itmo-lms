alter table if exists work_templates add column if not exists items_json jsonb not null default '[]'::jsonb;
