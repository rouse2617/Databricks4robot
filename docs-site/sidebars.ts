import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docsSidebar: [
    {
      type: 'category',
      label: 'Getting Started',
      collapsed: false,
      items: [
        'intro',
        'getting-started/quickstart',
        'getting-started/authentication',
        'getting-started/sdk-installation',
      ],
    },
    {
      type: 'category',
      label: 'Core Concepts',
      collapsed: false,
      items: [
        'core-concepts/assets',
        'core-concepts/pipelines',
        'core-concepts/deliveries',
        'core-concepts/tags',
        'core-concepts/data-model',
      ],
    },
    {
      type: 'category',
      label: 'Guides',
      collapsed: false,
      items: [
        'guides/asset-management',
        'guides/pipeline-operations',
        'guides/delivery-management',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      collapsed: false,
      items: [
        'api/overview',
      ],
    },
    'changelog',
  ],
};

export default sidebars;
