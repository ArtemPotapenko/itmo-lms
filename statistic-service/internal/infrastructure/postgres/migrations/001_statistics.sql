create table if not exists attempts (
    id text primary key,
    user_id text not null,
    content_id text not null,
    topic_ids jsonb not null,
    tag_scores jsonb not null default '[]'::jsonb,
    answer text not null,
    is_correct boolean not null,
    source text not null,
    created_at timestamptz not null
);
