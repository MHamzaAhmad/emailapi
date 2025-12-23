export interface ActivityLog {
    id: string;
    user_id: string;
    entity_type: string;
    entity_id: string;
    action: string;
    status: string;
    details: string;
    metadata: string; // JSON string
    timestamp: string; // ISO string
}

export interface ActivityFilters {
    entity_type?: string;
    action?: string;
    start_time?: number; // Unix ms
    end_time?: number;   // Unix ms
}

export interface ListActivityLogsRequest extends ActivityFilters {
    page_size?: number;
    offset?: number;
}

export interface ListActivityLogsResponse {
    logs: ActivityLog[];
    total_count: number;
}
