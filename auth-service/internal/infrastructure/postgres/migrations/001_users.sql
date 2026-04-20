create table if not exists users (
    id text primary key,
    phone text not null unique,
    email text not null default '',
    first_name text not null,
    last_name text not null,
    nick text not null,
    password_hash text not null,
    roles_json jsonb not null,
    status text not null,
    created_at timestamptz not null
);
