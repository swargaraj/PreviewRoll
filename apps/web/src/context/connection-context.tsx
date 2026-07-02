import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import type { SavedConnection } from "@/lib/connection";
import { loadSavedConnection, saveConnection, clearSavedConnection } from "@/lib/connection";

interface ConnectionContextValue {
  connection: SavedConnection;
  isConnected: boolean;
  setConnection: (connection: SavedConnection) => void;
  disconnect: () => void;
}

const ConnectionContext = createContext<ConnectionContextValue | null>(null);

export function ConnectionProvider({ children }: { children: ReactNode }) {
  const [connection, setConnectionState] = useState<SavedConnection>(loadSavedConnection);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    const saved = loadSavedConnection();
    setConnectionState(saved);
    setIsConnected(!!saved.server);
  }, []);

  const setConnection = (conn: SavedConnection) => {
    saveConnection(conn);
    setConnectionState(conn);
    setIsConnected(!!conn.server);
  };

  const disconnect = () => {
    clearSavedConnection();
    setConnectionState({ server: "", username: "" });
    setIsConnected(false);
  };

  return (
    <ConnectionContext.Provider value={{ connection, isConnected, setConnection, disconnect }}>
      {children}
    </ConnectionContext.Provider>
  );
}

export function useConnection() {
  const context = useContext(ConnectionContext);
  if (!context) {
    throw new Error("useConnection must be used within a ConnectionProvider");
  }
  return context;
}
