import { Link, useLocation } from "react-router-dom";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard,
  Image,
  GitBranch,
  Puzzle,
  Globe,
  FileText,
  Tags,
  Settings,
  Bot,
  BarChart3,
} from "lucide-react";

interface NavItem {
  label: string;
  path: string;
  icon: React.ReactNode;
}

const mainNav: NavItem[] = [
  { label: "Dashboard", path: "/admin/dashboard", icon: <LayoutDashboard className="h-4 w-4" /> },
  { label: "Conteúdo", path: "/admin/posts", icon: <FileText className="h-4 w-4" /> },
  { label: "Categorias", path: "/admin/categories", icon: <Tags className="h-4 w-4" /> },
  { label: "Mídia", path: "/admin/media", icon: <Image className="h-4 w-4" /> },
];

const workflowNav: NavItem[] = [
  { label: "Workflows", path: "/admin/workflow", icon: <GitBranch className="h-4 w-4" /> },
  { label: "Plugins", path: "/admin/plugins", icon: <Puzzle className="h-4 w-4" /> },
];

const systemNav: NavItem[] = [
  { label: "Sites", path: "/admin/sites", icon: <Globe className="h-4 w-4" /> },
  { label: "AI", path: "/admin/ai", icon: <Bot className="h-4 w-4" /> },
  { label: "Relatórios", path: "/admin/reports", icon: <BarChart3 className="h-4 w-4" /> },
  { label: "Configurações", path: "/admin/settings", icon: <Settings className="h-4 w-4" /> },
];

interface SidebarProps {
  onNavigate?: () => void;
}

export function Sidebar({ onNavigate }: SidebarProps) {
  const location = useLocation();

  function isActive(path: string): boolean {
    if (path === "/admin/dashboard") return location.pathname === path;
    return location.pathname.startsWith(path);
  }

  return (
    <div className="flex h-full flex-col gap-1">
      <div className="flex h-14 items-center gap-2 border-b border-sidebar-border px-4">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-brand-600 text-xs font-bold text-white">
          N
        </div>
        <span className="text-sm font-semibold text-sidebar-foreground">Nexora CMS</span>
      </div>

      <div className="flex-1 overflow-y-auto px-3 py-2">
        <NavSection title="Principal" items={mainNav} isActive={isActive} onNavigate={onNavigate} />
        <NavSection title="Workflows" items={workflowNav} isActive={isActive} onNavigate={onNavigate} />
        <NavSection title="Sistema" items={systemNav} isActive={isActive} onNavigate={onNavigate} />
      </div>
    </div>
  );
}

function NavSection({
  title,
  items,
  isActive,
  onNavigate,
}: {
  title: string;
  items: NavItem[];
  isActive: (path: string) => boolean;
  onNavigate?: () => void;
}) {
  const hasActive = items.some((item) => isActive(item.path));

  return (
    <div className="mb-4">
      <p className="mb-1 px-2 text-xs font-medium uppercase tracking-wider text-sidebar-foreground/50">
        {title}
      </p>
      <div className="space-y-0.5">
        {items.map((item) => (
          <Link
            key={item.path}
            to={item.path}
            onClick={onNavigate}
            className={cn(
              "flex items-center gap-3 rounded-md px-2 py-1.5 text-sm font-medium transition-colors",
              isActive(item.path)
                ? "bg-sidebar-muted text-sidebar-foreground"
                : "text-sidebar-foreground/70 hover:bg-sidebar-muted/50 hover:text-sidebar-foreground",
            )}
          >
            {item.icon}
            {item.label}
          </Link>
        ))}
      </div>
    </div>
  );
}
