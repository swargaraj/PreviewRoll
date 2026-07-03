import type { SavedConnection } from "@/lib/connection";
import { apiRequest, type ApiResponse } from "@/lib/api";

export interface Project {
  id: number;
  user_id: number;
  name: string;
  repo_url: string;
  vcs_provider: string;
  webhook_secret: string;
  created_at: string | null;
  updated_at: string | null;
}

export interface PaginatedProjects {
  items: Project[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface CreateProjectParams {
  name: string;
  repo_url: string;
  vcs_provider?: string;
}

export async function listProjects(
  connection: SavedConnection,
  page: number = 1,
  pageSize: number = 10,
  search?: string,
): Promise<ApiResponse<PaginatedProjects>> {
  const params = new URLSearchParams({
    page: String(page),
    page_size: String(pageSize),
  });
  if (search) params.set("search", search);

  return apiRequest<PaginatedProjects>(connection, {
    method: "GET",
    path: `/api/v1/projects?${params}`,
  });
}

export async function createProject(
  connection: SavedConnection,
  params: CreateProjectParams,
): Promise<ApiResponse<{ id: number }>> {
  return apiRequest<{ id: number }>(connection, {
    method: "POST",
    path: "/api/v1/projects",
    body: params,
  });
}

export async function deleteProject(connection: SavedConnection, id: number): Promise<ApiResponse> {
  return apiRequest(connection, {
    method: "DELETE",
    path: `/api/v1/projects/${id}`,
  });
}
