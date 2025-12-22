import { useEffect, useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { AppPortal } from 'svix-react';
import 'svix-react/style.css';
import { webhookService } from '@/services';
import { useCurrentUser } from '@/hooks';

export const Route = createFileRoute('/webhooks')({
    component: WebhooksPage,
});

function WebhooksPage() {
    const { data: user } = useCurrentUser();
    const isAuthenticated = !!user;
    const [url, setUrl] = useState<string | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (isAuthenticated) {
            setLoading(true);
            webhookService.getAppPortalAccess()
                .then((data) => {
                    setUrl(data.url);
                })
                .catch((err) => {
                    console.error('Failed to get App Portal access:', err);
                    setError('Failed to load Webhooks Portal. Please try again.');
                })
                .finally(() => {
                    setLoading(false);
                });
        }
    }, [isAuthenticated]);

    if (!isAuthenticated) {
        return (
            <div className="p-8 text-center text-muted-foreground">
                Please log in to manage webhooks.
            </div>
        );
    }

    if (loading) {
        return (
            <div className="p-8 text-center text-muted-foreground">
                Loading Webhooks Portal...
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-8 text-center text-red-500">
                {error}
            </div>
        );
    }

    if (!url) {
        return null;
    }

    return (
        <div className="h-[calc(100vh-4rem)] w-full">
            <AppPortal
                url={url}
                fullSize={true}
            />
        </div>
    );
}

export default WebhooksPage;
