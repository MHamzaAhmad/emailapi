import { createFileRoute } from '@tanstack/react-router';
import { ApiReferenceReact } from '@scalar/api-reference-react';
import '@scalar/api-reference-react/style.css';

export const Route = createFileRoute('/docs/api-reference')({
    component: ApiReferencePage,
});

function ApiReferencePage() {
    return (
        <div className="min-h-screen">
            <ApiReferenceReact
                configuration={{
                    url: '/api/openapi.json',
                    darkMode: true,
                    hideModels: false,
                    hideDownloadButton: false,
                    defaultHttpClient: {
                        targetKey: 'js',
                        clientKey: 'fetch',
                    },
                    theme: 'purple',
                }}
            />
        </div>
    );
}
