import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import type { User } from "@/services/auth";
import { me } from "@/services/auth";
import { useConnection } from "./connection-context";

interface AuthContextValue {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  checkAuth: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const { connection } = useConnection();
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const checkAuth = async () => {
    if (!connection.server) {
      setUser(null);
      setIsLoading(false);
      return;
    }

    setIsLoading(true);
    const result = await me(connection);
    setUser(result.ok ? result.data : null);
    setIsLoading(false);
  };

  useEffect(() => {
    checkAuth();
  }, [connection.server]);

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: !!user,
        checkAuth,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
