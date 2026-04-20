create table if not exists document_jobs (
    id text primary key,
    format text not null,
    status text not null,
    files_json jsonb not null,
    error text not null default '',
    created_at timestamptz not null,
    completed_at timestamptz not null
);
