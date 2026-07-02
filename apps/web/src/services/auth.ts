import type { SavedConnection } from "@/lib/connection";
import { apiRequest, type ApiResponse } from "@/lib/api";

export interface LoginCredentials {
  username: string;
  password: string;
}

export interface User {
  id: number;
  username: string;
  created_at: string;
}

export async function login(
  connection: SavedConnection,
  credentials: LoginCredentials
): Promise<ApiResponse> {
  return apiRequest(connection, {
    method: "POST",
    path: "/api/auth/login",
    body: credentials,
  });
}

export async function me(connection: SavedConnection): Promise<ApiResponse<User>> {
  return apiRequest<User>(connection, {
    method: "GET",
    path: "/api/auth/me",
  });
}

export async function logout(connection: SavedConnection): Promise<ApiResponse> {
  return apiRequest(connection, {
    method: "POST",
    path: "/api/auth/logout",
  });
}
