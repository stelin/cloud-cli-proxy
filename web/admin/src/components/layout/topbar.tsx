import { useRouterState } from "@tanstack/react-router";
import { getRole } from "@/lib/auth";

const pageTitles: Record<string, string> = {
  "/": "仪表板",
  "/users": "用户管理",
  "/egress-ips": "出口 IP",
  "/hosts": "主机管理",
  "/tasks": "任务列表",
  "/portal": "我的面板",
};

export function Topbar() {
  const routerState = useRouterState();
  const pathname = routerState.location.pathname;
  const title =
    pageTitles[pathname] ??
    (pathname.startsWith("/portal/hosts/") ? "主机详情" : "管理后台");

  const role = getRole();
  const roleLabel = role === "admin" ? "管理员" : "用户";

  return (
    <header className="flex h-16 items-center justify-between border-b bg-background px-6">
      <h2 className="text-lg font-semibold">{title}</h2>
      <span className="text-sm text-muted-foreground">{roleLabel}</span>
    </header>
  );
}
