import { Link, useLocation, useNavigate } from "react-router";
import {
  GitPullRequestArrow,
  LayoutGrid,
  Box,
  Rocket,
  Waypoints,
  HardDrive,
  Logs,
  Settings,
  FileText,
  HelpCircle,
  LogOut,
} from "lucide-react";

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@previewroll/ui/components/sidebar";

import { useConnection } from "@/context/connection-context";
import { useAuth } from "@/context/auth-context";
import { logout } from "@/services/auth";

const mainNav = [
  { label: "Overview", to: "/dashboard", icon: LayoutGrid },
  { label: "Projects", to: "/dashboard/projects", icon: Box },
  { label: "Deployments", to: "/dashboard/deployments", icon: Rocket },
  { label: "Previews", to: "/dashboard/previews", icon: Waypoints },
  { label: "Workers", to: "/dashboard/workers", icon: HardDrive },
  { label: "Logs", to: "/dashboard/logs", icon: Logs },
  { label: "Settings", to: "/dashboard/settings", icon: Settings },
];

const secondaryNav = [
  { label: "Docs", to: "/docs", icon: FileText, external: true },
  { label: "Support", to: "/support", icon: HelpCircle, external: true },
];

export function AppSidebar() {
  const location = useLocation();
  const navigate = useNavigate();
  const { connection } = useConnection();
  const { checkAuth } = useAuth();

  const handleLogout = async () => {
    await logout(connection);
    await checkAuth();
    navigate("/login");
  };

  return (
    <Sidebar className="border-none">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" render={<Link to="/dashboard" />}>
              <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                <GitPullRequestArrow className="size-4" />
              </div>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">PreviewRoll</span>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {mainNav.map((item) => (
                <SidebarMenuItem key={item.to}>
                  <SidebarMenuButton
                    isActive={location.pathname === item.to}
                    render={<Link to={item.to} />}
                  >
                    <item.icon className="size-4" />
                    {item.label}
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {secondaryNav.map((item) => (
                <SidebarMenuItem key={item.to}>
                  <SidebarMenuButton
                    render={
                      item.external ? (
                        <a href={item.to} target="_blank" rel="noopener noreferrer" />
                      ) : (
                        <Link to={item.to} />
                      )
                    }
                  >
                    <item.icon className="size-4" />
                    {item.label}
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
              <SidebarMenuItem>
                <SidebarMenuButton onClick={handleLogout}>
                  <LogOut className="size-4" />
                  Logout
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarFooter>
    </Sidebar>
  );
}
