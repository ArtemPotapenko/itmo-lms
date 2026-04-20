create table if not exists courses (
    id text primary key,
    title text not null,
    owner_id text not null,
    status text not null,
    created_at timestamptz not null
);

create table if not exists course_members (
    course_id text not null,
    user_id text not null,
    role text not null,
    primary key(course_id, user_id)
);

create table if not exists assignments (
    id text primary key,
    course_id text not null,
    title text not null,
    work_id text not null default '',
    task_ids jsonb not null,
    due_at timestamptz,
    assigned_by text not null,
    status text not null,
    created_at timestamptz not null
);

create table if not exists submissions (
    id text primary key,
    assignment_id text not null,
    user_id text not null,
    answers_json jsonb not null,
    status text not null,
    submitted_at timestamptz not null,
    review_json jsonb
);
