import { Link, useLocation } from "react-router-dom";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { ModeToggle } from "./mode-toggle";

export default function Layout({ children }: { children: React.ReactNode }) {
    const location = useLocation();

    const navItems = [
        { name: "Library", path: "/" },
        { name: "Missing Update", path: "/missing-update" },
        { name: "Missing DLC", path: "/missing-dlc" },
        { name: "Organize", path: "/organize" },
        { name: "Issues", path: "/issues" },
    ];

    return (
        <div className="min-h-screen bg-background text-foreground flex flex-col">
            <header className="border-b border-border bg-card">
                <div className="container mx-auto px-4 py-3 flex items-center justify-between">
                    <div className="font-bold text-xl">Switch Library Manager</div>
                    <nav className="flex space-x-2">
                        {navItems.map((item) => (
                            <Link key={item.path} to={item.path}>
                                <Button
                                    variant={location.pathname === item.path ? "secondary" : "ghost"}
                                    className={cn(
                                        "text-sm font-medium transition-colors",
                                        location.pathname === item.path
                                            ? "bg-secondary text-secondary-foreground"
                                            : "text-muted-foreground hover:text-foreground"
                                    )}
                                >
                                    {item.name}
                                </Button>
                            </Link>
                        ))}
                    </nav>
                    <div>
                        {/* Settings or other actions could go here */}
                        <Button variant="ghost" size="sm">Settings</Button>
                        <ModeToggle />
                    </div>
                </div>
            </header>
            <main className="flex-1 container mx-auto px-4 py-6">
                {children}
            </main>
        </div>
    );
}
