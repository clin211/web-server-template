-- 003_add_menu_extended_fields.sql
-- 为 menu 表添加扩展字段，支持 Elegant Router 机制的完整元数据

-- 添加扩展字段
ALTER TABLE public.menu
    ADD COLUMN IF NOT EXISTS local_icon VARCHAR(100);

ALTER TABLE public.menu
    ADD COLUMN IF NOT EXISTS icon_font_size INT;

ALTER TABLE public.menu
    ADD COLUMN IF NOT EXISTS constant SMALLINT NOT NULL DEFAULT 0;

ALTER TABLE public.menu
    ADD COLUMN IF NOT EXISTS active_menu VARCHAR(100);

ALTER TABLE public.menu
    ADD COLUMN IF NOT EXISTS hide_in_menu SMALLINT NOT NULL DEFAULT 0;

ALTER TABLE public.menu
    ADD COLUMN IF NOT EXISTS keep_alive SMALLINT NOT NULL DEFAULT 0;

ALTER TABLE public.menu
    ADD COLUMN IF NOT EXISTS href VARCHAR(500);

-- 添加注释
COMMENT ON COLUMN public.menu.local_icon IS '本地图标（可选）';
COMMENT ON COLUMN public.menu.icon_font_size IS '图标大小（可选）';
COMMENT ON COLUMN public.menu.constant IS '常量路由（0=否,1=是），不参与权限过滤';
COMMENT ON COLUMN public.menu.active_menu IS '当前激活的菜单（用于面包屑）';
COMMENT ON COLUMN public.menu.hide_in_menu IS '在菜单中隐藏（0=否,1=是）';
COMMENT ON COLUMN public.menu.keep_alive IS '页面缓存（0=否,1=是）';
COMMENT ON COLUMN public.menu.href IS '外链地址';

-- 创建缺失的索引
CREATE INDEX IF NOT EXISTS idx_menu_constant ON public.menu(constant)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_menu_hide_in_menu ON public.menu(hide_in_menu)
    WHERE deleted_at IS NULL;