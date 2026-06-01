import {readFileSync} from 'node:fs';
import path from 'node:path';
import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const openApiSpec = readFileSync(
  path.resolve(process.cwd(), '../api/openapi.yaml'),
  'utf8',
);

const config: Config = {
  title: 'Cyber Databrew',
  tagline: '数据流水线平台 — 资产、算法、交付',
  favicon: 'img/favicon.svg',

  trailingSlash: false,

  future: {
    v4: true,
  },

  url: 'https://cyber-databrew.cyberorigin.ai',
  baseUrl: '/doc/',

  scripts: [
    {
      src: '/doc/js/api-reference-default.js',
      async: false,
    },
  ],

  organizationName: 'CyberOrigin2077',
  projectName: 'cyber-databrew',

  onBrokenLinks: 'warn',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'zh-Hans',
    locales: ['zh-Hans'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: '/',
          editUrl:
            'https://github.com/CyberOrigin2077/cyber-databrew/tree/main/docs-site/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  plugins: [
    [
      '@scalar/docusaurus',
      {
        label: 'API Reference',
        route: '/api/reference',
        showNavLink: false,
        configuration: {
          title: 'Cyber Databrew API',
          content: openApiSpec,
          layout: 'modern',
          theme: 'default',
          defaultHttpClient: {
            targetKey: 'shell',
            clientKey: 'curl', // pragma: allowlist secret
          },
          defaultOpenFirstTag: true,
          documentDownloadType: 'none',
          expandAllResponses: false,
          hideModels: true,
          hideClientButton: false,
          persistAuth: true,
          showSidebar: true,
          showDeveloperTools: 'never',
          servers: [
            {
              url: 'http://localhost:8080',
              description: 'Local development',
            },
          ],
        },
      },
    ],
  ],

  themeConfig: {
    image: 'img/docusaurus-social-card.jpg',
    colorMode: {
      defaultMode: 'light',
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'Cyber Databrew',
      logo: {
        alt: 'Cyber Databrew',
        src: 'img/favicon.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Guides',
        },
        {
          to: '/guides/asset-management',
          position: 'left',
          label: 'Recipes',
        },
        {
          to: '/api/reference',
          position: 'left',
          label: 'REST API Reference',
        },
        {
          to: '/getting-started/sdk-installation',
          position: 'left',
          label: 'Python API Reference',
        },
        {
          to: '/changelog',
          position: 'left',
          label: 'Changelog',
        },
        {
          href: 'https://github.com/CyberOrigin2077/cyber-databrew',
          position: 'right',
          className: 'header-github-link',
          'aria-label': 'GitHub repository',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [],
      copyright: `Copyright © ${new Date().getFullYear()} CyberOrigin. All rights reserved.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['python', 'bash', 'yaml', 'json'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
