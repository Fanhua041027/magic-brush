import { createClient } from 'https://esm.sh/@supabase/supabase-js@2'

const supabaseUrl = Deno.env.get('SUPABASE_URL')!
const serviceKey = Deno.env.get('SUPABASE_SERVICE_ROLE_KEY')!
const admin = createClient(supabaseUrl, serviceKey)

Deno.serve(async (req) => {
  if (req.method !== 'POST') return new Response('Method Not Allowed', { status: 405 })
  const token = req.headers.get('Authorization')?.replace('Bearer ', '')
  if (!token) return Response.json({ error: '未登录' }, { status: 401 })
  const caller = await admin.auth.getUser(token)
  if (caller.error) return Response.json({ error: '登录已失效' }, { status: 401 })
  const { data: profile } = await admin.from('profiles').select('role,enabled').eq('id', caller.data.user.id).single()
  if (!profile || profile.role !== 'admin' || !profile.enabled) return Response.json({ error: '需要管理员权限' }, { status: 403 })

  const body = await req.json()
  if (body.action === 'create') {
    if (typeof body.username !== 'string' || typeof body.password !== 'string' || body.username.length < 3 || body.password.length < 8) return Response.json({ error: '账号至少3位，密码至少8位' }, { status: 400 })
    const created = await admin.auth.admin.createUser({ email: `${body.username}@magicbrush.local`, password: body.password, email_confirm: true, user_metadata: { username: body.username, role: 'user', must_change_password: false } })
    if (created.error) return Response.json({ error: created.error.message }, { status: 400 })
    return Response.json({ ok: true, id: created.data.user.id })
  }
  if (body.action === 'delete') {
    if (typeof body.user_id !== 'string') return Response.json({ error: '用户 ID 无效' }, { status: 400 })
    const result = await admin.auth.admin.deleteUser(body.user_id)
    return result.error ? Response.json({ error: result.error.message }, { status: 400 }) : Response.json({ ok: true })
  }
  return Response.json({ error: '不支持的操作' }, { status: 400 })
})
