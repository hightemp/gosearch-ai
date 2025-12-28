-- +goose Up

create table if not exists page_cache (
  url text primary key,
  title text not null default '',
  content text not null default '',
  snippets jsonb not null default '[]'::jsonb,
  fetched_at timestamptz not null default now()
);
create index if not exists page_cache_fetched_at_idx on page_cache(fetched_at desc);

-- +goose Down
drop table if exists page_cache;
