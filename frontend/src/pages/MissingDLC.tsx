import { useEffect, useState } from "react";
import { GetMissingDLC } from "@/wailsjs/go/main/App";
import { process } from "@/wailsjs/go/models";
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
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

export default function MissingDLC() {
    const [items, setItems] = useState<process.IncompleteTitle[]>([]);
    const [loading, setLoading] = useState(false);
    const [filter, setFilter] = useState("");

    const loadData = async () => {
        setLoading(true);
        try {
            const data = await GetMissingDLC();
            setItems(data || []);
        } catch (err) {
            console.error("Failed to load missing DLC", err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadData();
    }, []);

    const filteredItems = items.filter((item) =>
        item.Attributes.name?.toLowerCase().includes(filter.toLowerCase()) ||
        item.Attributes.id.toLowerCase().includes(filter.toLowerCase())
    );

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h2 className="text-2xl font-bold tracking-tight">Missing DLC</h2>
                <Button onClick={loadData} disabled={loading}>
                    {loading ? "Refreshing..." : "Refresh"}
                </Button>
            </div>

            <Card>
                <CardHeader className="pb-3">
                    <CardTitle>Items ({filteredItems.length})</CardTitle>
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
                                    <TableHead>Missing DLCs</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {filteredItems.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={6} className="h-24 text-center">
                                            No missing DLC found.
                                        </TableCell>
                                    </TableRow>
                                ) : (
                                    filteredItems.map((item: process.IncompleteTitle, index: number) => (
                                        <TableRow key={item.Attributes.id}>
                                            <TableCell>{index + 1}</TableCell>
                                            <TableCell>
                                                {item.Attributes.iconUrl ? (
                                                    <img
                                                        src={item.Attributes.iconUrl}
                                                        alt={item.Attributes.name}
                                                        className="h-10 w-10 rounded object-cover"
                                                        loading="lazy"
                                                    />
                                                ) : (
                                                    <div className="h-10 w-10 rounded bg-muted" />
                                                )}
                                            </TableCell>
                                            <TableCell className="font-medium">{item.Attributes.name}</TableCell>
                                            <TableCell className="font-mono text-xs">{item.Attributes.id}</TableCell>
                                            <TableCell>{item.Attributes.region}</TableCell>
                                            <TableCell>
                                                <div className="flex flex-wrap gap-1">
                                                    {item.missing_dlc?.map((dlcId) => (
                                                        <Badge key={dlcId} variant="secondary" className="text-xs font-mono">
                                                            {dlcId}
                                                        </Badge>
                                                    ))}
                                                </div>
                                            </TableCell>
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
