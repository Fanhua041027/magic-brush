-- Prevent client-controlled signup metadata from granting privileges.
create or replace function public.handle_new_user() returns trigger
language plpgsql security definer set search_path = public as $$
begin
  insert into public.profiles (id, username, role, must_change_password)
  values (
    new.id,
    left(coalesce(new.raw_user_meta_data->>'username', split_part(new.email, '@', 1)), 64),
    'user',
    false
  )
  on conflict (id) do nothing;
  return new;
end;
$$;

drop trigger if exists on_auth_user_created on auth.users;
create trigger on_auth_user_created after insert on auth.users
for each row execute procedure public.handle_new_user();
