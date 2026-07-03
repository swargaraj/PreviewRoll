import { useState } from "react";
import { Box, LayoutGrid, Plus } from "lucide-react";
import { Button } from "@previewroll/ui/components/button";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@previewroll/ui/components/empty";
import { Spinner } from "@previewroll/ui/components/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@previewroll/ui/components/table";

import type { Route } from "./+types/index";
import { useConnection } from "@/context/connection-context";
import { PageHeader } from "@/components/page-header";
import { SearchInput } from "@/components/search-input";
import { PaginationBar } from "@/components/pagination-bar";
import { CreateProjectDialog } from "./create-project-dialog";
import { usePaginatedQuery } from "@/hooks/use-paginated-query";
import { listProjects } from "@/services/projects";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Projects · PreviewRoll" },
    { name: "description", content: "Manage your projects" },
  ];
}

export default function Projects() {
  const { connection } = useConnection();
  const [dialogOpen, setDialogOpen] = useState(false);

  const {
    items: projects,
    page,
    totalPages,
    total,
    search,
    isLoading,
    setPage,
    setSearch,
  } = usePaginatedQuery({
    connection,
    queryKey: ["projects"],
    fetcher: listProjects,
  });

  return (
    <div className="flex flex-1 flex-col gap-4">
      <PageHeader title="Projects" description={`${total} project${total !== 1 ? "s" : ""}`}>
        <SearchInput value={search} onChange={setSearch} placeholder="Search projects" />
        <div className="flex-1" />
        <CreateProjectDialog open={dialogOpen} onOpenChange={setDialogOpen} />
      </PageHeader>

      {isLoading ? (
        <div className="flex flex-1 items-center justify-center py-20">
          <Spinner className="size-6 text-muted-foreground" />
        </div>
      ) : projects.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyMedia>
              <LayoutGrid />
            </EmptyMedia>
            <EmptyTitle>No projects</EmptyTitle>
            <EmptyDescription>
              {search ? "No projects match your search." : "Create a project to get started"}
            </EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            {!search && (
              <Button size="lg" onClick={() => setDialogOpen(true)}>
                <Plus className="size-4" />
                Create project
              </Button>
            )}
          </EmptyContent>
        </Empty>
      ) : (
        <>
          <Table variant="card">
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Repository</TableHead>
                <TableHead>Provider</TableHead>
                <TableHead>Created</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {projects.map((project) => (
                <TableRow key={project.id}>
                  <TableCell className="font-medium">{project.name}</TableCell>
                  <TableCell className="text-muted-foreground max-w-75 truncate">
                    {project.repo_url}
                  </TableCell>
                  <TableCell>
                    <span className="capitalize">{project.vcs_provider}</span>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {project.created_at ? new Date(project.created_at).toLocaleDateString() : "—"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <PaginationBar page={page} totalPages={totalPages} onPageChange={setPage} />
        </>
      )}
    </div>
  );
}
