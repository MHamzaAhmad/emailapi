import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const sidebar: SidebarsConfig = {
  apisidebar: [
    {
      type: "doc",
      id: "api/v-1-email-proto",
    },
    {
      type: "category",
      label: "EmailService",
      items: [
        {
          type: "doc",
          id: "api/email-service-list-emails",
          label: "ListEmails retrieves a list of emails.",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/email-service-get-email",
          label: "GetEmail retrieves an email by ID.",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/email-service-send-email",
          label: "SendEmail queues an email for sending.",
          className: "api-method post",
        },
      ],
    },
    {
      type: "category",
      label: "UserService",
      items: [
        {
          type: "doc",
          id: "api/user-service-create-user",
          label: "CreateUser creates a new user and returns an API key.",
          className: "api-method post",
        },
        {
          type: "doc",
          id: "api/user-service-get-current-user",
          label: "GetCurrentUser retrieves the authenticated user.",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/user-service-regenerate-api-key",
          label: "RegenerateAPIKey generates a new API key for the user.",
          className: "api-method post",
        },
      ],
    },
    {
      type: "category",
      label: "WebhookService",
      items: [
        {
          type: "doc",
          id: "api/webhook-service-list-webhooks",
          label: "ListWebhooks retrieves all webhooks.",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/webhook-service-create-webhook",
          label: "CreateWebhook creates a new webhook.",
          className: "api-method post",
        },
        {
          type: "doc",
          id: "api/webhook-service-get-webhook",
          label: "GetWebhook retrieves a webhook by ID.",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/webhook-service-delete-webhook",
          label: "DeleteWebhook deletes a webhook.",
          className: "api-method delete",
        },
        {
          type: "doc",
          id: "api/webhook-service-update-webhook",
          label: "UpdateWebhook updates a webhook.",
          className: "api-method patch",
        },
      ],
    },
  ],
};

export default sidebar.apisidebar;
