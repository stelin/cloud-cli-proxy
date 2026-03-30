import { Link, useRouterState } from "@tanstack/react-router";
import {
  LayoutDashboard,
  Users,
  Globe,
  Server,
  ListTodo,
  ScrollText,
  LogOut,
  Cloud,
} from "lucide-react";
import { logout } from "@/lib/auth";
import { cn } from "@/lib/utils";

const navItems = [
  { label: "仪表板", to: "/", icon: LayoutDashboard },
  { label: "用户管理", to: "/users", icon: Users },
  { label: "出口 IP", to: "/egress-ips", icon: Globe },
  { label: "主机管理", to: "/hosts", icon: Server },
  { label: "任务列表", to: "/tasks", icon: ListTodo },
  { label: "事件日志", to: "/events", icon: ScrollText },
] as const;

export function Sidebar() {
  const routerState = useRouterState();
  const currentPath = routerState.location.pathname;

  return (
    <aside className="flex w-60 flex-col bg-sidebar text-sidebar-foreground">
      <div className="flex h-16 items-center gap-2.5 px-5">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
          <Cloud className="h-4.5 w-4.5 text-primary-foreground" />
        </div>
        <div className="flex flex-col">
          <span className="text-sm font-semibold text-white leading-tight">Cloud CLI</span>
          <span className="text-[10px] font-medium text-sidebar-muted tracking-wider uppercase">Proxy</span>
        </div>
      </div>

      <nav className="flex-1 px-3 pt-4 space-y-0.5">
        <p className="px-3 pb-2 text-[10px] font-semibold uppercase tracking-wider text-sidebar-muted">
          管理
        </p>
        {navItems.map((item) => {
          const isActive =
            item.to === "/"
              ? currentPath === "/"
              : currentPath.startsWith(item.to);

          return (
            <Link
              key={item.to}
              to={item.to}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-[13px] font-medium transition-all duration-150",
                isActive
                  ? "bg-sidebar-accent text-white shadow-sm"
                  : "text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-white",
              )}
            >
              <item.icon className={cn("h-4 w-4", isActive && "text-primary-foreground")} />
              {item.label}
            </Link>
          );
        })}
      </nav>

      <div className="px-3 pb-4">
        <button
          onClick={logout}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-[13px] font-medium text-sidebar-foreground/50 transition-colors hover:bg-sidebar-accent/50 hover:text-white"
        >
          <LogOut className="h-4 w-4" />
          退出登录
        </button>
      </div>
    </aside>
  );
}
