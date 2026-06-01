-- 添加菜单国际化key字段
-- 运行: 在 PostgreSQL 中执行此 SQL

ALTER TABLE public.menu ADD COLUMN IF NOT EXISTS i18n_key varchar(100);
COMMENT ON COLUMN public.menu.i18n_key IS '国际化key（用于前端翻译）';

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_menu_i18n_key ON public.menu(i18n_key);
