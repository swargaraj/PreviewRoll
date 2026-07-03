import { useEffect } from "react";
import { Outlet, useNavigate } from "react-router";

import type { Route } from "./+types/layout";
import { useAuth } from "@/context/auth-context";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@previewroll/ui/components/sidebar";
import { AppSidebar } from "@/components/app-sidebar";

export function meta({}: Route.MetaArgs) {
  return [{ title: "PreviewRoll" }, { name: "description", content: "PreviewRoll" }];
}

export default function DashboardLayout() {
  const { isAuthenticated, isLoading } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoading) {
      if (!isAuthenticated) {
        navigate("/login", { replace: true });
      }
    }
  }, [isLoading, isAuthenticated, navigate]);

  if (isLoading || !isAuthenticated) {
    return (
      <div className="flex min-h-svh items-center justify-center">
        <p className="shimmer text-muted-foreground text-lg">PreviewRoll</p>
      </div>
    );
  }

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <div className="flex items-center gap-2 pt-4">
            <SidebarTrigger />
          </div>
          <Outlet />
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
