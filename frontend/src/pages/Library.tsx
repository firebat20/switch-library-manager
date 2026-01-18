import { useEffect, useState } from "react";
import { UpdateLocalLibrary } from "@/wailsjs/go/main/App";
import { main } from "@/wailsjs/go/models";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export default function Library() {
    const [libraryData, setLibraryData] = useState<main.LocalLibraryData | null>(null);
    const [loading, setLoading] = useState(false);
    const [filter, setFilter] = useState("");

    const loadLibrary = async () => {
        setLoading(true);
        try {
            // Passing empty string as argument if required, or adjust based on signature
            const data = await UpdateLocalLibrary("");
            setLibraryData(data);
        } catch (err) {
            console.error("Failed to load library", err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadLibrary();
    }, []);

    const filteredGames = libraryData?.library_data?.filter((game) =>
        game.name.toLowerCase().includes(filter.toLowerCase()) ||
        game.titleId.toLowerCase().includes(filter.toLowerCase())
    ) || [];

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h2 className="text-2xl font-bold tracking-tight">Library</h2>
                <div className="flex items-center space-x-2">
                    <Button onClick={loadLibrary} disabled={loading}>
                        {loading ? "Refreshing..." : "Refresh Library"}
                    </Button>
                </div>
            </div>

            <Card>
                <CardHeader className="pb-3">
                    <CardTitle>Installed Games ({filteredGames.length})</CardTitle>
                    <div className="pt-2">
                        <Input
                            placeholder="Filter by title or ID..."
                            value={filter}
                            onChange={(e) => setFilter(e.target.value)}
                            className="max-w-sm"
                        />
                    </div>
                </CardHeader>
                <CardContent>
                    <div className="rounded-md border">
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead className="w-[50px]">#</TableHead>
                                    <TableHead className="w-[80px]">Icon</TableHead>
                                    <TableHead>Title</TableHead>
                                    <TableHead>Title ID</TableHead>
                                    <TableHead>Region</TableHead>
                                    <TableHead>Type</TableHead>
                                    <TableHead>Update</TableHead>
                                    <TableHead>Version</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {filteredGames.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={8} className="h-24 text-center">
                                            No results.
                                        </TableCell>
                                    </TableRow>
                                ) : (
                                    filteredGames.map((game, index) => (
                                        <TableRow key={game.id}>
                                            <TableCell>{index + 1}</TableCell>
                                            <TableCell>
                                                {game.icon ? (
                                                    <img
                                                        src={game.icon}
                                                        alt={game.name}
                                                        className="h-10 w-10 rounded object-cover"
                                                        loading="lazy"
                                                    />
                                                ) : (
                                                    <div className="h-10 w-10 rounded bg-muted" />
                                                )}
                                            </TableCell>
                                            <TableCell className="font-medium">{game.name}</TableCell>
                                            <TableCell className="font-mono text-xs">{game.titleId}</TableCell>
                                            <TableCell>{game.region}</TableCell>
                                            <TableCell>
                                                <Badge variant="outline">{game.type}</Badge>
                                            </TableCell>
                                            <TableCell>{game.update}</TableCell>
                                            <TableCell>{game.version}</TableCell>
                                        </TableRow>
                                    ))
                                )}
                            </TableBody>
                        </Table>
                    </div>
                </CardContent>
            </Card>
        </div>
    );
}
