# Supabase 部署与管理员账号

## 1. 配置项目

在本机安装 Supabase CLI 后，在项目根目录执行：

```powershell
supabase login
supabase link --project-ref <项目 ID>
supabase db push
supabase functions deploy admin-users --no-verify-jwt
```

`admin-users` 使用 Supabase 的 `SUPABASE_SERVICE_ROLE_KEY`，该密钥只在 Edge Function 服务端使用，不能写入客户端。

## 2. 创建第一个管理员

在 Supabase Dashboard 的 Authentication → Users 中创建一个邮箱密码用户（邮箱可使用 `admin@magicbrush.local`），记下用户 UUID；然后在 SQL Editor 执行：

```sql
insert into public.profiles (id, username, role, enabled, must_change_password)
values ('替换为用户 UUID', 'admin', 'admin', true, false)
on conflict (id) do update set role = 'admin', enabled = true;
```

客户端使用“用户名”登录，程序会把它映射为 `<用户名>@magicbrush.local`。管理员在后台创建的用户会自动写入 `auth.users` 和 `public.profiles`，随后可直接用创建时的账号密码登录。

## 3. 启用远程认证

启动客户端前设置：

```powershell
$env:SUPABASE_URL = 'https://<项目 ID>.supabase.co'
$env:SUPABASE_ANON_KEY = '<anon key>'
```

未设置这两个变量时，程序仍使用原有本地认证，便于离线开发。
