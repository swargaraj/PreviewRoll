import { useCallback, useEffect, useRef, useState } from "react";
import { Box, Plus, SearchIcon } from "lucide-react";
import { Button } from "@previewroll/ui/components/button";
import {
  Dialog,
  DialogClose,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogPanel,
  DialogPopup,
  DialogTitle,
  DialogTrigger,
} from "@previewroll/ui/components/dialog";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@previewroll/ui/components/empty";
import { Field, FieldControl, FieldError, FieldLabel } from "@previewroll/ui/components/field";
import { Form } from "@previewroll/ui/components/form";
import { Input } from "@previewroll/ui/components/input";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@previewroll/ui/components/pagination";
import { Spinner } from "@previewroll/ui/components/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@previewroll/ui/components/table";

import type { Route } from "./+types/projects";
import { useConnection } from "@/context/connection-context";
import { PageHeader } from "@/components/page-header";
import {
  listProjects,
  createProject,
  type Project,
} from "@/services/projects";
import {InputGroup, InputGroupAddon, InputGroupInput} from "@previewroll/ui/components/input-group";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Projects · PreviewRoll" },
    { name: "description", content: "Manage your projects" },
  ];
}

export default function Projects() {
  const { connection } = useConnection();
  const [projects, setProjects] = useState<Project[]>([]);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const searchTimeout = useRef<ReturnType<typeof setTimeout>>(undefined);

  const fetchProjects = useCallback(async (p: number, q?: string) => {
    setLoading(true);
    const result = await listProjects(connection, p, 10, q);
    if (result.ok && result.data) {
      setProjects(result.data.items);
      setTotalPages(result.data.total_pages);
      setTotal(result.data.total);
      setPage(result.data.page);
    }
    setLoading(false);
  }, [connection]);

  useEffect(() => {
    fetchProjects(page);
  }, [fetchProjects, page]);

  const handleSearchChange = (value: string) => {
    setSearch(value);
    clearTimeout(searchTimeout.current);
    searchTimeout.current = setTimeout(() => {
      setPage(1);
      fetchProjects(1, value);
    }, 300);
  };

  const handleCreate = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError(null);
    setCreating(true);

    const formData = new FormData(event.currentTarget);
    const name = formData.get("name") as string;
    const repoUrl = formData.get("repo_url") as string;

    const result = await createProject(connection, {
      name,
      repo_url: repoUrl,
      vcs_provider: "github",
    });

    if (!result.ok) {
      setError(result.error || "Failed to create project");
      setCreating(false);
      return;
    }

    setDialogOpen(false);
    setCreating(false);
    setPage(1);
    fetchProjects(1, search);
  };

  return (
      <div className="flex flex-1 flex-col gap-4">
        <PageHeader
          title="Projects"
          description={`${total} project${total !== 1 ? "s" : ""}`}
        >
          <div className="relative w-64">
            <InputGroup>
              <InputGroupInput aria-label="Search" placeholder="Search Projects" type="search" size="lg"
                               value={search}
                               onChange={(e) => handleSearchChange(e.target.value)}
              />
              <InputGroupAddon>
                <SearchIcon aria-hidden="true" className="text-muted-foreground" />
              </InputGroupAddon>
            </InputGroup>
          </div>
          <div className="flex-1" />
          <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
            <DialogTrigger
              render={<Button size="lg" />}
            >
              <Plus className="size-4" />
              Create project
            </DialogTrigger>
            <DialogPopup>
              <Form onSubmit={handleCreate}>
                <DialogHeader>
                  <DialogTitle>Create project</DialogTitle>
                  <DialogDescription>
                    Add a new project to deploy previews for.
                  </DialogDescription>
                </DialogHeader>
                <DialogPanel>
                  <div className="flex flex-col gap-4">
                    <Field name="name">
                      <FieldLabel htmlFor="project-name">Name</FieldLabel>
                      <FieldControl
                        render={
                          <Input
                            id="project-name"
                            name="name"
                            placeholder="my-project"
                            size="lg"
                            required
                          />
                        }
                      />
                      <FieldError />
                    </Field>
                    <Field name="repo_url">
                      <FieldLabel htmlFor="project-repo">Repository URL</FieldLabel>
                      <FieldControl
                        render={
                          <Input
                            id="project-repo"
                            name="repo_url"
                            placeholder="https://github.com/user/repo"
                            size="lg"
                            required
                          />
                        }
                      />
                      <FieldError />
                    </Field>
                    {error && (
                      <p className="text-destructive text-sm">{error}</p>
                    )}
                  </div>
                </DialogPanel>
                <DialogFooter>
                  <DialogClose render={<Button variant="outline" />}>
                    Cancel
                  </DialogClose>
                  <Button type="submit" loading={creating}>
                    Create
                  </Button>
                </DialogFooter>
              </Form>
            </DialogPopup>
          </Dialog>
        </PageHeader>

        {loading ? (
          <div className="flex flex-1 items-center justify-center py-20">
            <Spinner className="size-6 text-muted-foreground" />
          </div>
        ) : projects.length === 0 ? (
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <Box />
              </EmptyMedia>
              <EmptyTitle>No projects</EmptyTitle>
              <EmptyDescription>
                {search ? "No projects match your search." : "Create a project to get started with preview deployments."}
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
                    <TableCell className="font-medium">
                      {project.name}
                    </TableCell>
                    <TableCell className="text-muted-foreground max-w-[300px] truncate">
                      {project.repo_url}
                    </TableCell>
                    <TableCell>
                      <span className="capitalize">{project.vcs_provider}</span>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {project.created_at
                        ? new Date(project.created_at).toLocaleDateString()
                        : "—"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>

            {totalPages > 1 && (
              <Pagination>
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious
                      href="#"
                      onClick={(e) => {
                        e.preventDefault();
                        if (page > 1) setPage(page - 1);
                      }}
                      aria-disabled={page <= 1}
                      className={page <= 1 ? "pointer-events-none opacity-50" : ""}
                    />
                  </PaginationItem>
                  {Array.from({ length: totalPages }, (_, i) => i + 1)
                    .filter((p) => {
                      if (totalPages <= 7) return true;
                      if (p === 1 || p === totalPages) return true;
                      if (Math.abs(p - page) <= 1) return true;
                      return false;
                    })
                    .reduce<(number | "ellipsis")[]>((acc, p, i, arr) => {
                      if (i > 0 && p - (arr[i - 1] as number) > 1) {
                        acc.push("ellipsis");
                      }
                      acc.push(p);
                      return acc;
                    }, [])
                    .map((item, i) =>
                      item === "ellipsis" ? (
                        <PaginationItem key={`ellipsis-${i}`}>
                          <span className="flex min-w-7 justify-center text-muted-foreground">...</span>
                        </PaginationItem>
                      ) : (
                        <PaginationItem key={item}>
                          <PaginationLink
                            href="#"
                            isActive={item === page}
                            onClick={(e) => {
                              e.preventDefault();
                              setPage(item);
                            }}
                          >
                            {item}
                          </PaginationLink>
                        </PaginationItem>
                      )
                    )}
                  <PaginationItem>
                    <PaginationNext
                      href="#"
                      onClick={(e) => {
                        e.preventDefault();
                        if (page < totalPages) setPage(page + 1);
                      }}
                      aria-disabled={page >= totalPages}
                      className={page >= totalPages ? "pointer-events-none opacity-50" : ""}
                    />
                  </PaginationItem>
                </PaginationContent>
              </Pagination>
            )}
          </>
        )}
      </div>
  );
}
