create table public.user (
    id uuid primary key default uuid_generate_v4(),

    email      text unique,
    title      text,
    first_name text not null, 
    last_name  text not null, 

    session_token text unique check(length(session_token) >= 16),
    
    created_at  timestamptz not null default now(),
    updated_at  timestamptz not null default now(),
    disabled_at timestamptz
);