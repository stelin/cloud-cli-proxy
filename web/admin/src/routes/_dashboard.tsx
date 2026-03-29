import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { isAuthenticated, getRole } from "@/lib/auth";
import { Sidebar } from "@/components/layout/sidebar";
import { Topbar } from "@/components/layout/topbar";

export const Route = createFileRoute("/_dashboard")({
  beforeLoad: () => {
    if (!isAuthenticated()) {
      throw redirect({ to: "/login" });
    }
    const role = getRole();
    if (role !== "admin") {
      throw redirect({ to: "/portal" });
    }
  },
  component: DashboardLayout,
});

function DashboardLayout() {
  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Topbar />
        <main className="flex-1 overflow-y-auto bg-muted/40 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
