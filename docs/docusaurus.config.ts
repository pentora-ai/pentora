import { themes as prismThemes } from 'prism-react-renderer'
import type { Config } from '@docusaurus/types'
import type * as Preset from '@docusaurus/preset-classic'

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

const config: Config = {
  title: 'Pentora',
  tagline: 'Modular, High-Performance Security Scanner',
  favicon: 'img/favicon.ico',

  // Future flags, see https://docusaurus.io/docs/api/docusaurus-config#future
  future: {
    v4: true, // Improve compatibility with the upcoming Docusaurus v4
  },

  // Set the production url of your site here
  url: 'https://docs.pentora.ai',
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: '/',

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: 'pentora', // Usually your GitHub org/user name.
  projectName: 'docs', // Usually your repo name.

  onBrokenLinks: 'throw',

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          routeBasePath: '/',
          sidebarPath: './sidebars.ts',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  markdown: {
    mermaid: true,
  },

  themes: ['@docusaurus/theme-mermaid'],

  themeConfig: {
    // Replace with your project's social card
    image: 'img/docusaurus-social-card.jpg',
    colorMode: {
      respectPrefersColorScheme: true,
    },
    // Algolia DocSearch - Command Palette Style
    algolia: {
      appId: 'BH4D9OD16A',
      apiKey: 'your-api-key-here',
      indexName: 'pentora',
      contextualSearch: true,
      searchPagePath: false,
    },
    navbar: {
      title: 'Pentora',
      logo: {
        alt: 'Pentora Logo',
        src: 'img/pentora-fill-color-30.svg',
        srcDark: 'img/logo-dark.svg',
        href: 'https://pentora.ai/',
        target: '_self',
      },
      items: [
        {
          to: '/',
          label: 'Docs',
          position: 'left',
          className: 'navbar__item navbar__link docs__link',
        },
        {
          to: 'https://pentora.ai/blog',
          label: 'Blog',
          position: 'right',
          target: '_self',
        },
        {
          href: 'https://github.com/pentora-ai/pentora',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    /*footer: {
      style: 'dark',
      links: [
        {
          title: 'Documentation',
          items: [
            {
              label: 'Getting Started',
              to: '/intro',
            },
            {
              label: 'CLI Reference',
              to: '/docs/cli/overview',
            },
            {
              label: 'Architecture',
              to: '/docs/architecture/overview',
            },
          ],
        },
        {
          title: 'Community',
          items: [
            {
              label: 'GitHub Issues',
              href: 'https://github.com/pentora-ai/pentora/issues',
            },
            {
              label: 'Discussions',
              href: 'https://github.com/pentora-ai/pentora/discussions',
            },
          ],
        },
        {
          title: 'More',
          items: [
            {
              label: 'Blog',
              to: '/blog',
            },
            {
              label: 'GitHub',
              href: 'https://github.com/pentora-ai/pentora',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Pentora Project. Open Source Security Scanner.`,
    },*/
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,
}

export default config
