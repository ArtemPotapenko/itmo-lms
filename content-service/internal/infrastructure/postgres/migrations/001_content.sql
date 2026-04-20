create table if not exists topics (
    id text primary key,
    parent_id text not null default '',
    title text not null,
    order_no integer not null,
    status text not null,
    created_at timestamptz not null
);

create table if not exists tags (
    id text primary key,
    code text not null unique,
    name text not null,
    description text not null default '',
    kind text not null,
    status text not null,
    created_at timestamptz not null
);

create table if not exists tasks (
    id text primary key,
    title text not null,
    latex_body text not null,
    topic_ids jsonb not null,
    tags jsonb not null,
    difficulty integer not null,
    correct_answer text not null,
    status text not null,
    author_id text not null default '',
    created_at timestamptz not null,
    updated_at timestamptz not null
);

create table if not exists task_tags (
    task_id text not null,
    tag_id text not null,
    weight double precision not null,
    primary key(task_id, tag_id)
);

create table if not exists theories (
    id text primary key,
    title text not null,
    body text not null,
    summary text not null,
    topic_ids jsonb not null,
    status text not null,
    created_at timestamptz not null,
    updated_at timestamptz not null
);

create table if not exists work_templates (
    id text primary key,
    title text not null,
    task_ids jsonb not null,
    items_json jsonb not null default '[]'::jsonb,
    status text not null,
    created_by text not null default '',
    created_at timestamptz not null
);
