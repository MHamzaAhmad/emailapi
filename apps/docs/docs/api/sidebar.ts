import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const sidebar: SidebarsConfig = {
  apisidebar: [
    {
      type: "doc",
      id: "api/v-1-domain-proto",
    },
    {
      type: "category",
      label: "DomainService",
      items: [
        {
          type: "doc",
          id: "api/domain-service-list-domains",
          label: "ListDomains retrieves all domains for the authenticated user.",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/domain-service-add-domain",
          label: "AddDomain registers a new sending domain with AWS SES.",
          className: "api-method post",
        },
        {
          type: "doc",
          id: "api/domain-service-get-domain",
          label: "GetDomain retrieves a domain by its ID.",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/domain-service-delete-domain",
          label: "DeleteDomain removes a domain from your account and AWS SES.",
          className: "api-method delete",
        },
        {
          type: "doc",
          id: "api/domain-service-set-mail-from-domain",
          label: "SetMailFromDomain configures a custom MAIL FROM subdomain.",
          className: "api-method post",
        },
        {
          type: "doc",
          id: "api/domain-service-get-domain-records",
          label: "GetDomainRecords returns all DNS records needed for full email deliverability.",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/domain-service-verify-domain",
          label: "VerifyDomain refreshes verification status from AWS SES.",
          className: "api-method post",
        },
      ],
    },
    {
      type: "category",
      label: "EmailService",
      items: [
        {
          type: "doc",
          id: "api/email-service-list-emails",
          label: "ListEmails retrieves a paginated list of emails sent by the authenticated user.",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/email-service-get-email",
          label: "GetEmail retrieves the details of a specific email by its ID.",
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
