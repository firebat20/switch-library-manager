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
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { AlertCircle } from "lucide-react";

export default function Issues() {
    const [issues, setIssues] = useState<main.Pair[]>([]);
    const [loading, setLoading] = useState(false);

    const loadIssues = async () => {
        setLoading(true);
        try {
            const data = await UpdateLocalLibrary("");
            setIssues(data?.issues || []);
        } catch (err) {
            console.error("Failed to load issues", err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadIssues();
    }, []);

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h2 className="text-2xl font-bold tracking-tight">Issues</h2>
                <Button onClick={loadIssues} disabled={loading}>
                    {loading ? "Refreshing..." : "Refresh"}
                </Button>
            </div>

            <Card>
                <CardHeader className="pb-3">
                    <CardTitle className="flex items-center gap-2">
                        <AlertCircle className="h-5 w-5 text-destructive" />
                        Detected Issues ({issues.length})
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="rounded-md border">
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead className="w-[200px]">File/Key</TableHead>
                                    <TableHead>Issue Description</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {issues.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={2} className="h-24 text-center">
                                            No issues found.
                                        </TableCell>
                                    </TableRow>
                                ) : (
                                    issues.map((issue, index) => (
                                        <TableRow key={index}>
                                            <TableCell className="font-mono text-xs break-all">
                                                {issue.key}
                                            </TableCell>
                                            <TableCell>{issue.value}</TableCell>
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
