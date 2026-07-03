import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

const vcsProviderPatterns: [string, string][] = [
  ["github.com", "github"],
  ["gitlab.com", "gitlab"],
  ["bitbucket.org", "bitbucket"],
  ["codeberg.org", "codeberg"],
];

export function detectVcsProvider(url: string): string | null {
  const lower = url.toLowerCase();
  for (const [host, provider] of vcsProviderPatterns) {
    if (lower.includes(host)) return provider;
  }
  return null;
}
