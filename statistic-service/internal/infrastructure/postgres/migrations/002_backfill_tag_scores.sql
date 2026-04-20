alter table if exists attempts add column if not exists tag_scores jsonb not null default '[]'::jsonb;
