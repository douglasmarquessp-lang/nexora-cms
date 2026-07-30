import { useSiteStore } from "@/stores/site";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export function SiteSwitcher() {
  const { sites, currentSite, setCurrentSite, isLoading } = useSiteStore();

  if (isLoading || sites.length === 0) return null;

  return (
    <Select
      value={currentSite?.id || ""}
      onValueChange={(value) => {
        const site = sites.find((s) => s.id === value);
        if (site) setCurrentSite(site);
      }}
    >
      <SelectTrigger className="h-8 w-[180px] border-none bg-sidebar-muted/50 text-xs text-sidebar-foreground">
        <SelectValue placeholder="Selecionar site" />
      </SelectTrigger>
      <SelectContent>
        {sites.map((site) => (
          <SelectItem key={site.id} value={site.id}>
            <div className="flex items-center gap-2">
              <div className="flex h-5 w-5 items-center justify-center rounded bg-brand-600/20 text-[10px] font-bold text-brand-600">
                {site.name.charAt(0).toUpperCase()}
              </div>
              <span className="text-xs">{site.name}</span>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
