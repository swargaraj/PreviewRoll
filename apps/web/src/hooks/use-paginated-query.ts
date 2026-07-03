import { useQuery } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import type { SavedConnection } from "@/lib/connection";
import type { ApiResponse } from "@/lib/api";

export interface PaginatedData<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

interface UsePaginatedQueryOptions<T> {
  connection: SavedConnection;
  queryKey: string[];
  fetcher: (
    connection: SavedConnection,
    page: number,
    pageSize: number,
    search?: string,
  ) => Promise<ApiResponse<PaginatedData<T>>>;
  pageSize?: number;
}

export function usePaginatedQuery<T>({
  connection,
  queryKey,
  fetcher,
  pageSize = 10,
}: UsePaginatedQueryOptions<T>) {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");

  const { data, isLoading, refetch } = useQuery({
    queryKey: [...queryKey, page, debouncedSearch, connection.server],
    queryFn: () => fetcher(connection, page, pageSize, debouncedSearch || undefined),
  });

  const items = data?.ok && data.data ? data.data.items : [];
  const totalPages = data?.ok && data.data ? data.data.total_pages : 1;
  const total = data?.ok && data.data ? data.data.total : 0;

  const goToPage = useCallback((p: number) => {
    setPage(p);
  }, []);

  const goToNextPage = useCallback(() => {
    setPage((prev) => Math.min(prev + 1, totalPages));
  }, [totalPages]);

  const goToPreviousPage = useCallback(() => {
    setPage((prev) => Math.max(prev - 1, 1));
  }, []);

  const updateSearch = useCallback((value: string) => {
    setSearch(value);
    setPage(1);
    setDebouncedSearch(value);
  }, []);

  return {
    items,
    page,
    totalPages,
    total,
    search,
    isLoading,
    refetch,
    setPage: goToPage,
    nextPage: goToNextPage,
    previousPage: goToPreviousPage,
    setSearch: updateSearch,
  };
}
