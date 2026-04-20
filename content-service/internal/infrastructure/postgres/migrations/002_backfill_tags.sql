alter table if exists tasks add column if not exists tags jsonb not null default '[]'::jsonb;
