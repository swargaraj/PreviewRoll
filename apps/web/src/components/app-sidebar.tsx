import { Link, useLocation, useNavigate } from "react-router";
import {
  LayoutGrid,
  Box,
  Rocket,
  Waypoints,
  HardDrive,
  Logs,
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
  { label: "Overview", to: "/", icon: LayoutGrid },
  { label: "Projects", to: "/projects", icon: Box },
  { label: "Deployments", to: "/deployments", icon: Rocket },
  { label: "Previews", to: "/previews", icon: Waypoints },
  { label: "Workers", to: "/workers", icon: HardDrive },
  { label: "Logs", to: "/logs", icon: Logs },
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
      <SidebarHeader className="pt-6 px-6">
        <Link to="/">
            <img src="/full-logo.png" alt="PreviewRoll" className="w-30 h-auto" />
        </Link>
      </SidebarHeader>
      <SidebarContent className="p-2">
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu className="gap-2">
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
      <SidebarFooter className="p-2">
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
