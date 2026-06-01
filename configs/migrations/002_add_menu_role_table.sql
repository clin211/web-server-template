-- 002_add_menu_role_table.sql
-- 创建菜单角色关联表，实现菜单与角色的多对多关系

-- 创建菜单角色关联表
CREATE TABLE IF NOT EXISTS public.menu_role (
    id          BIGSERIAL PRIMARY KEY,
    menu_id     VARCHAR(64) NOT NULL,
    role_id     VARCHAR(64) NOT NULL,
    created_at  TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT menu_role_menu_id_role_id_key UNIQUE (menu_id, role_id)
);

-- 添加外键约束
ALTER TABLE public.menu_role
    ADD CONSTRAINT menu_role_menu_id_fkey
    FOREIGN KEY (menu_id) REFERENCES public.menu(menu_id) ON DELETE CASCADE;

ALTER TABLE public.menu_role
    ADD CONSTRAINT menu_role_role_id_fkey
    FOREIGN KEY (role_id) REFERENCES public.role(role_id) ON DELETE CASCADE;

-- 添加注释
COMMENT ON TABLE public.menu_role IS '菜单角色关联表，实现菜单与角色的多对多关系';
COMMENT ON COLUMN public.menu_role.id IS '内部主键ID（自增序列）';
COMMENT ON COLUMN public.menu_role.menu_id IS '菜单UUID（外键）';
COMMENT ON COLUMN public.menu_role.role_id IS '角色UUID（外键）';
COMMENT ON COLUMN public.menu_role.created_at IS '创建时间';

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_menu_role_menu_id ON public.menu_role(menu_id);
CREATE INDEX IF NOT EXISTS idx_menu_role_role_id ON public.menu_role(role_id);