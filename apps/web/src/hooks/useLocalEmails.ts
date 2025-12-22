import { useState, useEffect } from 'react';

export interface LocalEmail {
    id: string;
    messageId?: string;
    from: string;
    to: string[];
    subject: string;
    body?: string;
    html?: string;
    status: string; // Using string to handle enum or custom values
    createdAt: string;
}

const STORAGE_KEY = 'test-emails-local-store';

export const useLocalEmails = () => {
    const [emails, setEmails] = useState<LocalEmail[]>([]);

    // Load from storage on mount
    useEffect(() => {
        const stored = localStorage.getItem(STORAGE_KEY);
        if (stored) {
            try {
                setEmails(JSON.parse(stored));
            } catch (e) {
                console.error('Failed to parse local emails', e);
            }
        }
    }, []);

    // Save to storage whenever emails change
    useEffect(() => {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(emails));
    }, [emails]);

    const addEmail = (email: Omit<LocalEmail, 'createdAt'>) => {
        const newEmail = {
            ...email,
            createdAt: new Date().toISOString(),
        };
        setEmails((prev) => [newEmail, ...prev]);
    };

    const clearEmails = () => {
        setEmails([]);
        localStorage.removeItem(STORAGE_KEY);
    };

    return {
        emails,
        addEmail,
        clearEmails,
    };
};
