import { SearchIcon } from "lucide-react";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@previewroll/ui/components/input-group";

interface SearchInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

export function SearchInput({ value, onChange, placeholder = "Search" }: SearchInputProps) {
  return (
    <div className="relative w-64">
      <InputGroup>
        <InputGroupInput
          aria-label="Search"
          placeholder={placeholder}
          type="search"
          size="lg"
          value={value}
          onChange={(e) => onChange(e.target.value)}
        />
        <InputGroupAddon>
          <SearchIcon aria-hidden="true" className="text-muted-foreground" />
        </InputGroupAddon>
      </InputGroup>
    </div>
  );
}
