import type { SidebarsConfig } from '@docusaurus/plugin-content-docs'

const sidebars: SidebarsConfig = {
  docs: [
    'intro',
    {
      type: 'category',
      label: 'SDKs',
      items: ['sdks/typescript', 'sdks/go'],
    },
  ],
  api: [
    {
      type: 'category',
      label: 'API Reference',
      link: {
        type: 'generated-index',
        title: 'Email API Reference',
        description: 'Auto-generated API reference from Protocol Buffers',
        slug: '/api',
      },
      items: require('./docs/api/sidebar.ts').default,
    },
  ],
}

export default sidebars
