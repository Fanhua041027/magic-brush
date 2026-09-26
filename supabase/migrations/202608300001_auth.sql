create table if not exists public.profiles (
  id uuid primary key references auth.users(id) on delete cascade,
  username text not null unique check (char_length(username) >= 3),
  role text not null default 'user' check (role in ('admin', 'user')),
  enabled boolean not null default true,
  created_at timestamptz not null default now(),
  last_login_at timestamptz,
  total_seconds bigint not null default 0,
  must_change_password boolean not null default false
);

alter table public.profiles enable row level security;
create or replace function public.is_admin() returns boolean language sql stable security definer set search_path = public as $$
  select exists(select 1 from public.profiles where id = auth.uid() and role = 'admin' and enabled);
$$;
create policy "users can read their profile" on public.profiles for select using (id = auth.uid() or public.is_admin());
create policy "admins can update profiles" on public.profiles for update using (public.is_admin()) with check (public.is_admin());

create or replace function public.handle_new_user() returns trigger language plpgsql security definer set search_path = public as $$
begin
  insert into public.profiles (id, username, role, must_change_password)
  values (new.id, coalesce(new.raw_user_meta_data->>'username', split_part(new.email, '@', 1)), coalesce(new.raw_user_meta_data->>'role', 'user'), coalesce((new.raw_user_meta_data->>'must_change_password')::boolean, false))
  on conflict (id) do nothing;
  return new;
end;
$$;
drop trigger if exists on_auth_user_created on auth.users;
create trigger on_auth_user_created after insert on auth.users for each row execute procedure public.handle_new_user();
