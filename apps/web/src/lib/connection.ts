const STORAGE_KEY = "previewroll_connection";

export interface SavedConnection {
  server: string;
  username: string;
}

const DEFAULTS: SavedConnection = {
  server: "",
  username: "",
};

export function loadSavedConnection(): SavedConnection {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw);
      return { server: parsed.server ?? "", username: parsed.username ?? "" };
    }
  } catch {}
  return DEFAULTS;
}

export function saveConnection(connection: SavedConnection): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(connection));
}

export function clearSavedConnection(): void {
  localStorage.removeItem(STORAGE_KEY);
}

export function buildServerUrl(connection: SavedConnection, path: string): string {
  const base = connection.server.includes("://")
    ? connection.server
    : `https://${connection.server}`;

  const url = new URL(base);
  url.pathname = path.startsWith("/") ? path : `/${path}`;

  return url.toString();
}
