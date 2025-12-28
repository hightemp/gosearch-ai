-- +goose Up

create extension if not exists pgcrypto;

create table if not exists users (
  id uuid primary key default gen_random_uuid(),
  email text not null unique,
  password_hash text not null,
  preferred_model text not null default 'openai/gpt-4.1-mini',
  created_at timestamptz not null default now()
);

create table if not exists refresh_tokens (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  token_hash text not null,
  expires_at timestamptz not null,
  revoked_at timestamptz null,
  replaced_by uuid null,
  created_at timestamptz not null default now()
);
create index if not exists refresh_tokens_user_id_idx on refresh_tokens(user_id);

create table if not exists chats (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  title text not null default '',
  pinned boolean not null default false,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  deleted_at timestamptz null
);
create index if not exists chats_user_id_idx on chats(user_id);
create index if not exists chats_user_id_updated_at_idx on chats(user_id, updated_at desc);

create table if not exists messages (
  id uuid primary key default gen_random_uuid(),
  chat_id uuid not null references chats(id) on delete cascade,
  user_id uuid not null references users(id) on delete cascade,
  role text not null check (role in ('user', 'assistant', 'system')),
  content text not null,
  created_at timestamptz not null default now()
);
create index if not exists messages_chat_id_created_at_idx on messages(chat_id, created_at);

create table if not exists runs (
  id uuid primary key default gen_random_uuid(),
  chat_id uuid not null references chats(id) on delete cascade,
  user_id uuid not null references users(id) on delete cascade,
  model text not null,
  status text not null,
  started_at timestamptz not null default now(),
  finished_at timestamptz null,
  error text null
);
create index if not exists runs_user_id_started_at_idx on runs(user_id, started_at desc);

create table if not exists run_steps (
  id uuid primary key default gen_random_uuid(),
  run_id uuid not null references runs(id) on delete cascade,
  type text not null,
  title text not null,
  payload jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);
create index if not exists run_steps_run_id_created_at_idx on run_steps(run_id, created_at);

create table if not exists search_queries (
  id uuid primary key default gen_random_uuid(),
  run_id uuid not null references runs(id) on delete cascade,
  query text not null,
  category text not null default 'general',
  created_at timestamptz not null default now()
);
create index if not exists search_queries_run_id_created_at_idx on search_queries(run_id, created_at);

create table if not exists search_results (
  id uuid primary key default gen_random_uuid(),
  query_id uuid not null references search_queries(id) on delete cascade,
  rank int not null,
  title text not null,
  url text not null,
  snippet text not null default '',
  engine text not null default 'searxng',
  raw jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);
create index if not exists search_results_query_id_rank_idx on search_results(query_id, rank);

create table if not exists sources (
  id uuid primary key default gen_random_uuid(),
  run_id uuid not null references runs(id) on delete cascade,
  url text not null,
  title text not null default '',
  domain text not null default '',
  favicon_url text not null default '',
  created_at timestamptz not null default now()
);
create index if not exists sources_run_id_idx on sources(run_id);

create table if not exists page_snippets (
  id uuid primary key default gen_random_uuid(),
  source_id uuid not null references sources(id) on delete cascade,
  quote text not null,
  context text not null default '',
  created_at timestamptz not null default now()
);
create index if not exists page_snippets_source_id_idx on page_snippets(source_id);

create table if not exists bookmarks (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  chat_id uuid not null references chats(id) on delete cascade,
  created_at timestamptz not null default now(),
  unique (user_id, chat_id)
);

-- +goose Down
drop table if exists bookmarks;
drop table if exists page_snippets;
drop table if exists sources;
drop table if exists search_results;
drop table if exists search_queries;
drop table if exists run_steps;
drop table if exists runs;
drop table if exists messages;
drop table if exists chats;
drop table if exists refresh_tokens;
drop table if exists users;

