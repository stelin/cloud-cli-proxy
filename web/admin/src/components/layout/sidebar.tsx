import { Link, useRouterState } from "@tanstack/react-router";
import {
  LayoutDashboard,
  Users,
  Globe,
  Server,
  ListTodo,
  ScrollText,
  LogOut,
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
    <aside className="flex w-60 flex-col border-r bg-background">
      <div className="flex h-16 items-center border-b px-6">
        <span className="text-lg font-semibold">Cloud CLI Proxy</span>
      </div>

      <nav className="flex-1 space-y-1 p-3">
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
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
              )}
            >
              <item.icon className="h-4 w-4" />
              {item.label}
            </Link>
          );
        })}
      </nav>

      <div className="border-t p-3">
        <button
          onClick={logout}
          className="flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
        >
          <LogOut className="h-4 w-4" />
          退出登录
        </button>
      </div>
    </aside>
  );
}
