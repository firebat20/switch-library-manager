import { useState } from "react";
import { OrganizeLibrary, UpdateDB } from "@/wailsjs/go/main/App";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Terminal } from "lucide-react";

export default function Organize() {
    const [loading, setLoading] = useState(false);
    const [message, setMessage] = useState("");

    const handleOrganize = async () => {
        setLoading(true);
        setMessage("Organizing library... This may take a while.");
        try {
            await OrganizeLibrary();
            setMessage("Library organization completed.");
        } catch (err) {
            console.error("Failed to organize library", err);
            setMessage("Failed to organize library.");
        } finally {
            setLoading(false);
        }
    };

    const handleUpdateDB = async () => {
        setLoading(true);
        setMessage("Updating database...");
        try {
            await UpdateDB();
            setMessage("Database updated.");
        } catch (err) {
            console.error("Failed to update DB", err);
            setMessage("Failed to update database.");
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="space-y-4">
            <h2 className="text-2xl font-bold tracking-tight">Organize Library</h2>

            <div className="grid gap-4 md:grid-cols-2">
                <Card>
                    <CardHeader>
                        <CardTitle>Organize Files</CardTitle>
                        <CardDescription>
                            Move and rename files based on the specified pattern in settings.
                        </CardDescription>
                    </CardHeader>
                    <CardContent>
                        <Button onClick={handleOrganize} disabled={loading}>
                            {loading ? "Processing..." : "Start Organization"}
                        </Button>
                    </CardContent>
                </Card>

                <Card>
                    <CardHeader>
                        <CardTitle>Update Database</CardTitle>
                        <CardDescription>
                            Fetch the latest title database from the internet.
                        </CardDescription>
                    </CardHeader>
                    <CardContent>
                        <Button onClick={handleUpdateDB} disabled={loading} variant="secondary">
                            Update DB
                        </Button>
                    </CardContent>
                </Card>
            </div>

            {message && (
                <Alert>
                    <Terminal className="h-4 w-4" />
                    <AlertTitle>Status</AlertTitle>
                    <AlertDescription>{message}</AlertDescription>
                </Alert>
            )}
        </div>
    );
}
